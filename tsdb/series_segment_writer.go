package tsdb

import (
	"github.com/pierrec/lz4"
)

// SeriesSegmentBuffer buffers writes to be compressed and returns values that help writing the
// data out.
type SeriesSegmentBuffer struct {
	hashTbl [1 << 16]int  // hash table for compression
	buf     [1 << 24]byte // 4MB uncompressed
	comp    [1 << 24]byte // 4MB compressed
	idx     uint32        // into buf
	// log     bool
}

// Append copies the byte slice into the uncompressed buffer. If it won't fit, it returns false and
// the caller should flush first. It returns the index the data was written at.
func (s *SeriesSegmentBuffer) Append(b []byte) (idx uint32, ok bool) {
	// most of the time the copy will proceed, so just try that first
	idx, n := s.idx, copy(s.buf[s.idx:], b)
	// if s.log {
	// 	fmt.Println(fmt.Sprintf("%p", s), "copied", n, "bytes starting at", s.idx)
	// }
	if n == len(b) {
		s.idx += uint32(n)
	}
	return idx, n == len(b)
}

// Buffered returns how many uncompressed bytes are buffered.
func (s *SeriesSegmentBuffer) Buffered() uint32 {
	return s.idx
}

// Reset undoes all of the appends that have happened.
func (s *SeriesSegmentBuffer) Reset() { s.idx = 0 }

// Compress resets the write buffer and returns a slice of the compressed data. It is only valid
// until the next call to Compress or Append. If size is the same as the data length, the data was
// not compressable.
func (s *SeriesSegmentBuffer) Compress() (size uint32, data []byte) {
	size = s.idx

	di, err := lz4.CompressBlock(s.buf[:s.idx], s.comp[:], s.hashTbl[:])
	s.Reset()

	if err != nil || di == 0 {
		return size, s.buf[:s.idx:s.idx]
	}
	return size, s.comp[:di:di]
}
