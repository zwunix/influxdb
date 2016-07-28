package tsm1

import (
	"reflect"
	//"os"
	//"strconv"
	"math/rand"
	"sync"
	//"time"
	"unsafe"

	"github.com/couchbase/go-slab" // slab
)

var sizeOfSliceHeader uintptr = unsafe.Sizeof(reflect.SliceHeader{})

type ByteSliceSlabPool struct {
	arena *slab.Arena
	sync.Mutex
	refs int64
}

func NewByteSliceSlabPool() *ByteSliceSlabPool {
	return &ByteSliceSlabPool{
		arena: slab.NewArena(1, 1024*1024, 2, nil),
		Mutex: sync.Mutex{},
		refs:  0,
	}
}

func (p *ByteSliceSlabPool) Get(l int) []byte {
	if l == 0 {
		return nil
	}
	p.Lock()
	x := p.arena.Alloc(l)
	p.refs++
	p.Unlock()
	return x
}
func (p *ByteSliceSlabPool) Inc(x []byte) {
	p.Lock()
	p.arena.AddRef(x)
	p.refs++
	p.Unlock()
}
func (p *ByteSliceSlabPool) Dec(x []byte) bool {
	p.Lock()
	ret := p.arena.DecRef(x)
	p.refs--
	p.Unlock()
	return ret
}

func (p *ByteSliceSlabPool) Refs() int64 {
	p.Lock()
	ret := p.refs
	p.Unlock()
	return ret
}

type ShardedByteSliceSlabPool struct {
	nshards int
	pools []*ByteSliceSlabPool
}

func NewShardedByteSliceSlabPool(nshards int) *ShardedByteSliceSlabPool {
	pools := make([]*ByteSliceSlabPool, nshards)
	for i := range pools {
		pools[i] = NewByteSliceSlabPool()
	}
	return &ShardedByteSliceSlabPool{
		nshards: nshards,
		pools: pools,
	}
}

func (p *ShardedByteSliceSlabPool) Get(l int) []byte {
	shardId := rand.Intn(p.nshards)
	pool := p.pools[shardId]

	l2 := 8 + l
	buf := pool.Get(l2)

	danglingMetadata := *(*reflect.SliceHeader)(unsafe.Pointer(&buf))

	shardIdDst := (*uint64)(unsafe.Pointer(&(buf[0])))
	*shardIdDst = uint64(shardId)

	publicBuf := reflect.SliceHeader{
		Data: danglingMetadata.Data + 8,
		Len: danglingMetadata.Len - 8,
		Cap: danglingMetadata.Cap,
	}

	ret := *(*[]byte)(unsafe.Pointer(&publicBuf))
	return ret
}
func (p *ShardedByteSliceSlabPool) Inc(x []byte) {
	privateBuf, shardId := p.parsePrivateBuf(x)
	pool := p.pools[shardId]
	pool.Inc(buf)
}
func (p *ShardedByteSliceSlabPool) parsePrivateBuf(x []byte) ([]byte, uint64) {
	publicMetadataHeader := *(*reflect.SliceHeader)(unsafe.Pointer(&x))
	privateMetadataHeader := reflect.SliceHeader{
		Data: publicMetadataHeader.Data - 8,
		Len: publicMetadataHeader.Len + 8,
		Cap: publicMetadataHeader.Cap,
	}

	shardId := *(*uint64)(unsafe.Pointer(privateMetadataHeader.Data))

	buf := *(*[]byte)(unsafe.Pointer(&privateMetadataHeader))

	return buf, shardId
}
func (p *ShardedByteSliceSlabPool) Dec(x []byte) bool {
	privateBuf, shardId := p.parsePrivateBuf(x)
	pool := p.pools[shardId]
	return pool.Dec(buf)
}

type StringSlabPool struct {
	ByteSliceSlabPool
}

func NewStringSlabPool() *StringSlabPool{
	return &StringSlabPool{
		ByteSliceSlabPool: *NewByteSliceSlabPool(),
	}
}

func (p *StringSlabPool) Get(l int) (string, []byte) {
	l2 := int(sizeOfSliceHeader) + l
	buf := p.ByteSliceSlabPool.Get(l2)

	// we have to serialize this because it will not be returned to
	// the caller:
	danglingMetadata := *(*reflect.SliceHeader)(unsafe.Pointer(&buf))

	metadata := (*reflect.SliceHeader)(unsafe.Pointer(&(buf[0])))
	metadata.Data = danglingMetadata.Data
	metadata.Len = danglingMetadata.Len
	metadata.Cap = danglingMetadata.Cap

	publicStr := reflect.StringHeader{
		Data: metadata.Data + sizeOfSliceHeader,
		Len: l,
	}
	publicBuf := reflect.SliceHeader{
		Data: metadata.Data + sizeOfSliceHeader,
		Len: l,
		Cap: l,
	}

	retA := *(*string)(unsafe.Pointer(&publicStr))
	retB := *(*[]byte)(unsafe.Pointer(&publicBuf))

	return retA, retB
}

func (p *StringSlabPool) Inc(s string) {
	publicMetadata := *(*reflect.StringHeader)(unsafe.Pointer(&s))

	metadata := *(*reflect.SliceHeader)(unsafe.Pointer(publicMetadata.Data - uintptr(sizeOfSliceHeader)))

	metadataBuf := *(*[]byte)(unsafe.Pointer(&metadata))

	p.ByteSliceSlabPool.Inc(metadataBuf)
}
func (p *StringSlabPool) Dec(s string) bool {
	publicMetadata := *(*reflect.StringHeader)(unsafe.Pointer(&s))

	metadata := *(*reflect.SliceHeader)(unsafe.Pointer(publicMetadata.Data - uintptr(sizeOfSliceHeader)))

	metadataBuf := *(*[]byte)(unsafe.Pointer(&metadata))

	return p.ByteSliceSlabPool.Dec(metadataBuf)
}
