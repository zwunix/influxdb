package storage

import (
	"context"

	"github.com/influxdata/influxdb"
	"github.com/influxdata/influxdb/tsdb"
	"github.com/influxdata/influxql"
)

// CreateSeriesCursor creates a SeriesCursor for usage with the read service.
func (e *Engine) CreateSeriesCursor(ctx context.Context, req SeriesCursorRequest, cond influxql.Expr) (SeriesCursor, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.closing == nil {
		return nil, ErrEngineClosed
	}

	return newSeriesCursor(req, e.index, e.sfile, cond)
}

func (e *Engine) CreateTagKeysIterator(ctx context.Context, name [influxdb.IDLength]byte, startTime, endTime int64, cond influxql.Expr) tsdb.TagKeyIterator {
	itr, _ := e.index.TagKeyIterator(name[:])
	return itr
}

func (e *Engine) CreateTagValuesIterator(ctx context.Context, name [influxdb.IDLength]byte, tagKey string, startTime, endTime int64, cond influxql.Expr) tsdb.TagValueIterator {
	return nil
}
