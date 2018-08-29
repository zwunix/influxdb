package storage

import (
	"context"
	"errors"

	"github.com/influxdata/influxdb/models"
	"github.com/influxdata/influxdb/query"
	"github.com/influxdata/influxdb/tsdb"
	"github.com/influxdata/influxql"
	"github.com/opentracing/opentracing-go"
)

const (
	measurementKey = "_measurement"
	fieldKey       = "_field"
)

var (
	measurementKeyBytes = []byte(measurementKey)
	fieldKeyBytes       = []byte(fieldKey)
)

type SeriesIndex interface {
	CreateCursor(ctx context.Context, req *ReadRequest, shards []*tsdb.Shard)
}

type SeriesCursor interface {
	Close()
	Next() *SeriesRow
	Err() error
}

type SeriesRow struct {
	SortKey    []byte
	Name       []byte      // measurement name
	SeriesTags models.Tags // unmodified series tags
	Tags       models.Tags
	Field      string
	Query      tsdb.CursorIterators
	ValueCond  influxql.Expr
}

type indexSeriesCursor struct {
	sqry            tsdb.SeriesCursor
	field           field
	err             error
	tags            models.Tags
	cond            influxql.Expr
	measurementCond influxql.Expr
	row             SeriesRow
	eof             bool
	hasFieldExpr    bool
	hasValueExpr    bool
}

func newIndexSeriesCursor(ctx context.Context, predicate *Predicate, shards []*tsdb.Shard) (*indexSeriesCursor, error) {
	queries, err := tsdb.CreateCursorIterators(ctx, shards)
	if err != nil {
		return nil, err
	}

	if queries == nil {
		return nil, nil
	}

	span := opentracing.SpanFromContext(ctx)
	if span != nil {
		span = opentracing.StartSpan("index_cursor.create", opentracing.ChildOf(span.Context()))
		defer span.Finish()
	}

	opt := query.IteratorOptions{
		Aux:        []influxql.VarRef{{Val: "key"}},
		Authorizer: query.OpenAuthorizer,
		Ascending:  true,
		Ordered:    true,
	}
	p := &indexSeriesCursor{row: SeriesRow{Query: queries}}

	if root := predicate.GetRoot(); root != nil {
		if p.cond, err = NodeToExpr(root, measurementRemap); err != nil {
			return nil, err
		}

		p.hasFieldExpr, p.hasValueExpr = HasFieldKeyOrValue(p.cond)
		if !(p.hasFieldExpr || p.hasValueExpr) {
			p.measurementCond = p.cond
			opt.Condition = p.cond
		} else {
			p.measurementCond = influxql.Reduce(RewriteExprRemoveFieldValue(influxql.CloneExpr(p.cond)), nil)
			if isTrueBooleanLiteral(p.measurementCond) {
				p.measurementCond = nil
			}

			opt.Condition = influxql.Reduce(RewriteExprRemoveFieldKeyAndValue(influxql.CloneExpr(p.cond)), nil)
			if isTrueBooleanLiteral(opt.Condition) {
				opt.Condition = nil
			}
		}
	}

	// TODO(jeff): HOLY MOLY! instead we assume _f exists in the tags and use that to figure out
	// the field name. which means we don't have to query for all the fields for all the
	// measurements or whatever. IS THIS OK!?

	sg := tsdb.Shards(shards)
	p.sqry, err = sg.CreateSeriesCursor(ctx, tsdb.SeriesCursorRequest{}, opt.Condition)
	if err != nil {
		p.Close()
		return nil, err
	}
	return p, nil
}

func (c *indexSeriesCursor) Close() {
	if !c.eof {
		c.eof = true
		if c.sqry != nil {
			c.sqry.Close()
			c.sqry = nil
		}
	}
}

func copyTags(dst, src models.Tags) models.Tags {
	if cap(dst) < src.Len() {
		dst = make(models.Tags, src.Len())
	} else {
		dst = dst[:src.Len()]
	}
	copy(dst, src)
	return dst
}

func (c *indexSeriesCursor) Next() *SeriesRow {
	if c.eof {
		return nil
	}

	for {
		if c.measurementCond == nil || evalExprBool(c.measurementCond, c) {
			break
		}

		// next series key
		sr, err := c.sqry.Next()
		if err != nil {
			c.err = err
			c.Close()
			return nil
		} else if sr == nil {
			c.Close()
			return nil
		}

		c.row.Name = sr.Name
		c.row.SeriesTags = sr.Tags
		c.tags = copyTags(c.tags, sr.Tags)
		c.tags.Set(measurementKeyBytes, sr.Name)
		nb := c.tags.Get(fieldKeyBytes)
		c.field = field{
			nb: nb,
			n:  string(nb),
		}
	}

	c.row.Field = c.field.n

	if c.cond != nil && c.hasValueExpr {
		// TODO(sgc): lazily evaluate valueCond
		c.row.ValueCond = influxql.Reduce(c.cond, c)
		if isTrueBooleanLiteral(c.row.ValueCond) {
			// we've reduced the expression to "true"
			c.row.ValueCond = nil
		}
	}

	c.row.Tags = copyTags(c.row.Tags, c.tags)

	return &c.row
}

func (c *indexSeriesCursor) Value(key string) (interface{}, bool) {
	switch key {
	case "_name":
		return c.row.Name, true
	case fieldKey:
		return c.field.n, true
	default:
		res := c.row.SeriesTags.Get([]byte(key))
		return res, res != nil
	}
}

func (c *indexSeriesCursor) Err() error {
	return c.err
}

type limitSeriesCursor struct {
	SeriesCursor
	n, o, c int64
}

func NewLimitSeriesCursor(ctx context.Context, cur SeriesCursor, n, o int64) SeriesCursor {
	return &limitSeriesCursor{SeriesCursor: cur, o: o, n: n}
}

func (c *limitSeriesCursor) Next() *SeriesRow {
	if c.o > 0 {
		for i := int64(0); i < c.o; i++ {
			if c.SeriesCursor.Next() == nil {
				break
			}
		}
		c.o = 0
	}

	if c.c >= c.n {
		return nil
	}
	c.c++
	return c.SeriesCursor.Next()
}

func isTrueBooleanLiteral(expr influxql.Expr) bool {
	b, ok := expr.(*influxql.BooleanLiteral)
	if ok {
		return b.Val
	}
	return false
}

func toFloatIterator(iter query.Iterator) (query.FloatIterator, error) {
	sitr, ok := iter.(query.FloatIterator)
	if !ok {
		return nil, errors.New("expected FloatIterator")
	}

	return sitr, nil
}

type field struct {
	n  string
	nb []byte
}
