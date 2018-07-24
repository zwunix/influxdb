package tsdb

import (
	"github.com/bkaradzic/go-lz4"
)

const (
	uncompressedSize = 1 << 19 // 512KB uncompressed
	compressedSize   = uncompressedSize + (uncompressedSize / 255) + 16 + 4
)

// SeriesSegmentBuffer buffers writes to be compressed and returns values that help writing the
// data out.
type SeriesSegmentBuffer struct {
	buf  [uncompressedSize]byte
	comp [compressedSize]byte
	idx  uint32 // into buf
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

	data, err := lz4.Encode(s.comp[:], s.buf[:s.idx])
	s.Reset()

	if err != nil || data == nil || uint32(len(data)) >= size {
		return size, s.buf[:size:size]
	}
	return size, data
}
