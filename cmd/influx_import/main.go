package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	itsdb "github.com/influxdata/influxdb/tsdb"
	itsm1 "github.com/influxdata/influxdb/tsdb/engine/tsm1"
	"github.com/influxdata/platform"
	"github.com/influxdata/platform/kit/cli"
	"github.com/influxdata/platform/models"
	"github.com/influxdata/platform/pkg/bufio"
	"github.com/influxdata/platform/storage"
	"github.com/influxdata/platform/tsdb"
)

const (
	// Name of internal database. Not imported.
	internalDBName = "_internal"

	// InfluxDB 1.x TSM index entry size.
	indexEntrySize1x = 0 +
		8 + // Block min time
		8 + // Block max time
		8 + // Offset of block
		4 // Size in bytes of block

		// tsm1 key field separator.
	keyFieldSeparator1x = "#!~#"
)

// General options
var (
	forceImport bool

	stdout     = os.Stdout
	stderr     = os.Stderr
	verbose    bool
	verboseOut = ioutil.Discard
	home       = os.Getenv("HOME")
)

// Details of 2.x InfluxDB data.
var (
	platformBase string

	toOrgID    string
	toBucketID string
	bucketID   *platform.ID
	orgID      *platform.ID
)

// Details of 1.x InfluxDB data.
var impBase string

// Filters for determining which shards to import.
var (
	impDB string
	impRP string
)

// Time range of TSM data to import. Applied to all shards.
var (
	impFrom     string
	impTo       string
	impFromNano int64 = math.MinInt64
	impToNano   int64 = math.MaxInt64
)

func main() {
	if home == "" {
		home = "~"
	}

	prog := &cli.Program{
		Name: "influx_import",
		Run:  run,
		Opts: []cli.Opt{
			{
				DestP:   &toOrgID,
				Flag:    "org",
				Default: "",
				Desc:    "Organisation ID to import into. Required.",
			},
			{
				DestP:   &toBucketID,
				Flag:    "bucket",
				Default: "",
				Desc:    "Bucket ID to import into. Required.",
			},
			{
				DestP:   &forceImport,
				Flag:    "force",
				Default: false,
				Desc:    "Setting force to true will instruct the tool to import shards it thinks might still be hot.",
			},
			{
				DestP:   &verbose,
				Flag:    "v",
				Default: false,
				Desc:    "Specify verbose progress output.",
			},
			{
				DestP:   &platformBase,
				Flag:    "v2-path",
				Default: "",
				Desc:    "Specify base path of InfluxDB 2.x. Defaults to ~/.influxdbv2",
			},
			{
				DestP:   &impBase,
				Flag:    "v1-path",
				Default: "",
				Desc:    "Specify base path of InfluxDB 1.x. Defaults to ~/.influxdb",
			},
			{
				DestP:   &impDB,
				Flag:    "import-db",
				Default: "",
				Desc:    "Specify database to import. By default all databases except _internal imported.",
			},
			{
				DestP:   &impRP,
				Flag:    "import-rp",
				Default: "",
				Desc:    "Specify retention policy to import. By default all retention policies of imported databases are imported.",
			},

			{
				DestP:   &impFrom,
				Flag:    "from",
				Default: nil,
				Desc:    "Minimum time of TSM data to import.",
			},
			{
				DestP:   &impTo,
				Flag:    "to",
				Default: nil,
				Desc:    "Maximum time of TSM data to import.",
			},
		},
	}

	cmd := cli.NewCommand(prog)
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func shardHotError(pth string) error {
	return fmt.Errorf("shard at path %q is not cold. Cannot import", pth)
}

func run() error {
	if platformBase == "" {
		platformBase = filepath.Join(home, ".influxdbv2")
	}

	if impBase == "" {
		impBase = filepath.Join(home, ".influxdb")
	}

	var err error
	bucketID, err = platform.IDFromString(toBucketID)
	if err != nil {
		return fmt.Errorf("invalid bucket ID: %v", err)
	}

	orgID, err = platform.IDFromString(toOrgID)
	if err != nil {
		return fmt.Errorf("invalid bucket ID: %v", err)
	}

	if impFrom != "" {
		t, err := time.Parse(time.RFC3339Nano, impFrom)
		if err != nil {
			return err
		}
		impFromNano = t.UnixNano()
	}

	if impTo != "" {
		t, err := time.Parse(time.RFC3339Nano, impTo)
		if err != nil {
			return err
		}
		impToNano = t.UnixNano()
	}

	if err := ProcessShards(); err != nil {
		fmt.Fprintf(stderr, "influx_import exited with error: %v", err)
		return err
	}
	return nil
}

// ProcessShards processes all shards in the data directory being imported, applying
// any database or retention policy filters.
func ProcessShards() error {
	var toProcessPaths []string
	err := walkShardDirs(filepath.Join(impBase, "data"), func(db string, rp string, path string) error {
		if db == internalDBName {
			return nil // Don't import TSM data from _internal.
		}

		// A database or retention policy filter has been specified and this
		// shard path does not match it.
		if (impDB != "" && db != impDB) || (impRP != "" && rp != impRP) {
			return nil
		}

		toProcessPaths = append(toProcessPaths, path)
		return nil
	})
	if err != nil {
		return err
	}

	now := time.Now()
	// TODO(edd): Parallelise this.
	for _, pth := range toProcessPaths {
		if err := ProcessShard(pth); err != nil {
			return err
		}
		fmt.Fprintf(stdout, "Processed shard %s in %v\n", pth, time.Since(now))
	}
	return nil
}

// ProcessShard checks the TSM data at the provided shard path is fully compacted,
// and then proceeds to send it to the provided io.Writer.
func ProcessShard(pth string) error {
	// Check full compaction
	// Stream TSM file into new TSM file
	//	- full blocks can be copied over if the time range matches.
	//  - partial blocks need to be decoded and written out up to the timestamp.
	//  - Index needs to include any entries that have at least one block in the
	//    time range.
	e := itsm1.NewEngine(0, nil, pth, "", nil, itsdb.NewEngineOptions())
	defer e.Close()
	if !forceImport && !e.IsIdle() {
		// Shard is not cold.
		return shardHotError(pth)
	}

	// Check for `tmp` files and identify TSM file(s) path.
	var tsmPaths []string // Possible a fully compacted shard has multiple TSM files.
	filepath.Walk(pth, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if strings.HasSuffix(p, ".tsm.tmp") {
			return fmt.Errorf("tmp TSM file detected at %q — aborting shard import", p)
		} else if ext := filepath.Ext(p); ext == "."+itsm1.TSMFileExtension {
			tsmPaths = append(tsmPaths, p)
		}

		// All other non-tsm shard contents are skipped.
		return nil
	})

	if len(tsmPaths) == 0 {
		return fmt.Errorf("no tsm data found at %q", pth)
	}

	for _, tsmp := range tsmPaths {
		fd, err := os.Open(tsmp)
		if err != nil {
			return err
		}

		r, err := itsm1.NewTSMReader(fd)
		if err != nil {
			fd.Close()
			return err
		}

		tsmMin, tsmMax := r.TimeRange()
		if !r.OverlapsTimeRange(impFromNano, impToNano) {
			fmt.Fprintf(verboseOut, "Skipping out-of-range (min-time: %v, max-time: %v) TSM file at path %q\n", time.Unix(0, tsmMin), time.Unix(0, tsmMax), tsmp)
			r.Close()
			fd.Close()
			continue
		}

		now := time.Now()
		// Entire TSM file is within the imported time range; copy all block data
		// and rewrite TSM index.
		if tsmMin >= impFromNano && tsmMax <= impToNano {
			if err := processTSMFileFast(r, fd); err != nil {
				r.Close()
				fd.Close()
				return fmt.Errorf("error processing TSM file %q: %v", tsmp, err)
			}
			continue
		}

		if err := processTSMFile(r); err != nil {
			return fmt.Errorf("error processing TSM file %q: %v", tsmp, err)
		}
		fmt.Fprintf(verboseOut, "Processed TSM file: %s in %v\n", tsmp, time.Since(now))
	}
	return nil
}

func processTSMFile(r *itsm1.TSMReader) error {
	panic("not yet implemented")
}

// processTSMFileFast processes all blocks in the provided TSM file, because all
// TSM data in the file is within the time range being imported.
func processTSMFileFast(r *itsm1.TSMReader, fi *os.File) (err error) {
	fo, fopath, err := writeCloser(r.Path())
	if err != nil {
		return err
	}

	// If there is no error writing the file then remove the .tmp extension.
	defer func() {
		fo.Close()
		if err == nil {
			_ = fopath
			// Rename import file.
			if err2 := os.Rename(fopath, strings.TrimSuffix(fopath, ".tmp")); err2 != nil {
				err = err2
				return
			}
		}
	}()

	// Determine end of block by reading index offset.
	indexOffset, err := indexOffset(fi)
	if err != nil {
		return err
	}

	// Return to beginning of file and copy the header and all block data to
	// new file.
	if _, err = fi.Seek(0, io.SeekStart); err != nil {
		return err
	}

	n, err := io.CopyN(fo, fi, int64(indexOffset))
	if err != nil {
		return err
	} else if n != int64(indexOffset) {
		return fmt.Errorf("short read of block data. Read %d/%d bytes", n, indexOffset)
	}

	// Rewrite TSM index into new file.

	var tagsBuf models.Tags // Buffer to re-use for each series.
	var oldM []byte
	var seriesKeyBuf []byte // Buffer to re-use for new series key.

	// TODO(edd/jeff): I'm not sold on us storing non-ascii measurements.
	newM := tsdb.EncodeName(*orgID, *bucketID)

	for i := 0; i < r.KeyCount(); i++ {
		tsmKey, typ, entries := r.Key(i, nil) // TODO(edd): see if we can pool this []IndexEntries.

		// Parse out the key.
		skey, fkey := itsm1.SeriesAndFieldFromCompositeKey(tsmKey)
		oldM, tagsBuf = models.ParseKeyBytesWithTags(skey, tagsBuf)

		// Rewrite the measurement and tags.
		seriesKey := rewriteSeriesKey(oldM, newM[:], fkey, tagsBuf, seriesKeyBuf)
		// Write the entries for the key back into new file.
		if err := writeIndexEntries(fo, seriesKey, fkey, typ, entries); err != nil {
			return err
		}
	}

	// Write Footer.
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], indexOffset)
	_, err = fo.Write(buf[:])
	return err
}

var tmpTags = make([]models.Tag, 2)

// rewriteSeriesKey takes a 1.x index/seriesfile series key and rewrites it to
// a 2.x format by including the `_m`, `_f` tag pairs and a new measurement
// comprising the org/bucket id.
func rewriteSeriesKey(oldM, newM []byte, fkey []byte, tags models.Tags, buf []byte) []byte {
	var needSort bool
	if len(tags) > 0 && bytes.Compare(tags[0].Key, tsdb.MeasurementTagKeyBytes) < 0 {
		needSort = true // Existing first tag is < tags we're injecting in.
	}

	// Add the `_f` and `_m` tags.
	tags = append(tags, tmpTags...) // Make room for two new tags.
	copy(tags[2:], tags)            // Copy existing tags down.
	tags[0] = models.NewTag(tsdb.FieldKeyTagKeyBytes, fkey)
	tags[1] = models.NewTag(tsdb.MeasurementTagKeyBytes, oldM)

	if needSort {
		sort.Sort(tags)
	}

	// Create a new series key using the new measurement name and tags.
	return models.AppendMakeKey(buf, newM, tags)
}

// indexOffset returns the offset to the TSM index of the provided file, which
// must be a valid TSM file.
func indexOffset(fd *os.File) (uint64, error) {
	_, err := fd.Seek(-8, io.SeekEnd)
	if err != nil {
		return 0, err
	}

	buf := make([]byte, 8)
	n, err := fd.Read(buf)
	if err != nil {
		return 0, err
	} else if n != 8 {
		return 0, fmt.Errorf("short read of index offset on file %q", fd.Name())
	}

	return binary.BigEndian.Uint64(buf), nil
}

var keyFieldSeparator1xBytes = []byte(keyFieldSeparator1x)

func writeIndexEntries(w io.Writer, key []byte, fkey []byte, typ byte, entries []itsm1.IndexEntry) error {
	// The key is not in a TSM format. Convert it to TSM format.
	key = append(key, keyFieldSeparator1xBytes...)
	key = append(key, fkey...)

	var buf [5 + indexEntrySize1x]byte
	binary.BigEndian.PutUint16(buf[0:2], uint16(len(key)))
	buf[2] = typ
	binary.BigEndian.PutUint16(buf[3:5], uint16(len(entries)))

	// Write the key length.
	if _, err := w.Write(buf[0:2]); err != nil {
		return fmt.Errorf("write: writer key length error: %v", err)
	}

	// Write the key.
	if _, err := w.Write(key); err != nil {
		return fmt.Errorf("write: writer key error: %v", err)
	}

	// Write the block type and count
	if _, err := w.Write(buf[2:5]); err != nil {
		return fmt.Errorf("write: writer block type and count error: %v", err)
	}

	// Write each index entry for all blocks for this key
	for _, entry := range entries {
		entry.AppendTo(buf[5:])
		n, err := w.Write(buf[5:])
		if err != nil {
			return err
		} else if n != indexEntrySize1x {
			return fmt.Errorf("incorrect number of bytes written for entry: %d", n)
		}
	}
	return nil
}

// writeCloser initialises an io.WriteCloser for writing a new TSM file.
func writeCloser(pth string) (io.WriteCloser, string, error) {
	// TODO(edd): Handle creating a network descriptor...
	name := filepath.Base(pth) + ".import.tmp"
	dir := filepath.Join(platformBase, "engine", storage.DefaultEngineDirectoryName)
	if err := os.MkdirAll(dir, 0777); err != nil { // TODO(edd): what permissions do we use?
		return nil, "", err
	}

	fullPath := filepath.Join(dir, name)
	fd, err := os.Create(fullPath)
	if err != nil {
		return nil, "", err
	}

	w := bufio.NewWriterSize(fd, 1<<20)
	return w, fullPath, nil
}

func walkShardDirs(root string, fn func(db, rp, path string) error) error {
	type location struct {
		db, rp, path string
		id           int
	}

	dirs := map[string]location{}
	if err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}
		if filepath.Ext(info.Name()) == "."+itsm1.TSMFileExtension {
			shardDir := filepath.Dir(path)

			id, err := strconv.Atoi(filepath.Base(shardDir))
			if err != nil || id < 1 {
				return fmt.Errorf("not a valid shard dir: %v", shardDir)
			}

			absPath, err := filepath.Abs(path)
			if err != nil {
				return err
			}
			parts := strings.Split(absPath, string(filepath.Separator))
			db, rp := parts[len(parts)-4], parts[len(parts)-3]
			dirs[shardDir] = location{db: db, rp: rp, id: id, path: shardDir}
			return nil
		}
		return nil
	}); err != nil {
		return err
	}

	dirsSlice := make([]location, 0, len(dirs))
	for _, v := range dirs {
		dirsSlice = append(dirsSlice, v)
	}

	sort.Slice(dirsSlice, func(i, j int) bool {
		return dirsSlice[i].id < dirsSlice[j].id
	})

	for _, shard := range dirs {
		if err := fn(shard.db, shard.rp, shard.path); err != nil {
			return err
		}
	}
	return nil
}
