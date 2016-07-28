package tsm1

import (
	//"fmt"
	"reflect"
	//"os"
	//"strconv"
	"sync"
	//"time"
	"unsafe"

	"github.com/couchbase/go-slab" // slab
)

var sizeOfSliceHeader = int(unsafe.Sizeof(reflect.SliceHeader{}))

type ByteSliceSlabPool struct {
	arena *slab.Arena
	sync.Mutex
	refs int64
}

func NewByteSliceSlabPool() *ByteSliceSlabPool {
	return &ByteSliceSlabPool{
		arena: slab.NewArena(1, 1024, 2, nil),
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

type StringSlabPool struct {
	ByteSliceSlabPool
}

func NewStringSlabPool() *StringSlabPool{
	return &StringSlabPool{
		ByteSliceSlabPool: *NewByteSliceSlabPool(),
	}
}

func (p *StringSlabPool) Get(l int) (string, []byte) {
	l2 := sizeOfSliceHeader + l
	buf := p.ByteSliceSlabPool.Get(l2)

	// the bytes of the sliceheader
	metadataBytes := (*(*[]byte)(unsafe.Pointer(&buf)))[:sizeOfSliceHeader]
	copy(buf[:sizeOfSliceHeader], metadataBytes)

	metadata := *(*reflect.SliceHeader)(unsafe.Pointer(&buf))
	publicStr := reflect.StringHeader{
		Data: metadata.Data + uintptr(sizeOfSliceHeader),
		Len: l,
	}
	publicBuf := reflect.SliceHeader{
		Data: metadata.Data + uintptr(sizeOfSliceHeader),
		Len: l,
		Cap: l,
	}

	retA := *(*string)(unsafe.Pointer(&publicStr))
	retB := *(*[]byte)(unsafe.Pointer(&publicBuf))

	return retA, retB
}
