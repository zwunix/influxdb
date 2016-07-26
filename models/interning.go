// This file could be its own package.
package models

import (
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"
	"unsafe"
)

//import "github.com/allegro/bigcache"

const (
	// taken from bigcache
	// offset64 FNVa offset basis. See https://en.wikipedia.org/wiki/Fowler–Noll–Vo_hash_function#FNV-1a_hash
	offset64 = 14695981039346656037

	// taken from bigcache
	// prime64 FNVa prime value. See https://en.wikipedia.org/wiki/Fowler–Noll–Vo_hash_function#FNV-1a_hash
	prime64 = 1099511628211
)

var (
	internShards          uint64 = 32
	internPrint           bool = false
	globalInternedBuckets [][]*internBucket
	//internMB              int = 1024
)

type internBucket struct {
	mu         sync.RWMutex
	averageLen float64
	count      int64
	items      map[string]string
}

//func (b *internBucket) debugStats() string {
//	b.mu.RLock()
//	a := b.averageLen
//	c := b.count
//	b.mu.RUnlock()
//	return fmt.Sprintf("%d,%f", a, c)
//}

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

func init() {
	if s := os.Getenv("INTERN_SHARDS"); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil {
			panic(err.Error())
		}
		internShards = uint64(n)
	}
	println("INTERN_SHARDS is", internPrint)
	if s := os.Getenv("INTERN_PRINT"); s != "" {
		internPrint = true
	}
	println("INTERN_PRINT is", internPrint)

	globalInternedBuckets = make([][]*internBucket, 5)
	for i := 0; i < 5; i++ {
		globalInternedBuckets[i] = make([]*internBucket, internShards)
		for j := uint64(0); j < internShards; j++ {
			globalInternedBuckets[i][j] = &internBucket{
				items: make(map[string]string),
			}
		}
	}
	go func() {
		for {
			<-time.After(1 * time.Second)
			for i := range globalInternedBuckets {
				dbg0 := []int64{}
				dbg1 := []int64{}
				for j := uint64(0); j < internShards; j++ {
					b := globalInternedBuckets[i][j]
					b.mu.RLock()
					x := b.averageLen
					y := b.count
					b.mu.RUnlock()
					dbg0 = append(dbg0, int64(x))
					dbg1 = append(dbg1, y)
				}

				fmt.Printf("%d avg: %v\n", i, dbg0)
				fmt.Printf("%d cnt: %v\n", i, dbg1)
			}
		}
	}()
	//if s := os.Getenv("INTERN_MB"); s != "" {
	//	n, err := strconv.Atoi(s)
	//	if err != nil {
	//		panic(err.Error())
	//	}
	//	internMB = n
	//}
	//println("INTERN_MB is", internMB)
	//config := bigcache.Config{
	//	// number of shards (must be a power of 2)
	//	Shards: internShards,
	//	// time after which entry can be evicted
	//	LifeWindow: 10 * time.Minute,
	//	// rps * lifeWindow, used only in initial memory allocation
	//	MaxEntriesInWindow: 1e7,
	//	// max entry size in bytes, used only in initial memory allocation
	//	MaxEntrySize: 1024,
	//	// prints information about additional memory allocation
	//	Verbose: true,
	//	// cache will not allocate more memory than this limit, value in MB
	//	// if value is reached then the oldest entries can be overridden for the new ones
	//	// 0 value means no size limit
	//	HardMaxCacheSize: internMB,
	//	// callback fired when the oldest entry is removed because of its
	//	// expiration time or no space left for the new entry. Default value is nil which
	//	// means no callback and it prevents from unwrapping the oldest entry.
	//	OnRemove: nil,
	//}

	//bc, err := bigcache.NewBigCache(config)
	//if err != nil {
	//	panic(err.Error())
	//}
	//globalInternedStrings = bc
}

//func byteSliceToString(b []byte) string {
//	return *(*string)(unsafe.Pointer(&b))
//}

func bucketPos(l int) int {
	// heuristic
	if l <= 8 {
		return 0
	} else if l <= 64 {
		return 1
	} else if l <= 256 {
		return 2
	} else if l <= 512 {
		return 3
	} else {
		return 4
	}
}
func GetInternedStringFromBytes(x []byte) string {
	h := int(FNV64a_Sum64(x) % uint64(internShards))
	b := globalInternedBuckets[bucketPos(len(x))][h]

	b.mu.RLock()
	s, ok := b.items[string(x)]
	b.mu.RUnlock()

	if ok {
		return s
	}

	b.mu.Lock()
	s, ok = b.items[string(x)]
	if !ok {
		// heap alloc
		s = string(x)
		b.items[s] = s

		newAvg := (b.averageLen*float64(b.count) + float64(len(s))) / (float64(b.count) + 1)

		b.averageLen = newAvg
		b.count++
	}
	b.mu.Unlock()
	return s

	//sKey := byteSliceToString(x)
	//bVal, err := globalInternedStrings.Get(sKey)
	//ok := err == nil // (*BigCache).Get only has one kind of error

	//if ok {
	//	return byteSliceToString(bVal)
	//}

	//// slow path: need to copy into the cache

	//err = globalInternedStrings.Set(sKey, x)
	//if err != nil {
	//	// (*BigCache).Set returns an error if it's full
	//	return string(x) // failsafe alloc
	//}

	//bVal, err = globalInternedStrings.Get(sKey)
	//if err != nil {
	//	panic("unepxected 2nd get error")
	//}
	//return byteSliceToString(bVal)
}
