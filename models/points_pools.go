package models

import "sync"

// hashKeyTagsPool is used by (Tags).HashKey
var hashKeyTagsPool = &sync.Pool{
	New: func() interface{} {
		return Tags{}
	},
}

// hashKeyBufSlicePool is used by (Tags).HashKey
var hashKeyBufSlicePool = &sync.Pool{
	New: func() interface{} {
		return [][]byte{}
	},
}

func getHashKeyTagsPool() Tags {
	return hashKeyTagsPool.Get().(Tags)
}
func putHashKeyTagsPool(x Tags) {
	x = x[:0]
	hashKeyTagsPool.Put(x)
}

func getHashKeyBufSlicePool() [][]byte {
	return hashKeyBufSlicePool.Get().([][]byte)
}
func putHashKeyBufSlicePool(x [][]byte) {
	x = x[:0]
	hashKeyBufSlicePool.Put(x)
}
