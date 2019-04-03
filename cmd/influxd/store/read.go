package store

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/gogo/protobuf/types"
	influxdb2 "github.com/influxdata/influxdb/query/stdlib/influxdata/influxdb"
	"github.com/influxdata/influxdb/storage/reads/datatypes"
	"github.com/influxdata/influxdb/storage/readservice"
	"github.com/influxdata/influxdb/tsdb/cursors"
	"github.com/spf13/cobra"
)

var readCommand = &cobra.Command{
	Use:  "read",
	RunE: readFE,
}

var readFlags struct {
	orgBucket
}

func init() {
	readFlags.orgBucket.AddFlags(readCommand)
	RootCommand.AddCommand(readCommand)
}

func readFE(_ *cobra.Command, _ []string) error {
	ctx := context.Background()

	engine, err := newEngine(ctx)
	if err != nil {
		return err
	}
	defer engine.Close()

	store := readservice.NewStore(engine)

	orgID, bucketID, err := readFlags.OrgBucketID()
	if err != nil {
		return err
	}

	var req datatypes.ReadRequest
	source, _ := store.GetSource(influxdb2.ReadSpec{OrganizationID: orgID, BucketID: bucketID})
	if any, err := types.MarshalAny(source); err != nil {
		return err
	} else {
		req.ReadSource = any
	}

	stop := storeFlags.profile.Start()
	defer stop()

	rs, err := store.Read(ctx, &req)
	if err != nil {
		return err
	}
	defer rs.Close()

	points := 0
	series := 0

	fmt.Println("Consuming data...")

	start := time.Now()
	defer func() {
		dur := time.Since(start)
		tw := tabwriter.NewWriter(os.Stdout, 10, 4, 0, ' ', 0)
		fmt.Fprintf(tw, "Series:\t%d\n", series)
		fmt.Fprintf(tw, "Points:\t%d\n", points)
		fmt.Fprintf(tw, "Time:\t%0.0fms\n", dur.Seconds()*1000)
		fmt.Fprintf(tw, "Series/s:\t%0.3f\n", float64(series)/dur.Seconds())
		fmt.Fprintf(tw, "Points/s:\t%0.3f\n", float64(points)/dur.Seconds())
		tw.Flush()
	}()

	for rs.Next() {
		series += 1
		cur := rs.Cursor()
		switch tcur := cur.(type) {
		case cursors.FloatArrayCursor:
			ts := tcur.Next()
			for ts.Len() > 0 {
				points += ts.Len()
				ts = tcur.Next()
			}

		case cursors.IntegerArrayCursor:
			ts := tcur.Next()
			for ts.Len() > 0 {
				points += ts.Len()
				ts = tcur.Next()
			}

		case cursors.UnsignedArrayCursor:
			ts := tcur.Next()
			for ts.Len() > 0 {
				points += ts.Len()
				ts = tcur.Next()
			}

		case cursors.StringArrayCursor:
			ts := tcur.Next()
			for ts.Len() > 0 {
				points += ts.Len()
				ts = tcur.Next()
			}

		case cursors.BooleanArrayCursor:
			ts := tcur.Next()
			for ts.Len() > 0 {
				points += ts.Len()
				ts = tcur.Next()
			}

		default:
			panic(fmt.Sprintf("unexpected type: %T", cur))
		}
		cur.Close()
	}

	return nil
}
