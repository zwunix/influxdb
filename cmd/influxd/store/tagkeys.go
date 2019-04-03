package store

import (
	"bytes"
	"context"
	"fmt"

	"github.com/influxdata/influxdb/models"
	"github.com/spf13/cobra"
)

var tagKeysCommand = &cobra.Command{
	Use:  "tag-keys",
	RunE: tagKeysFE,
}

var tagKeysFlags struct {
	orgBucket
}

func init() {
	tagKeysFlags.orgBucket.AddFlags(tagKeysCommand)
	RootCommand.AddCommand(tagKeysCommand)
}

func tagKeysFE(_ *cobra.Command, _ []string) error {
	ctx := context.Background()

	engine, err := newEngine(ctx)
	if err != nil {
		return err
	}
	defer engine.Close()

	name, err := tagKeysFlags.Name()
	if err != nil {
		return err
	}

	itr := engine.CreateTagKeysIterator(ctx, name, models.MinNanoTime, models.MaxNanoTime, nil)
	defer itr.Close()
	for {
		if buf, err := itr.Next(); err != nil {
			return err
		} else if len(buf) == 0 {
			break
		} else {
			if len(buf) == 1 {
				if bytes.Equal(buf, models.MeasurementTagKeyBytes) {
					fmt.Println("_m")
					continue
				} else if bytes.Equal(buf, models.FieldKeyTagKeyBytes) {
					fmt.Println("_f")
					continue
				}
			}
			fmt.Println(string(buf))
		}
	}

	return nil
}
