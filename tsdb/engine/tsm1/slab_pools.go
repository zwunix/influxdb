package tsm1

import (
	"encoding/binary"
	"bytes"
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
	*sync.Mutex
	refs int64
}

func NewByteSliceSlabPool() *ByteSliceSlabPool {
	return &ByteSliceSlabPool{
		arena: slab.NewArena(1, 1024*1024, 2, nil),
		Mutex: &sync.Mutex{},
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
	pools   []*ByteSliceSlabPool
}

func NewShardedByteSliceSlabPool(nshards int) *ShardedByteSliceSlabPool {
	pools := make([]*ByteSliceSlabPool, nshards)
	for i := range pools {
		pools[i] = NewByteSliceSlabPool()
	}
	return &ShardedByteSliceSlabPool{
		nshards: nshards,
		pools:   pools,
	}
}

func (p *ShardedByteSliceSlabPool) ApproximateRefs() int64 {
	var n int64
	for i := 0; i < p.nshards; i++ {
		n += p.pools[i].Refs()
	}
	return n
}

func (p *ShardedByteSliceSlabPool) RandShardID() int {
	return rand.Intn(p.nshards)
}

func (p *ShardedByteSliceSlabPool) Get(l, shardId int) []byte {
	if shardId < 0 {
		shardId = 0
		if p.nshards > 1 {
			shardId = rand.Intn(p.nshards)
		}
	}

	pool := p.pools[shardId]

	l2 := 8 + l
	buf := pool.Get(l2)

	danglingMetadata := *(*reflect.SliceHeader)(unsafe.Pointer(&buf))

	binary.LittleEndian.PutUint64(buf, uint64(shardId))
	//shardIdDst := (*uint64)(unsafe.Pointer(&(buf[0])))
	//*shardIdDst = uint64(shardId)

	publicBuf := reflect.SliceHeader{
		Data: danglingMetadata.Data + 8,
		Len:  danglingMetadata.Len - 8,
		Cap:  danglingMetadata.Cap,
	}

	ret := *(*[]byte)(unsafe.Pointer(&publicBuf))

	_, got1 := p.parsePublic(ret)
	_ = bytes.Equal
	if got1 != uint64(shardId) {
		panic("bad shardId equal on sanity check")
	}
	return ret
}
func (p *ShardedByteSliceSlabPool) Inc(x []byte) {
	privateBuf, shardId := p.parsePublic(x)
	pool := p.pools[shardId]
	pool.Inc(privateBuf)
}
func (p *ShardedByteSliceSlabPool) Dec(x []byte) bool {
	privateBuf, shardId := p.parsePublic(x)
	pool := p.pools[shardId]
	return pool.Dec(privateBuf)
}
func (p *ShardedByteSliceSlabPool) parsePublic(x []byte) ([]byte, uint64) {
	publicMetadataHeader := *(*reflect.SliceHeader)(unsafe.Pointer(&x))
	privateMetadataHeader := reflect.SliceHeader{
		Data: publicMetadataHeader.Data - 8,
		Len:  publicMetadataHeader.Len + 8,
		Cap:  publicMetadataHeader.Cap,
	}

	//shardId := *(*uint64)(unsafe.Pointer(privateMetadataHeader.Data))

	buf := *(*[]byte)(unsafe.Pointer(&privateMetadataHeader))
	shardId := binary.LittleEndian.Uint64(buf)

	return buf, shardId
}

type StringSlabPool struct {
	ShardedByteSliceSlabPool
}

func NewStringSlabPool(nshards int) *StringSlabPool {
	if nshards <= 0 {
		nshards = 1
	}
	return &StringSlabPool{
		ShardedByteSliceSlabPool: *NewShardedByteSliceSlabPool(nshards),
	}
}

func (p *StringSlabPool) Get(l, shardId int) (string, []byte) {
	l2 := int(sizeOfSliceHeader) + l
	buf := p.ShardedByteSliceSlabPool.Get(l2, shardId)

	// we have to serialize this because it will not be returned to
	// the caller:
	danglingMetadata := *(*reflect.SliceHeader)(unsafe.Pointer(&buf))

	metadata := (*reflect.SliceHeader)(unsafe.Pointer(&(buf[0])))
	metadata.Data = danglingMetadata.Data
	metadata.Len = danglingMetadata.Len
	metadata.Cap = danglingMetadata.Cap

	publicStr := reflect.StringHeader{
		Data: metadata.Data + sizeOfSliceHeader,
		Len:  l,
	}
	publicBuf := reflect.SliceHeader{
		Data: metadata.Data + sizeOfSliceHeader,
		Len:  l,
		Cap:  l,
	}

	retA := *(*string)(unsafe.Pointer(&publicStr))
	retB := *(*[]byte)(unsafe.Pointer(&publicBuf))

	return retA, retB
}

func (p *StringSlabPool) Inc(s string) {
	privateBuf := p.parsePublic(s)
	p.ShardedByteSliceSlabPool.Inc(privateBuf)
}
func (p *StringSlabPool) Dec(s string) bool {
	privateBuf := p.parsePublic(s)
	return p.ShardedByteSliceSlabPool.Dec(privateBuf)
}

func (p *StringSlabPool) parsePublic(s string) []byte {
	publicMetadata := *(*reflect.StringHeader)(unsafe.Pointer(&s))

	metadata := *(*reflect.SliceHeader)(unsafe.Pointer(publicMetadata.Data - uintptr(sizeOfSliceHeader)))

	privateBuf := *(*[]byte)(unsafe.Pointer(&metadata))

	return privateBuf
}
