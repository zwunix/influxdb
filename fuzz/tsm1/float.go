package tsm1

import (
	"github.com/influxdata/influxdb/tsdb/engine/tsm1"
)

func Fuzz(data []byte) int {
	if _, err := tsm1.FloatArrayDecodeAll(data, nil); err != nil {
		return 0
	}
	return 1
}
