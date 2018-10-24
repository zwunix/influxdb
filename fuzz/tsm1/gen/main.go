package main

import (
	"log"
	"math/rand"
	"os"

	"github.com/dvyukov/go-fuzz/gen"
	"github.com/influxdata/influxdb/tsdb/engine/tsm1"
)

func main() {
	for _, n := range []int{0, 5, 50, 100, 550, 1000} {
		src := make([]float64, n)
		for i := 0; i < n; i++ {
			src[i] = rand.Float64()
		}
		buf, err := tsm1.FloatArrayEncodeAll(src, nil)
		if err != nil {
			log.Fatalf("FloatArrayEncodeAll failed: %q", err)
			os.Exit(1)
		}

		gen.Emit(buf, nil, true)
	}
}
