package tsm1

import (
	"fmt"
	"hash"
	//"hash/fnv"
	"github.com/pierrec/xxHash/xxHash32"
	"os"
	"sync"
	"strings"
	"strconv"
	"unsafe"
)

type (
	// seriesKey is a typed string that holds the identity of a series.
	// format: [measurement],[canonical tags]
	// example: cpu,hostname=host0,region=us-east
	seriesKey string

	// fieldKey is a typed string that holds the identity of a field.
	// format: [name]
	// example: usage_user
	fieldKey string
)

// CompositeKey stores namespaced strings that fully identify a series.
type CompositeKey struct {
	SeriesKey seriesKey
	FieldKey  fieldKey
}

// NewCompositeKey makes a composite key from normal strings.
func NewCompositeKey(l, v string) CompositeKey {
	return CompositeKey{
		SeriesKey: seriesKey(l),
		FieldKey:  fieldKey(v),
	}
}

// StringToCompositeKey is a convenience function that parses a CompositeKey
// out of an untyped string. It assumes that the string has the following
// format (an example):
// "measurement_name,tag0=key0" + keyFieldSeparator + "field_name"
// It is a utility to help migrate existing code to use the CacheStore.
func StringToCompositeKey(s string) CompositeKey {
	sepStart := strings.Index(s, keyFieldSeparator)
	if sepStart == -1 {
		panic("logic error: bad StringToCompositeKey input")
	}

	l := s[:sepStart]
	v := s[sepStart+len(keyFieldSeparator):]
	return NewCompositeKey(l, v)
}

// StringKey makes a plain string version of a CompositeKey. It uses the
// magic `keyFieldSeparator` value defined elsewhere in the tsm1 code.
func (ck CompositeKey) StringKey() string {
	return fmt.Sprintf("%s%s%s", ck.SeriesKey, keyFieldSeparator, ck.FieldKey)
}

var LOCKFREE_SHARDS uint32 = 16
var globalHasherPool = &sync.Pool{
	New: func() interface{} {
		return xxHash32.New(1234)
	},
}

func init() {
	if s := os.Getenv("LOCKFREE_SHARDS"); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil {
			panic(err.Error())
		}
		LOCKFREE_SHARDS = uint32(n)
	}
	println("LOCKFREE_SHARDS is %d", LOCKFREE_SHARDS)
}

// CacheStore is a sharded map used for storing series data in a *tsm1.Cache.
// It breaks away from previous tsm1.Cache designs by namespacing the keys into
// parts, using CompositeKey, which allows for less contention on the root
// map instance. Using this type is a speed improvement over the previous
// map[string]*entry type that the tsm1.Cache used.
type CacheStore struct {
	buckets []map[seriesKey]*fieldData
	avgFieldsPerSeries int64
	avgPointsPerField int64
	hasherPool *sync.Pool
}

// fieldData stores field-related data. An instance of this type makes up a
// 'shard' in a CacheStore.
type fieldData struct {
	// TODO(rw): explore using a lock to implement finer-grained
	// concurrency control.
	data map[fieldKey]*entry
}

// NewCacheStore creates a new CacheStore.
func NewCacheStoreWithCapacities(series, fields, points int64) CacheStore {
	hasherPool := globalHasherPool
	var avgSeriesPerBucket, avgFieldsPerSeries int64
	var avgPointsPerField int64
	if series > 0 {
		avgSeriesPerBucket = series / int64(LOCKFREE_SHARDS)
	}
	if series > 0 && fields > 0 {
		avgFieldsPerSeries = fields / series
}
	if series > 0 && fields > 0 && points > 0 {
		avgPointsPerField = points / fields
	}
	bb := make([]map[seriesKey]*fieldData, LOCKFREE_SHARDS)
	for i := range bb {
		bb[i] = make(map[seriesKey]*fieldData, avgSeriesPerBucket)
	}
	return CacheStore{
		buckets: bb,
		avgFieldsPerSeries: avgFieldsPerSeries,
		avgPointsPerField: avgPointsPerField,
		hasherPool: hasherPool,
	}
}

func (cs CacheStore) getHasher() hash.Hash32 {
	return cs.hasherPool.Get().(hash.Hash32)
}
func (cs CacheStore) putHasher(h hash.Hash32) {
	h.Reset()
	cs.hasherPool.Put(h)
}
func (cs CacheStore) bucketId(ck CompositeKey) uint32 {
	hasher := cs.getHasher()
	xx := *(*[]byte)(unsafe.Pointer(&ck.SeriesKey))
	hasher.Write(xx)
	n := hasher.Sum32()
	cs.putHasher(hasher)

	m := n % LOCKFREE_SHARDS

	return m
}

func (cs CacheStore) bucketFor(ck CompositeKey) map[seriesKey]*fieldData {
	return cs.buckets[cs.bucketId(ck)]
}

// Len computes the total number of elements.
func (cs CacheStore) Len() int64 {
	var n int64
	for _, b := range cs.buckets {
		for _, sub := range b {
			n += int64(len(sub.data))
		}
	}
	return n
}

// Stats computes the number of elements at each level, for use with
// NewCacheStoreWithCapacity.
func (cs CacheStore) Stats() (int64, int64, int64) {
	return 0, 0, 0
	var series int64
	var fields int64
	var points int64
	for _, b := range cs.buckets {
		series += int64(len(b))
		for _, sub := range b {
			fields += int64(len(sub.data))
			//for _, e := range sub.data {
			//	points += int64(len(e.values))
			//}
		}
	}
	return series, fields, points
}


// Get fetches the value associated with the CacheStore, if any. It is
// equivalent to the one-variable form of a Go map access.
func (cs CacheStore) Get(ck CompositeKey) *entry {
	e, ok := cs.GetChecked(ck)
	if ok {
		return e
	}
	return nil
}

// Get fetches the value associated with the CacheStore. It is equivalent to
// the two-variable form of a Go map access.
func (cs CacheStore) GetChecked(ck CompositeKey) (*entry, bool) {
	b := cs.bucketFor(ck)
	sub, ok := b[ck.SeriesKey]
	if sub == nil || !ok {
		return nil, false
	}
	e, ok2 := sub.data[ck.FieldKey]
	if e == nil || !ok2 {
		return e, false
	}
	return e, true
}

// Put puts the given value into the CacheStore.
func (cs CacheStore) Put(ck CompositeKey, e *entry) {
	b := cs.bucketFor(ck)
	sub, ok := b[ck.SeriesKey]
	if sub == nil || !ok {
		sub = &fieldData{
			data: make(map[fieldKey]*entry, cs.avgFieldsPerSeries),
		}
		b[ck.SeriesKey] = sub
	}
	sub.data[ck.FieldKey] = e
}

// Delete deletes the given key from the CacheStore, if applicable.
func (cs CacheStore) Delete(ck CompositeKey) {
	b := cs.bucketFor(ck)
	sub, ok := b[ck.SeriesKey]
	if sub == nil || !ok {
		return
	}
	delete(sub.data, ck.FieldKey)
	if len(sub.data) == 0 {
		delete(b, ck.SeriesKey)
	}
}

// Iter iterates over (key, value) pairs in the CacheStore. It takes a
// callback function that acts upon each (key, value) pair, and aborts if that
// callback returns an error. It is equivalent to the two-variable range
// statement with the normal Go map.
func (cs CacheStore) Iter(f func(CompositeKey, *entry) error) error {
	for _, bucket := range cs.buckets {
		for seriesKey, sub := range bucket {
			for fieldKey, e := range sub.data {
				ck := CompositeKey{
					SeriesKey: seriesKey,
					FieldKey:  fieldKey,
				}
				err := f(ck, e)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}
