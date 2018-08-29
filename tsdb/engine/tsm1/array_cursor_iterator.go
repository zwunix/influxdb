package tsm1

import (
	"context"
	"fmt"

	"github.com/influxdata/influxdb/models"
	"github.com/influxdata/influxdb/pkg/metrics"
	"github.com/influxdata/influxdb/query"
	"github.com/influxdata/influxdb/tsdb"
	"github.com/influxdata/influxql"
)

type arrayCursorIterator struct {
	e   *Engine
	key []byte

	asc struct {
		Float    *floatArrayAscendingCursor
		Integer  *integerArrayAscendingCursor
		Unsigned *unsignedArrayAscendingCursor
		Boolean  *booleanArrayAscendingCursor
		String   *stringArrayAscendingCursor
	}

	desc struct {
		Float    *floatArrayDescendingCursor
		Integer  *integerArrayDescendingCursor
		Unsigned *unsignedArrayDescendingCursor
		Boolean  *booleanArrayDescendingCursor
		String   *stringArrayDescendingCursor
	}
}

func (q *arrayCursorIterator) Next(ctx context.Context, r *tsdb.CursorRequest) (tsdb.Cursor, error) {
	// TODO(jeff): use the series file off of the engine to look up the type
	return nil, nil

	if grp := metrics.GroupFromContext(ctx); grp != nil {
		grp.GetCounter(numberOfRefCursorsCounter).Add(1)
	}

	var opt query.IteratorOptions
	opt.Ascending = r.Ascending
	opt.StartTime = r.StartTime
	opt.EndTime = r.EndTime

	// Return appropriate cursor based on type.
	switch typ := influxql.Float; typ {
	case influxql.Float:
		return q.buildFloatArrayCursor(ctx, r.Name, r.Tags, r.Field, opt), nil
	case influxql.Integer:
		return q.buildIntegerArrayCursor(ctx, r.Name, r.Tags, r.Field, opt), nil
	case influxql.Unsigned:
		return q.buildUnsignedArrayCursor(ctx, r.Name, r.Tags, r.Field, opt), nil
	case influxql.String:
		return q.buildStringArrayCursor(ctx, r.Name, r.Tags, r.Field, opt), nil
	case influxql.Boolean:
		return q.buildBooleanArrayCursor(ctx, r.Name, r.Tags, r.Field, opt), nil
	default:
		panic(fmt.Sprintf("unreachable: %T", typ))
	}
}

func (q *arrayCursorIterator) seriesFieldKeyBytes(name []byte, tags models.Tags, field string) []byte {
	q.key = models.AppendMakeKey(q.key[:0], name, tags)
	q.key = append(q.key, keyFieldSeparatorBytes...)
	q.key = append(q.key, field...)
	return q.key
}
