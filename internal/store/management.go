package store

import (
	"fmt"
	"os"
	"time"

	bolt "go.etcd.io/bbolt"
)

type Statistics struct {
	Readings      int   `json:"readings"`
	Alerts        int   `json:"alerts"`
	Batches       int   `json:"batches"`
	Reports       int   `json:"reports"`
	Verifications int   `json:"verifications"`
	FileBytes     int64 `json:"file_bytes"`
}

func (store *Store) Statistics() (Statistics, error) {
	var result Statistics
	err := store.View(func(tx *bolt.Tx) error {
		result.Readings = tx.Bucket([]byte("sensor_readings")).Stats().KeyN
		result.Alerts = tx.Bucket([]byte("alert_summaries")).Stats().KeyN
		result.Batches = tx.Bucket([]byte("snapshot_batches")).Stats().KeyN
		result.Reports = tx.Bucket([]byte("throughput_reports")).Stats().KeyN
		result.Verifications = tx.Bucket([]byte("verification_runs")).Stats().KeyN
		return nil
	})
	if err != nil {
		return result, err
	}
	info, err := os.Stat(store.path)
	if err == nil {
		result.FileBytes = info.Size()
	}
	return result, nil
}
func (store *Store) DeleteMinute(minute time.Time) error {
	minuteKey := []byte(minute.UTC().Truncate(time.Minute).Format(time.RFC3339))
	return store.Update(func(tx *bolt.Tx) error {
		id := copyBytes(tx.Bucket([]byte("minute_index")).Get(minuteKey))
		if id == nil {
			return ErrNotFound
		}
		if err := tx.Bucket([]byte("snapshot_batches")).Delete(id); err != nil {
			return err
		}
		return tx.Bucket([]byte("minute_index")).Delete(minuteKey)
	})
}
func (store *Store) Backup(path string) error {
	if path == "" {
		return fmt.Errorf("backup path is required")
	}
	return store.View(func(tx *bolt.Tx) error { return tx.CopyFile(path, 0o600) })
}
func (store *Store) Check() error {
	return store.View(func(tx *bolt.Tx) error {
		for _, bucket := range buckets {
			if tx.Bucket(bucket) == nil {
				return fmt.Errorf("missing bucket %s", bucket)
			}
		}
		for issue := range tx.Check() {
			if issue != nil {
				return issue
			}
		}
		return nil
	})
}
