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

func (os OwnedString) ViewAsBytes() []byte {
	osHeader := *(*reflect.StringHeader)(unsafe.Pointer(&os))
	sliceHeader := reflect.SliceHeader{
		Data: osHeader.Data,
		Len: osHeader.Len,
		Cap: osHeader.Len,
	}
	bufHeader := *(*[]byte)(unsafe.Pointer(&sliceHeader))
	return bufHeader
}
func StringViewAsBytes(s string) []byte {
	osHeader := *(*reflect.StringHeader)(unsafe.Pointer(&s))
	sliceHeader := reflect.SliceHeader{
		Data: osHeader.Data,
		Len: osHeader.Len,
		Cap: osHeader.Len,
	}
	bufHeader := *(*[]byte)(unsafe.Pointer(&sliceHeader))
	return bufHeader
}

func verboseMalloc(x int) []byte {
	println("go malloc", x)
	return make([]byte, x)
}

func NewCacheLocalArena() *CacheLocalArena {
	arenas := make([]*slab.Arena, 32)
	mus := make([]*sync.Mutex, 32)
	for i := range arenas {
		j := i
		f := func(l int) []byte {
			println("go malloc", j, l)
			return make([]byte, l)
		}
		arenas[i] = slab.NewArena(1, 4*1024*1024, 2, f)
		mus[i] = &sync.Mutex{}
	}
	return &CacheLocalArena{
		arenas: arenas,
		mus: mus,
	}
}

type CacheLocalArena struct {
	mus    []*sync.Mutex
	arenas []*slab.Arena
}

func (s *CacheLocalArena) get(arenaId, l int) []byte {
	arena := s.arenas[arenaId]
	mu := s.mus[arenaId]
	mu.Lock()
	buf := arena.Alloc(l)
	mu.Unlock()
	return buf
}
func (s *CacheLocalArena) GetOwnedString(src string) OwnedString {
	arenaId := int(FNV64a_Sum64(StringViewAsBytes(src)) % uint64(32))

	buf := s.get(arenaId, int(sizeOfSliceHeader) + len(src) + 8)
	x := embedStrInBuf(buf, src)
	os := *(*OwnedString)(unsafe.Pointer(&x))


	//s.Inc(os) // sanity check
	//s.Dec(os) // sanity check
	return os
}
func (s *CacheLocalArena) Inc(os OwnedString) {
	strCast := *(*string)(unsafe.Pointer(&os))
	embeddedBuf, hash := accessBufFromStr(strCast)

	arenaId := hash % uint64(32)
	arena := s.arenas[arenaId]
	mu := s.mus[arenaId]


	mu.Lock()
	arena.AddRef(embeddedBuf)
	mu.Unlock()
}
func (s *CacheLocalArena) Dec(os OwnedString) {
	strCast := *(*string)(unsafe.Pointer(&os))
	embeddedBuf, hash := accessBufFromStr(strCast)

	arenaId := hash % uint64(32)
	arena := s.arenas[arenaId]
	mu := s.mus[arenaId]

	mu.Lock()
	arena.DecRef(embeddedBuf)
	mu.Unlock()
}

func (s *CacheLocalArena) DecMulti(os OwnedString, n int) {
	strCast := *(*string)(unsafe.Pointer(&os))
	embeddedBuf, hash := accessBufFromStr(strCast)

	arenaId := hash % uint64(32)
	arena := s.arenas[arenaId]
	mu := s.mus[arenaId]

	mu.Lock()
	for i := 0; i < n; i++ {
		arena.DecRef(embeddedBuf)
	}
	mu.Unlock()
}

func embedStrInBuf(buf []byte, s string) string {
	if len(buf) != len(s) + int(sizeOfSliceHeader) + 8 {
		panic("logic error in embedStrInBuf input")
	}

	// first, copy the byte slice header info into the shadow prefix:
	srcHeader := (*reflect.SliceHeader)(unsafe.Pointer(&buf))
	dstHeader := (*reflect.SliceHeader)(unsafe.Pointer(&(buf[:sizeOfSliceHeader][0])))
	*dstHeader = *srcHeader

	// second, copy the hash into the next 8 bytes:
	hash := FNV64a_Sum64(StringViewAsBytes(s))
	dst8 := (*uint64)(unsafe.Pointer(&buf[sizeOfSliceHeader:sizeOfSliceHeader+8][0]))
	*dst8 = hash

	// third, copy the string bytes to the buffer:
	copy(buf[sizeOfSliceHeader+8:], s)

	// fourth, and finally, construct a string header on the stack and
	// return it as a string:
	strHeader := reflect.StringHeader{
		Data: uintptr(unsafe.Pointer(&(buf[sizeOfSliceHeader+8]))),
		Len: len(s),
	}

	str := *(*string)(unsafe.Pointer(&strHeader))
	return str
}
func accessBufFromStr(os string) ([]byte, uint64) {
	strHeader := *(*reflect.StringHeader)(unsafe.Pointer(&os))
	sliceHeaderStart := strHeader.Data - sizeOfSliceHeader - 8
	sliceHeader := *(*reflect.SliceHeader)(unsafe.Pointer(sliceHeaderStart))
	hashStart := strHeader.Data - 8
	hash := *(*uint64)(unsafe.Pointer(hashStart))
	slice := *(*[]byte)(unsafe.Pointer(&sliceHeader))
	return slice, hash
}
// hashing/buckets
const (
	// taken from bigcache
	// offset64 FNVa offset basis. See https://en.wikipedia.org/wiki/Fowler–Noll–Vo_hash_function#FNV-1a_hash
	offset64 = 14695981039346656037

	// taken from bigcache
	// prime64 FNVa prime value. See https://en.wikipedia.org/wiki/Fowler–Noll–Vo_hash_function#FNV-1a_hash
	prime64 = 1099511628211
)
// adapted from bigcache
// Sum64 gets the string and returns its uint64 hash value.
func FNV64a_Sum64(key []byte) uint64 {
	var hash uint64 = offset64
	i := 0
	// this speedup may break FNV1a hash properties
	for i+8 <= len(key) {
		hash ^= *(*uint64)(unsafe.Pointer(&key[i]))
		hash *= prime64
		i += 8
	}
	for ; i < len(key); i++ {
		hash ^= uint64(key[i])
		hash *= prime64
	}

	return hash
}
