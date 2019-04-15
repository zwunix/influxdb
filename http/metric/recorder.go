package metric

import (
	"context"

	"github.com/influxdata/influxdb"
)

// Recorder records meta-data associated with http requests.
type Recorder interface {
	Record(ctx context.Context, m Metric)
}

// Metric represents the meta data associated with an API request.
// TODO(desa): is there a better name we can use to get rid of the stutter?
type Metric struct {
	OrgID         influxdb.ID
	Endpoint      string
	RequestBytes  int
	ResponseBytes int
	Status        int
}
