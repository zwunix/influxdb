package tsm1

import (
	"github.com/influxdata/influxdb/tsdb"
)

// TimeRangeIterator will iterate over the keys of a TSM file, starting at
// the provided key. It is used to determine if each key has data which exists
// within a specified time interval.
type TimeRangeIterator struct {
	r    *TSMReader
	iter *TSMIndexIterator
	tr   TimeRange
	err  error

	// temporary storage
	trbuf []TimeRange
	buf   []byte
	a     tsdb.TimestampArray
}

func (b *TimeRangeIterator) Err() error {
	if b.err != nil {
		return b.err
	}
	return b.iter.Err()
}

// Next advances the iterator and reports if it is still valid.
func (b *TimeRangeIterator) Next() bool {
	if b.Err() != nil {
		return false
	}

	return b.iter.Next()
}

// Key reports the current key.
func (b *TimeRangeIterator) Key() []byte {
	return b.iter.Key()
}

// HasData reports true if the current key has data for the time range.
func (b *TimeRangeIterator) HasData() bool {
	if b.Err() != nil {
		return false
	}

	e := excludeEntries(b.iter.Entries(), b.tr)
	if len(e) == 0 {
		return false
	}

	b.trbuf = b.r.TombstoneRange(b.iter.Key(), b.trbuf[:0])
	if len(b.trbuf) == 0 {
		// no tombstones, fast path will avoid decoding blocks
		// if queried time interval intersects with one of the entries
		if intersectsEntry(e, b.tr) {
			return true
		}

		for i := range e {
			_, b.buf, b.err = b.r.ReadBytes(&e[i], b.buf)
			if b.err != nil {
				return false
			}

			b.err = DecodeTimestampArrayBlock(b.buf, &b.a)
			if b.err != nil {
				return false
			}

			if b.a.Contains(b.tr.Min, b.tr.Max) {
				return true
			}
		}
	} else {
		for i := range e {
			_, b.buf, b.err = b.r.ReadBytes(&e[i], b.buf)
			if b.err != nil {
				return false
			}

			b.err = DecodeTimestampArrayBlock(b.buf, &b.a)
			if b.err != nil {
				return false
			}

			// remove tombstoned timestamps
			for i := range b.trbuf {
				b.a.Exclude(b.trbuf[i].Min, b.trbuf[i].Max)
			}

			if b.a.Contains(b.tr.Min, b.tr.Max) {
				return true
			}
		}
	}

	return false
}

/*
intersectsEntry determines whether the range [min, max]
intersects one or both boundaries of IndexEntry.

          +------------------+
          |    IndexEntry    |
+---------+------------------+---------+
|  RANGE  |                  |  RANGE  |
+-+-------+-+           +----+----+----+
  |  RANGE  |           |  RANGE  |
  +----+----+-----------+---------+
       |          RANGE           |
       +--------------------------+
*/

// intersectsEntry determines if tr overlaps one or both boundaries
// of at least one element of e. If that is the case,
// and the block has no tombstones, the block timestamps do not
// need to be decoded.
func intersectsEntry(e []IndexEntry, tr TimeRange) bool {
	for i := range e {
		min, max := e[i].MinTime, e[i].MaxTime
		if tr.Overlaps(min, max) && !tr.Within(min, max) {
			return true
		}
	}
	return false
}

// excludeEntries returns a slice which excludes leading and trailing
// elements of e that are outside the time range specified by tr.
func excludeEntries(e []IndexEntry, tr TimeRange) []IndexEntry {
	for i := range e {
		if e[i].OverlapsTimeRange(tr.Min, tr.Max) {
			e = e[i:]
			break
		}
	}

	for i := range e {
		if !e[i].OverlapsTimeRange(tr.Min, tr.Max) {
			e = e[:i]
			break
		}
	}

	return e
}
