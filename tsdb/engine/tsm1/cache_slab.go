package tsm1

import (
	//"fmt"
	"reflect"
	"sync"
	"unsafe"

	"github.com/couchbase/go-slab" // slab
)

var sizeOfuintptr = unsafe.Sizeof(uintptr(0))
var sizeOfint = unsafe.Sizeof(uint(0))
var sizeOfSliceHeader = unsafe.Sizeof(reflect.SliceHeader{})
var sizeOfStringHeader = unsafe.Sizeof(reflect.StringHeader{})

type OwnedString string

func verboseMalloc(x int) []byte {
	println("go malloc", x)
	return make([]byte, x)
}

func NewCacheLocalArena() *CacheLocalArena {
	return &CacheLocalArena{
		arena: slab.NewArena(1, 64*1024*1024, 2, verboseMalloc),
	}
}

type CacheLocalArena struct {
	mu    sync.Mutex
	arena *slab.Arena
}

func (s *CacheLocalArena) get(l int) []byte {
	s.mu.Lock()
	buf := s.arena.Alloc(l)
	s.mu.Unlock()
	return buf
}
func (s *CacheLocalArena) GetOwnedString(src string) OwnedString {
	buf := s.get(int(sizeOfSliceHeader) + len(src))
	x := embedStrInBuf(buf, src)
	os := *(*OwnedString)(unsafe.Pointer(&x))


	s.Inc(os) // sanity check
	s.Dec(os) // sanity check
	return os
}
func (s *CacheLocalArena) Inc(os OwnedString) {
	strCast := *(*string)(unsafe.Pointer(&os))
	embeddedBuf := accessBufFromStr(strCast)

	s.mu.Lock()
	s.arena.AddRef(embeddedBuf)
	s.mu.Unlock()
}
func (s *CacheLocalArena) Dec(os OwnedString) {
	strCast := *(*string)(unsafe.Pointer(&os))
	embeddedBuf := accessBufFromStr(strCast)

	s.mu.Lock()
	s.arena.DecRef(embeddedBuf)
	s.mu.Unlock()
}

func embedStrInBuf(buf []byte, s string) string {
	if len(buf) != len(s) + int(sizeOfSliceHeader) {
		panic("logic error in embedStrInBuf input")
	}

	// first, copy the byte slice header info into the shadow prefix:
	srcHeader := (*reflect.SliceHeader)(unsafe.Pointer(&buf))
	dstHeader := (*reflect.SliceHeader)(unsafe.Pointer(&(buf[:sizeOfSliceHeader][0])))
	*dstHeader = *srcHeader

	// second, copy the string bytes to the buffer:
	copy(buf[sizeOfSliceHeader:], s)

	// third, and finally, construct a string header on the stack and
	// return it as a string:
	strHeader := reflect.StringHeader{
		Data: uintptr(unsafe.Pointer(&(buf[sizeOfSliceHeader]))),
		Len: len(s),
	}

	str := *(*string)(unsafe.Pointer(&strHeader))
	return str
}
func accessBufFromStr(os string) []byte {
	strHeader := *(*reflect.StringHeader)(unsafe.Pointer(&os))
	sliceHeaderStart := strHeader.Data - sizeOfSliceHeader
	sliceHeader := *(*reflect.SliceHeader)(unsafe.Pointer(sliceHeaderStart))
	slice := *(*[]byte)(unsafe.Pointer(&sliceHeader))
	return slice
}
