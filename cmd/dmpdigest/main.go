package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"runtime/pprof"
	"sort"
	"time"

	"github.com/influxdata/influxdb/tsdb/engine/tsm1"
)

var (
	path       string
	verbose    bool
	keys       bool
	keystats   bool
	rfc3339    bool
	ratelimit  bool
	cpuprofile string
)

func main() {
	flag.StringVar(&path, "path", "digest.tsd", "path to the digest file to dump")
	flag.StringVar(&cpuprofile, "cpuprofile", "", "write cpu profile to `file`")
	flag.BoolVar(&keys, "keys", false, "print series keys to stdout")
	flag.BoolVar(&keystats, "keystats", false, "print series key stats")
	flag.BoolVar(&verbose, "v", false, "print all series keys and ranges to stdout")
	flag.BoolVar(&rfc3339, "rfc3339", false, "used with -verbose, timestamps are printed as human readable dates")
	flag.BoolVar(&ratelimit, "ratelimit", false, "limits dmpdigest CPU and I/O usage")

	flag.Parse()

	if cpuprofile != "" {
		f, err := os.Create(cpuprofile)
		check(err)
		check(pprof.StartCPUProfile(f))
		defer pprof.StopCPUProfile()
	}

	f, err := os.Open(path)
	check(err)

	r, err := tsm1.NewDigestReader(f)
	check(err)

	mfest, err := r.ReadManifest()
	check(err)

	b, err := json.Marshal(mfest)
	check(err)

	var (
		rangeCnts []int
		keyLens   []int
	)

	for {
		if ratelimit && len(rangeCnts)%1000 == 0 {
			time.Sleep(20 * time.Millisecond)
		}

		key, tspan, err := r.ReadTimeSpan()
		if err == io.EOF {
			break
		}
		check(err)

		if keys || verbose {
			fmt.Printf("%s\n", key)
		}

		if verbose {
			for i, _ := range tspan.Ranges {
				if rfc3339 {
					min := time.Unix(0, tspan.Ranges[i].Min).Format(time.RFC3339)
					max := time.Unix(0, tspan.Ranges[i].Max).Format(time.RFC3339)
					fmt.Printf("\t%s\t%s\t%d\t%d\n",
						min,
						max,
						tspan.Ranges[i].N,
						tspan.Ranges[i].CRC)
					continue
				}

				fmt.Printf("\t%d\t%d\t%d\t%d\n",
					tspan.Ranges[i].Min,
					tspan.Ranges[i].Max,
					tspan.Ranges[i].N,
					tspan.Ranges[i].CRC)
			}
			fmt.Println("")
		}

		keyLens = append(keyLens, len(key))
		rangeCnts = append(rangeCnts, len(tspan.Ranges))
	}

	sort.Ints(keyLens)
	sort.Ints(rangeCnts)

	var shardSize int64
	for _, e := range mfest.Entries {
		shardSize += e.Size
	}

	totRanges := sum(rangeCnts)

	fmt.Printf("manifest: %s\n", string(b))
	fmt.Printf("tsm files: %d\n", len(mfest.Entries))
	fmt.Printf("shard size: %s\n", humanize(shardSize))
	fmt.Printf("series: %s\n", humanize(int64(len(rangeCnts))))
	fmt.Printf("tot key len: %s\n", humanize(sum(keyLens)))
	fmt.Printf("min key len: %.1f\n", min(keyLens))
	fmt.Printf("max key len: %.1f\n", max(keyLens))
	fmt.Printf("avg key len: %.1f\n", mean(keyLens))
	fmt.Printf("med key len: %.1f\n", median(keyLens))
	fmt.Printf("ranges: %s\n", humanize(totRanges))
	fmt.Printf("tot ranges size: %s\n", humanize(totRanges*22))
	fmt.Printf("min ranges in a series: %.1f\n", min(rangeCnts))
	fmt.Printf("max ranges in a series: %.1f\n", max(rangeCnts))
	fmt.Printf("avg ranges per series: %.1f\n", mean(rangeCnts))
	fmt.Printf("med ranges per series: %.1f\n", median(rangeCnts))
}

func min(a []int) float64 {
	if len(a) == 0 {
		return math.NaN()
	}
	return float64(a[0])
}

func max(a []int) float64 {
	if len(a) == 0 {
		return math.NaN()
	}
	return float64(a[len(a)-1])
}

func sum(a []int) int64 {
	var n int64
	for _, v := range a {
		n += int64(v)
	}
	return n
}

func mean(a []int) float64 {
	if len(a) == 0 {
		return 0.0
	}

	return float64(sum(a)) / float64(len(a))
}

func median(a []int) float64 {
	l := len(a)
	if l == 0 {
		return math.NaN()
	} else if l%2 == 0 {
		i := l/2 - 1
		return float64((a[i] + a[i+1])) / 2.0
	}
	return float64(a[l/2])
}

func humanize(n int64) string {
	if n >= 1000000000 {
		v := float64(n) / 1000000000.0
		return fmt.Sprintf("%.1f G", v)
	} else if n >= 1000000 {
		v := float64(n) / 1000000.0
		return fmt.Sprintf("%.1f M", v)
	} else if n >= 1000 {
		v := float64(n) / 1000.0
		return fmt.Sprintf("%.1f K", v)
	}
	return fmt.Sprintf("%d", n)
}

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
