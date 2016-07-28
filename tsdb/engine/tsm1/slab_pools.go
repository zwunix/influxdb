package tsm1

import (
	//"fmt"
	//"reflect"
	//"os"
	//"strconv"
	"sync"
	//"time"
	//"unsafe"

	"github.com/couchbase/go-slab" // slab
)

type ByteSliceSlabPool struct {
	arena *slab.Arena
	sync.Mutex
	refs int64
}

func NewByteSliceSlabPool() *ByteSliceSlabPool {
	return &ByteSliceSlabPool{
		arena: slab.NewArena(1, 1024, 2, nil),
		Mutex: sync.Mutex{},
		refs: 0,
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
