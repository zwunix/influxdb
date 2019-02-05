package bolt

import (
	"context"

	bolt "github.com/coreos/bbolt"
	platform "github.com/influxdata/influxdb"
)

const (
	rootBucket = "tasks"
)

var (
	orgByTaskID = []byte("/tasks/v1/org_by_task_id")
)

func (c *Client) initializeTasks(ctx context.Context, tx *bolt.Tx) ([]byte, error) {
	bucket := []byte(rootBucket)

	// create root
	root, err := tx.CreateBucketIfNotExists(bucket)
	if err != nil {
		return nil, err
	}
	// create the buckets inside the root
	_, err = root.CreateBucketIfNotExists(orgByTaskID)
	if err != nil {
		return nil, err
	}

	return bucket, nil
}

// FindTaskOrganizationID returns the ID of the organization a task belongs to.
func (c *Client) FindTaskOrganizationID(ctx context.Context, id platform.ID) (platform.ID, error) {
	var orgID platform.ID

	encodedID, err := id.Encode()
	if err != nil {
		return 0, err
	}

	err = c.db.View(func(tx *bolt.Tx) error {
		// get root bucket
		b := tx.Bucket(c.taskBucket)
		if err := orgID.Decode(b.Bucket(orgByTaskID).Get(encodedID)); err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return 0, err
	}

	return orgID, nil
}
