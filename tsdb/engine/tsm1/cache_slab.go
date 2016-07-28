package tsm1

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"sync"
	"time"
	"unsafe"

	"github.com/couchbase/go-slab" // slab
)

var sizeOfuintptr = unsafe.Sizeof(uintptr(0))
var sizeOfint = unsafe.Sizeof(uint(0))

//var sizeOfSliceHeader = unsafe.Sizeof(reflect.SliceHeader{})
var sizeOfStringHeader = unsafe.Sizeof(reflect.StringHeader{})

type OwnedString string

func (os OwnedString) ViewAsBytes() []byte {
	osHeader := *(*reflect.StringHeader)(unsafe.Pointer(&os))
	sliceHeader := reflect.SliceHeader{
		Data: osHeader.Data,
		Len:  osHeader.Len,
		Cap:  osHeader.Len,
	}
	bufHeader := *(*[]byte)(unsafe.Pointer(&sliceHeader))
	return bufHeader
}
func StringViewAsBytes(s string) []byte {
	osHeader := *(*reflect.StringHeader)(unsafe.Pointer(&s))
	sliceHeader := reflect.SliceHeader{
		Data: osHeader.Data,
		Len:  osHeader.Len,
		Cap:  osHeader.Len,
	}
	bufHeader := *(*[]byte)(unsafe.Pointer(&sliceHeader))
	return bufHeader
}

func verboseMalloc(x int) []byte {
	println("go malloc", x)
	return make([]byte, x)
}

var CACHE_SLAB_SHARDS int64 = 4
var CACHE_SLAB_MB int = 1
var CACHE_SLAB_ITEM_BYTES int = 1
var CACHE_SLAB_QUEUE_DEPTH int = 1000

func init() {
	if s := os.Getenv("CACHE_SLAB_SHARDS"); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil {
			panic(err.Error())
		}
		CACHE_SLAB_SHARDS = int64(n)
	}
	println("CACHE_SLAB_SHARDS is", CACHE_SLAB_SHARDS)
	if s := os.Getenv("CACHE_SLAB_MB"); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil {
			panic(err.Error())
		}
		CACHE_SLAB_MB = int(n)
	}
	println("CACHE_SLAB_MB is", CACHE_SLAB_MB)
	if s := os.Getenv("CACHE_SLAB_ITEM_BYTES"); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil {
			panic(err.Error())
		}
		CACHE_SLAB_ITEM_BYTES = int(n)
	}
	println("CACHE_SLAB_ITEM_BYTES is", CACHE_SLAB_ITEM_BYTES)
	if s := os.Getenv("CACHE_SLAB_QUEUE_DEPTH"); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil {
			panic(err.Error())
		}
		CACHE_SLAB_QUEUE_DEPTH = int(n)
	}
	println("CACHE_SLAB_QUEUE_DEPTH is", CACHE_SLAB_QUEUE_DEPTH)
}

func NewCacheLocalArena() *CacheLocalArena {
	arenas := make([]*slab.Arena, CACHE_SLAB_SHARDS)
	mus := make([]*sync.Mutex, CACHE_SLAB_SHARDS)
	queues := make([]chan CLAJob, CACHE_SLAB_SHARDS)
	for i := range arenas {
		j := i
		f := func(l int) []byte {
			println("go malloc", j, l)
			return make([]byte, l)
		}
		arenas[i] = slab.NewArena(CACHE_SLAB_ITEM_BYTES, CACHE_SLAB_MB*1024*1024, 2, f)
		mus[i] = &sync.Mutex{}
		queues[i] = make(chan CLAJob, CACHE_SLAB_QUEUE_DEPTH)
	}
	cla := &CacheLocalArena{
		arenas: arenas,
		mus:    mus,
		queues: queues,
	}

	for i := range arenas {
		go func(i int) {
			queue := cla.queues[i]
			println("started worker", i)
			n := 0
			for j := range queue {
				n++
				if n%100000 == 0 {
					println(i, "worker did", n)
				}
				//println(i, "doing a job")
				j.Do()
			}
		}(i)
	}

	go func(cla *CacheLocalArena) {
		for {
			<-time.After(5 * time.Second)
			for i := range cla.mus {
				stats := map[string]int64{}
				//cla.mus[i].Lock()
				cla.arenas[i].Stats(stats)
				//cla.mus[i].Unlock()
				if stats["totAllocs"] > 0 {
					fmt.Printf("%d/%d: totAllocs: %d, totAddRefs: %d, totDecRefs: %d, totDecRefZeroes: %d\n", i+1, len(cla.mus),
						stats["totAllocs"], stats["totAddRefs"], stats["totDecRefs"], stats["totDecRefZeroes"])
					//keys := []string{}
					//for key := range stats {
					//	keys = append(keys, key)
					//}
					//sort.Strings(keys)
					//s := fmt.Sprintf("%d: ", i)
					//for i, key := range keys {
					//	s += fmt.Sprintf("%v: %v", key, stats[key])
					//	if i + 1 < len(keys) {
					//		s += ", "
					//	}
					//}
					//fmt.Println(s)
				}
			}
		}
	}(cla)
	return cla
}

type CacheLocalArena struct {
	mus    []*sync.Mutex
	arenas []*slab.Arena
	queues []chan CLAJob
}

func (s *CacheLocalArena) get(arenaId, l int) []byte {
	panic("unused")
	arena := s.arenas[arenaId]
	mu := s.mus[arenaId]
	mu.Lock()
	buf := arena.Alloc(l)
	mu.Unlock()
	return buf
}

type GetOwnedStringJob struct {
	l     int
	mu    *sync.Mutex
	arena *slab.Arena
	ret   chan []byte
}

func (j *GetOwnedStringJob) Do() {
	//println("trying to do GetOwnedString")
	//j.mu.Lock()
	buf := j.arena.Alloc(j.l)
	//j.mu.Unlock()
	j.ret <- buf
}

var GetOwnedStringJobPool = &sync.Pool{
	New: func() interface{} {
		return &GetOwnedStringJob{
			ret: make(chan []byte, 1),
		}
	},
}

func (s *CacheLocalArena) GetOwnedString(src string) OwnedString {
	l := int(sizeOfSliceHeader) + len(src) + 8
	arenaId := int(FNV64a_Sum64(StringViewAsBytes(src)) % uint64(CACHE_SLAB_SHARDS))

	j := GetOwnedStringJobPool.Get().(*GetOwnedStringJob)
	j.l = l
	j.arena = s.arenas[arenaId]
	j.mu = s.mus[arenaId]

	s.queues[arenaId] <- j
	buf := <-j.ret
	GetOwnedStringJobPool.Put(j)

	x := embedStrInBuf(buf, src)
	os := *(*OwnedString)(unsafe.Pointer(&x))

	//s.Inc(os) // sanity check
	//s.Dec(os) // sanity check
	return os
}

type IncJob struct {
	n           int
	embeddedBuf []byte
	mu          *sync.Mutex
	arena       *slab.Arena
	wg          *sync.WaitGroup
}

func (j *IncJob) Do() {
	//println("trying to do GetOwnedString")
	for ; j.n > 0; j.n-- {
		//j.mu.Lock()
		j.arena.AddRef(j.embeddedBuf)
		//j.mu.Unlock()
	}
	j.wg.Done()
}

var IncJobPool = &sync.Pool{
	New: func() interface{} {
		return &IncJob{
			wg: &sync.WaitGroup{},
		}
	},
}

func (s *CacheLocalArena) Inc(os OwnedString, n int) {
	strCast := *(*string)(unsafe.Pointer(&os))
	embeddedBuf, hash := accessBufFromStr(strCast)

	arenaId := hash % uint64(CACHE_SLAB_SHARDS)
	arena := s.arenas[arenaId]
	mu := s.mus[arenaId]

	j := IncJobPool.Get().(*IncJob)
	j.embeddedBuf = embeddedBuf
	j.n = n
	j.mu = mu
	j.arena = arena
	j.wg.Add(1)

	s.queues[arenaId] <- j
	j.wg.Wait()
	IncJobPool.Put(j)
}

type DecJob struct {
	n           int
	embeddedBuf []byte
	mu          *sync.Mutex
	arena       *slab.Arena
	wg          *sync.WaitGroup
}

func (j *DecJob) Do() {
	//println("trying to do GetOwnedString")
	for ; j.n > 0; j.n-- {
		//j.mu.Lock()
		j.arena.DecRef(j.embeddedBuf)
		//j.mu.Unlock()
	}
	j.wg.Done()
}

var DecJobPool = &sync.Pool{
	New: func() interface{} {
		return &DecJob{
			wg: &sync.WaitGroup{},
		}
	},
}

func (s *CacheLocalArena) Dec(os OwnedString, n int) {
	strCast := *(*string)(unsafe.Pointer(&os))
	embeddedBuf, hash := accessBufFromStr(strCast)

	arenaId := hash % uint64(CACHE_SLAB_SHARDS)
	arena := s.arenas[arenaId]
	//mu := s.mus[arenaId]

	j := DecJobPool.Get().(*DecJob)
	j.n = n
	j.embeddedBuf = embeddedBuf
	j.arena = arena
	j.wg.Add(1)

	s.queues[arenaId] <- j
	j.wg.Wait()
	DecJobPool.Put(j)

	return
}

func embedStrInBuf(buf []byte, s string) string {
	if len(buf) != len(s)+int(sizeOfSliceHeader)+8 {
		panic("logic error in embedStrInBuf input")
	}

	// first, copy the byte slice header info into the shadow prefix:
	//srcHeader := (*reflect.SliceHeader)(unsafe.Pointer(&buf))
	//dstHeader := (*reflect.SliceHeader)(unsafe.Pointer(&(buf[:sizeOfSliceHeader][0])))
	//*dstHeader = *srcHeader
	//srcHeader := *(*[24]byte)(unsafe.Pointer(&buf))
	dstHeader := (*reflect.SliceHeader)(unsafe.Pointer(&(buf[0])))
	(*dstHeader).Data = uintptr(unsafe.Pointer(&(buf[0])))
	(*dstHeader).Len = len(buf)
	(*dstHeader).Cap = cap(buf)

	// second, copy the hash into the next 8 bytes:
	hash := FNV64a_Sum64(StringViewAsBytes(s))
	dst8 := (*uint64)(unsafe.Pointer(&buf[sizeOfSliceHeader : sizeOfSliceHeader+8][0]))
	*dst8 = hash

	// third, copy the string bytes to the buffer:
	copy(buf[sizeOfSliceHeader+8:], s)

	// fourth, and finally, construct a string header on the stack and
	// return it as a string:
	strHeader := reflect.StringHeader{
		Data: uintptr(unsafe.Pointer(&(buf[sizeOfSliceHeader+8]))),
		Len:  len(s),
	}

	str := *(*string)(unsafe.Pointer(&strHeader))
	return str
}
func accessBufFromStr(os string) ([]byte, uint64) {
	strHeader := *(*reflect.StringHeader)(unsafe.Pointer(&os))
	sliceHeaderStart := strHeader.Data - uintptr(sizeOfSliceHeader) - 8
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

type CLAJob interface {
	Do()
}
