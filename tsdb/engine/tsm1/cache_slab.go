package tsm1

import (
	"github.com/couchbase/go-slab" // slab
	"unsafe"
)

type OwnedString string

func (os OwnedString) CopyInto(src []byte) {
	dst := ownedStringToByteSlice(os)
	copy(dst, src)
}

func NewCacheLocalArena() *CacheLocalArena {
	return &CacheLocalArena{
		arena: slab.NewArena(8, 1*1024*1024, 2, nil),
	}
}

type CacheLocalArena struct {
	arena *slab.Arena
}
func (s *CacheLocalArena) Get(l int) OwnedString {
	buf := s.arena.Alloc(l)
	x := byteSliceToOwnedString(buf)
	return x
}
func (s *CacheLocalArena) Dec(x OwnedString) {
	 buf := ownedStringToByteSlice(x)
	 s.arena.DecRef(buf)
}

func byteSliceToOwnedString(b []byte) OwnedString {
	return *(*OwnedString)(unsafe.Pointer(&b))
}
func ownedStringToByteSlice(s OwnedString) []byte {
	return *(*[]byte)(unsafe.Pointer(&s))
}
