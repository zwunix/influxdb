package tsdb

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/influxdata/influxdb/pkg/mmap"
	"github.com/pierrec/lz4"
)

const (
	SeriesSegmentVersion = 1
	SeriesSegmentMagic   = "SSEG"

	SeriesSegmentHeaderSize = 4 + 1 // magic + version
)

// Series entry constants.
const (
	SeriesEntryFlagSize   = 1
	SeriesEntryHeaderSize = 1 + 8 // flag + id

	SeriesEntryInsertFlag    = 0x01
	SeriesEntryTombstoneFlag = 0x02
)

var (
	ErrInvalidSeriesSegment        = errors.New("invalid series segment")
	ErrInvalidSeriesSegmentVersion = errors.New("invalid series segment version")
	ErrSeriesSegmentNotWritable    = errors.New("series segment not writable")
)

// SeriesSegment represents a log of series entries.
type SeriesSegment struct {
	id   uint16
	path string
	lz4  bool // if the segment is lz4 compressed
	log  bool

	mu        sync.Mutex
	udata     []byte
	cachedPos uint32

	data   []byte               // mmap file
	file   *os.File             // write file handle
	w      *bufio.Writer        // bufferred file handle
	lz4buf *SeriesSegmentBuffer // buffered compressable data
	size   uint32               // current file size
}

// NewSeriesSegment returns a new instance of SeriesSegment.
func NewSeriesSegment(id uint16, path string, log bool) *SeriesSegment {
	return &SeriesSegment{
		id:   id,
		path: path,
		lz4:  strings.HasSuffix(path, ".lz4"),
		log:  log,
	}
}

// CreateSeriesSegment generates an empty segment at path.
func CreateSeriesSegment(id uint16, path string, log bool) (*SeriesSegment, error) {
	// Generate segment in temp location.
	f, err := os.Create(path + ".initializing")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Write header to file and close.
	hdr := NewSeriesSegmentHeader()
	if _, err := hdr.WriteTo(f); err != nil {
		return nil, err
	} else if err := f.Truncate(int64(SeriesSegmentSize(id))); err != nil {
		return nil, err
	} else if err := f.Close(); err != nil {
		return nil, err
	}

	// Swap with target path.
	if err := os.Rename(f.Name(), path); err != nil {
		return nil, err
	}

	// Open segment at new location.
	segment := NewSeriesSegment(id, path, log)
	if err := segment.Open(); err != nil {
		return nil, err
	}
	return segment, nil
}

// Open memory maps the data file at the file's path.
func (s *SeriesSegment) Open() error {
	if err := func() (err error) {
		// Memory map file data.
		if s.data, err = mmap.Map(s.path, int64(SeriesSegmentSize(s.id))); err != nil {
			return err
		}

		// Read header.
		hdr, err := ReadSeriesSegmentHeader(s.data)
		if err != nil {
			return err
		} else if hdr.Version != SeriesSegmentVersion {
			return ErrInvalidSeriesSegmentVersion
		}

		return nil
	}(); err != nil {
		s.Close()
		return err
	}

	return nil
}

// InitForWrite initializes a write handle for the segment.
// This is only used for the last segment in the series file.
func (s *SeriesSegment) InitForWrite() (err error) {
	// Only calculcate segment data size if writing.
	s.size, err = s.ForEachEntry(func(_ uint8, _ uint64, _ int64, _ []byte) error {
		return nil
	})
	if err != nil {
		return err
	}

	// Open file handler for writing & seek to end of data.
	if s.file, err = os.OpenFile(s.path, os.O_WRONLY|os.O_CREATE, 0666); err != nil {
		return err
	} else if _, err := s.file.Seek(int64(s.size), io.SeekStart); err != nil {
		return err
	}
	s.w = bufio.NewWriterSize(s.file, 32*1024)

	if s.lz4 {
		s.lz4buf = new(SeriesSegmentBuffer)
		// s.lz4buf.log = s.log
	}

	return nil
}

// Close unmaps the segment.
func (s *SeriesSegment) Close() (err error) {
	if e := s.CloseForWrite(); e != nil && err == nil {
		err = e
	}

	if s.data != nil {
		if e := mmap.Unmap(s.data); e != nil && err == nil {
			err = e
		}
		s.data = nil
	}

	return err
}

func (s *SeriesSegment) CloseForWrite() (err error) {
	if s.w != nil {
		if e := s.lz4Flush(); e != nil && err == nil {
			err = e
		}
		if e := s.w.Flush(); e != nil && err == nil {
			err = e
		}
		s.w = nil
	}

	if s.file != nil {
		if e := s.file.Close(); e != nil && err == nil {
			err = e
		}
		s.file = nil
	}
	return err
}

// Data returns the raw data.
func (s *SeriesSegment) Data() []byte { return s.data }

// ID returns the id the segment was initialized with.
func (s *SeriesSegment) ID() uint16 { return s.id }

// Size returns the size of the data in the segment.
// This is only populated once InitForWrite() is called.
func (s *SeriesSegment) Size() int64 { return int64(s.size) }

var cacheHits, cacheTotal uint64
var compBytes, compTotal uint64

func init() {
	go func() {
		for range time.NewTicker(time.Second).C {
			hits, total := atomic.LoadUint64(&cacheHits), atomic.LoadUint64(&cacheTotal)
			cBytes, cTotal := atomic.LoadUint64(&compBytes), atomic.LoadUint64(&compTotal)
			fmt.Println(
				"hits:", hits,
				"total:", total,
				"perc:", float64(hits)/float64(total),
				"compressed:", cBytes,
				"total:", cTotal,
				"perc:", float64(cBytes)/float64(cTotal),
			)
		}
	}()
}

// Slice returns a byte slice starting at pos.
func (s *SeriesSegment) Slice(pos uint32, index uint32, compressed bool) []byte {
	data := s.data[pos:]
	if !compressed {
		return data
	}

	// if s.log {
	// 	fmt.Println("<=", s.id, pos, index)
	// }

	usize := binary.BigEndian.Uint32(data[0:4])
	csize := binary.BigEndian.Uint32(data[4:8])
	data = data[8:]

	if usize == csize { // it didn't compress
		return data[index:]
	}

	// if s.log && index > usize {
	// 	panic("index > usize")
	// }

	// TODO(jeff): we can do a better job here with the caching
	s.mu.Lock()
	atomic.AddUint64(&cacheTotal, 1)
	if pos != s.cachedPos || s.udata == nil {
		s.udata = make([]byte, usize)
		lz4.UncompressBlock(data[:csize], s.udata)
		s.cachedPos = pos
	} else {
		atomic.AddUint64(&cacheHits, 1)
	}
	udata := s.udata
	s.mu.Unlock()

	return udata[index:]
}

func logStack() {
	var buf [4096]byte
	fmt.Println(string(buf[:runtime.Stack(buf[:], false)]))
}

func (s *SeriesSegment) lz4Flush() (err error) {
	if !s.lz4 {
		return nil
	}

	size, data := s.lz4buf.Compress()

	var buf [8]byte
	binary.BigEndian.PutUint32(buf[0:4], uint32(size))
	binary.BigEndian.PutUint32(buf[4:8], uint32(len(data)))

	if _, err := s.w.Write(buf[:]); err != nil {
		return err
	}
	if _, err := s.w.Write(data); err != nil {
		return err
	}

	s.size += 8 + uint32(len(data))

	// if s.log {
	// 	fmt.Println(fmt.Sprintf("%p", s.lz4buf), "compressed", size, "into", len(data), "and now at", s.size)
	// }
	atomic.AddUint64(&compTotal, uint64(size))
	atomic.AddUint64(&compBytes, uint64(len(data)))

	return nil
}

// WriteLogEntry writes entry data into the segment.
// Returns the offset of the beginning of the entry.
func (s *SeriesSegment) WriteLogEntry(data []byte) (offset int64, err error) {
	if s.lz4 {
		for first := true; ; first = false {
			if !s.CanWrite(data) {
				return 0, ErrSeriesSegmentNotWritable
			}

			index, ok := s.lz4buf.Append(data)
			if ok {
				// if s.log {
				// 	fmt.Println("=>", s.id, s.size, index)
				// }
				return JoinSeriesOffset(s.id, s.size, index, true), nil
			} else if !first {
				return 0, ErrSeriesSegmentNotWritable
			} else if err := s.lz4Flush(); err != nil {
				return 0, err
			}

			// if s.log {
			// 	fmt.Println("performed flush")
			// }
			first = false
		}
	}

	if !s.CanWrite(data) {
		return 0, ErrSeriesSegmentNotWritable
	}

	offset = JoinSeriesOffset(s.id, s.size, 0, false)
	if _, err := s.w.Write(data); err != nil {
		return 0, err
	}
	s.size += uint32(len(data))

	return offset, nil
}

// CanWrite returns true if segment has space to write entry data.
func (s *SeriesSegment) CanWrite(data []byte) bool {
	if s.w == nil {
		return false
	}
	worst := s.size + uint32(len(data))
	if s.lz4 {
		worst += 8 + s.lz4buf.Buffered()
	}
	return worst <= SeriesSegmentSize(s.id)
}

// Flush flushes the buffer to disk.
func (s *SeriesSegment) Flush() error {
	if s.w == nil {
		return nil
	}
	if err := s.lz4Flush(); err != nil {
		return err
	}
	return s.w.Flush()
}

// AppendSeriesIDs appends all the segments ids to a slice. Returns the new slice.
func (s *SeriesSegment) AppendSeriesIDs(a []uint64) []uint64 {
	s.ForEachEntry(func(flag uint8, id uint64, _ int64, _ []byte) error {
		if flag == SeriesEntryInsertFlag {
			a = append(a, id)
		}
		return nil
	})
	return a
}

// MaxSeriesID returns the highest series id in the segment.
func (s *SeriesSegment) MaxSeriesID() uint64 {
	var max uint64
	s.ForEachEntry(func(flag uint8, id uint64, _ int64, _ []byte) error {
		if flag == SeriesEntryInsertFlag && id > max {
			max = id
		}
		return nil
	})
	return max
}

// ForEachEntry executes fn for every entry in the segment.
func (s *SeriesSegment) ForEachEntry(fn func(flag uint8, id uint64, offset int64, key []byte) error) (pos uint32, err error) {
	if s.lz4 {
		var udata []byte
	blocks:
		for pos = uint32(SeriesSegmentHeaderSize); pos < uint32(len(s.data)); {
			usize := binary.BigEndian.Uint32(s.data[pos+0 : pos+4])
			csize := binary.BigEndian.Uint32(s.data[pos+4 : pos+8])
			if usize == 0 || csize == 0 {
				break
			}

			if uint32(len(udata)) < usize {
				udata = make([]byte, usize)
			}
			lz4.UncompressBlock(s.data[pos:pos+csize], udata)

			for index := uint32(0); index < uint32(len(udata)); {
				flag, id, key, sz := ReadSeriesEntry(udata[index:])
				if !IsValidSeriesEntryFlag(flag) {
					break blocks
				}

				offset := JoinSeriesOffset(s.id, pos, index, true)
				if err := fn(flag, id, offset, key); err != nil {
					return 0, err
				}

				index += uint32(sz)
			}

			pos += csize
		}
		return pos, nil
	}

	for pos = uint32(SeriesSegmentHeaderSize); pos < uint32(len(s.data)); {
		flag, id, key, sz := ReadSeriesEntry(s.data[pos:])
		if !IsValidSeriesEntryFlag(flag) {
			break
		}

		offset := JoinSeriesOffset(s.id, pos, 0, false)
		if err := fn(flag, id, offset, key); err != nil {
			return 0, err
		}
		pos += uint32(sz)
	}
	return pos, nil
}

// Clone returns a copy of the segment. Excludes the write handler, if set.
func (s *SeriesSegment) Clone() *SeriesSegment {
	return &SeriesSegment{
		id:   s.id,
		path: s.path,
		data: s.data,
		size: s.size,
	}
}

// CloneSeriesSegments returns a copy of a slice of segments.
func CloneSeriesSegments(a []*SeriesSegment) []*SeriesSegment {
	other := make([]*SeriesSegment, len(a))
	for i := range a {
		other[i] = a[i].Clone()
	}
	return other
}

// FindSegment returns a segment by id.
func FindSegment(a []*SeriesSegment, id uint16) *SeriesSegment {
	for _, segment := range a {
		if segment.id == id {
			return segment
		}
	}
	return nil
}

// ReadSeriesKeyFromSegments returns a series key from an offset within a set of segments.
func ReadSeriesKeyFromSegments(a []*SeriesSegment, offset int64) []byte {
	segmentID, pos, index, compressed := SplitSeriesOffset(offset)
	segment := FindSegment(a, segmentID)
	if segment == nil {
		return nil
	}
	buf := segment.Slice(pos, index, compressed)
	key, _ := ReadSeriesKey(buf)
	return key
}

// JoinSeriesOffset returns an offset that combines the 2-byte segmentID and 4-byte pos.
func JoinSeriesOffset(segmentID uint16, pos uint32, index uint32, compressed bool) int64 {
	out := (int64(segmentID) << 32) | int64(pos)
	if compressed {
		out |= int64(index&0x3FFFFFF)<<40 | 1<<62
	}
	return out
}

// SplitSeriesOffset splits a offset into its 2-byte segmentID and 4-byte pos parts.
func SplitSeriesOffset(offset int64) (segmentID uint16, pos uint32, index uint32, compressed bool) {
	segmentID = uint16((offset >> 32) & 0xFFFF)
	pos = uint32(offset & 0xFFFFFFFF)

	if offset&(1<<62) != 0 {
		compressed = true
		index = uint32((offset >> 40) & 0x3FFFFF)
		segmentID &= 0xFF
	}

	return segmentID, pos, index, compressed
}

// func init() {
// 	offset := JoinSeriesOffset(1, 2, 3, true)
// 	fmt.Printf("%064b\n", 1<<62)
// 	seg, pos, index, comp := SplitSeriesOffset(offset)
// 	fmt.Println(fmt.Sprintf("%064b", offset), offset)
// 	fmt.Println(seg, pos, index, comp)
// 	os.Exit(0)
// }

// IsValidSeriesSegmentFilename returns true if filename is a 4-character lowercase hexidecimal number.
func IsValidSeriesSegmentFilename(filename string) bool {
	return seriesSegmentFilenameRegex.MatchString(filename)
}

// ParseSeriesSegmentFilename returns the id represented by the hexidecimal filename.
func ParseSeriesSegmentFilename(filename string) (uint16, error) {
	i, err := strconv.ParseUint(filename, 16, 32)
	return uint16(i), err
}

var seriesSegmentFilenameRegex = regexp.MustCompile(`^[0-9a-f]{4}$`)

// SeriesSegmentSize returns the maximum size of the segment.
// The size goes up by powers of 2 starting from 4MB and reaching 256MB.
func SeriesSegmentSize(id uint16) uint32 {
	const min = 22 // 4MB
	const max = 28 // 256MB

	shift := id + min
	if shift >= max {
		shift = max
	}
	return 1 << shift
}

// SeriesSegmentHeader represents the header of a series segment.
type SeriesSegmentHeader struct {
	Version uint8
}

// NewSeriesSegmentHeader returns a new instance of SeriesSegmentHeader.
func NewSeriesSegmentHeader() SeriesSegmentHeader {
	return SeriesSegmentHeader{Version: SeriesSegmentVersion}
}

// ReadSeriesSegmentHeader returns the header from data.
func ReadSeriesSegmentHeader(data []byte) (hdr SeriesSegmentHeader, err error) {
	r := bytes.NewReader(data)

	// Read magic number.
	magic := make([]byte, len(SeriesSegmentMagic))
	if _, err := io.ReadFull(r, magic); err != nil {
		return hdr, err
	} else if !bytes.Equal([]byte(SeriesSegmentMagic), magic) {
		return hdr, ErrInvalidSeriesSegment
	}

	// Read version.
	if err := binary.Read(r, binary.BigEndian, &hdr.Version); err != nil {
		return hdr, err
	}

	return hdr, nil
}

// WriteTo writes the header to w.
func (hdr *SeriesSegmentHeader) WriteTo(w io.Writer) (n int64, err error) {
	var buf bytes.Buffer
	buf.WriteString(SeriesSegmentMagic)
	binary.Write(&buf, binary.BigEndian, hdr.Version)
	return buf.WriteTo(w)
}

func ReadSeriesEntry(data []byte) (flag uint8, id uint64, key []byte, sz int64) {
	// If flag byte is zero then no more entries exist.
	flag, data = uint8(data[0]), data[1:]
	if !IsValidSeriesEntryFlag(flag) {
		return 0, 0, nil, 1
	}

	id, data = binary.BigEndian.Uint64(data), data[8:]
	switch flag {
	case SeriesEntryInsertFlag:
		key, _ = ReadSeriesKey(data)
	}
	return flag, id, key, int64(SeriesEntryHeaderSize + len(key))
}

func AppendSeriesEntry(dst []byte, flag uint8, id uint64, key []byte) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, id)

	dst = append(dst, flag)
	dst = append(dst, buf...)

	switch flag {
	case SeriesEntryInsertFlag:
		dst = append(dst, key...)
	case SeriesEntryTombstoneFlag:
	default:
		panic(fmt.Sprintf("unreachable: invalid flag: %d", flag))
	}
	return dst
}

// IsValidSeriesEntryFlag returns true if flag is valid.
func IsValidSeriesEntryFlag(flag byte) bool {
	switch flag {
	case SeriesEntryInsertFlag, SeriesEntryTombstoneFlag:
		return true
	default:
		return false
	}
}
