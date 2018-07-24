package tsdb

import "github.com/pierrec/lz4"

// SeriesSegmentBuffer buffers writes to be compressed and returns values that help writing the
// data out.
type SeriesSegmentBuffer struct {
	hashTbl [1 << 16]int  // hash table for compression
	buf     [1 << 24]byte // 4MB uncompressed
	comp    [1 << 24]byte // 4MB compressed
	idx     uint32        // into buf
}

// Append copies the byte slice into the uncompressed buffer. If it won't fit, it returns false and
// the caller should flush first. It returns the index the data was written at.
func (s *SeriesSegmentBuffer) Append(b []byte) (idx uint32, ok bool) {
	// most of the time the copy will proceed, so just try that first
	idx, n := s.idx, copy(s.buf[s.idx:], b)
	if n == len(b) {
		s.idx += uint32(n)
	}
	return idx, n == len(b)
}

// Reset undoes all of the appends that have happened.
func (s *SeriesSegmentBuffer) Reset() { s.idx = 0 }

// Compress resets the write buffer and returns a slice of the compressed data. It is only valid
// until the next call to Compress or Append. If size is the same as the data length, the data was
// not compressable.
func (s *SeriesSegmentBuffer) Compress() (size uint32, data []byte) {
	di, err := lz4.CompressBlock(s.buf[:s.idx], s.comp[:], s.hashTbl[:])
	s.Reset()
	if err != nil || di == 0 || uint32(di) == s.idx {
		return s.idx, s.buf[:s.idx:s.idx]
	}
	return uint32(di), s.comp[:di:di]
}
