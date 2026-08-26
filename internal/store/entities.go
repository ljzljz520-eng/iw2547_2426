package store

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"example.com/roomsnapshot/internal/model"
	bolt "go.etcd.io/bbolt"
)

func putJSON(bucket *bolt.Bucket, key string, value any) error {
	if key == "" {
		return fmt.Errorf("entity key is required")
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return bucket.Put([]byte(key), encoded)
}
func getJSON(bucket *bolt.Bucket, key string, target any) error {
	encoded := bucket.Get([]byte(key))
	if encoded == nil {
		return ErrNotFound
	}
	return json.Unmarshal(copyBytes(encoded), target)
}
func listJSON[T any](bucket *bolt.Bucket) ([]T, error) {
	values := make([]T, 0, bucket.Stats().KeyN)
	err := bucket.ForEach(func(_, encoded []byte) error {
		var value T
		if err := json.Unmarshal(encoded, &value); err != nil {
			return err
		}
		values = append(values, value)
		return nil
	})
	return values, err
}

func (store *Store) SaveReading(value model.SensorReading) error {
	if err := model.ValidateReading(value); err != nil {
		return err
	}
	return store.Update(func(tx *bolt.Tx) error { return putJSON(tx.Bucket([]byte("sensor_readings")), value.ID, value) })
}
func (store *Store) Reading(id string) (model.SensorReading, error) {
	var value model.SensorReading
	err := store.View(func(tx *bolt.Tx) error { return getJSON(tx.Bucket([]byte("sensor_readings")), id, &value) })
	return value, err
}
func (store *Store) ReadingsByMinute(minute time.Time) ([]model.SensorReading, error) {
	values, err := store.ListReadings()
	if err != nil {
		return nil, err
	}
	minute = model.NormalizeMinute(minute)
	result := values[:0]
	for _, value := range values {
		if value.RecordedMinute == minute {
			result = append(result, value)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Sequence < result[j].Sequence })
	return result, nil
}
func (store *Store) ListReadings() ([]model.SensorReading, error) {
	var values []model.SensorReading
	err := store.View(func(tx *bolt.Tx) error {
		var err error
		values, err = listJSON[model.SensorReading](tx.Bucket([]byte("sensor_readings")))
		return err
	})
	return values, err
}

func (store *Store) SaveAlert(value model.AlertSummary) error {
	if err := model.ValidateAlert(value); err != nil {
		return err
	}
	return store.Update(func(tx *bolt.Tx) error { return putJSON(tx.Bucket([]byte("alert_summaries")), value.ID, value) })
}
func (store *Store) Alert(id string) (model.AlertSummary, error) {
	var value model.AlertSummary
	err := store.View(func(tx *bolt.Tx) error { return getJSON(tx.Bucket([]byte("alert_summaries")), id, &value) })
	return value, err
}
func (store *Store) AlertsByMinute(minute time.Time) ([]model.AlertSummary, error) {
	values, err := store.ListAlerts()
	if err != nil {
		return nil, err
	}
	minute = model.NormalizeMinute(minute)
	result := values[:0]
	for _, value := range values {
		if value.RecordedMinute == minute {
			result = append(result, value)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Sequence < result[j].Sequence })
	return result, nil
}
func (store *Store) ListAlerts() ([]model.AlertSummary, error) {
	var values []model.AlertSummary
	err := store.View(func(tx *bolt.Tx) error {
		var err error
		values, err = listJSON[model.AlertSummary](tx.Bucket([]byte("alert_summaries")))
		return err
	})
	return values, err
}

func (store *Store) SaveBatch(value model.SnapshotBatch) error {
	if err := model.ValidateBatch(value); err != nil {
		return err
	}
	return store.Update(func(tx *bolt.Tx) error {
		if err := putJSON(tx.Bucket([]byte("snapshot_batches")), value.ID, value); err != nil {
			return err
		}
		index := tx.Bucket([]byte("minute_index"))
		return index.Put([]byte(value.Minute.Format(time.RFC3339)), []byte(value.ID))
	})
}
func (store *Store) Batch(id string) (model.SnapshotBatch, error) {
	var value model.SnapshotBatch
	err := store.View(func(tx *bolt.Tx) error { return getJSON(tx.Bucket([]byte("snapshot_batches")), id, &value) })
	return value, err
}
func (store *Store) BatchByMinute(minute time.Time) (model.SnapshotBatch, error) {
	var value model.SnapshotBatch
	err := store.View(func(tx *bolt.Tx) error {
		id := tx.Bucket([]byte("minute_index")).Get([]byte(model.NormalizeMinute(minute).Format(time.RFC3339)))
		if id == nil {
			return ErrNotFound
		}
		return getJSON(tx.Bucket([]byte("snapshot_batches")), string(id), &value)
	})
	return value, err
}
func (store *Store) ListBatches() ([]model.SnapshotBatch, error) {
	var values []model.SnapshotBatch
	err := store.View(func(tx *bolt.Tx) error {
		var err error
		values, err = listJSON[model.SnapshotBatch](tx.Bucket([]byte("snapshot_batches")))
		return err
	})
	sort.Slice(values, func(i, j int) bool { return values[i].Minute.Before(values[j].Minute) })
	return values, err
}
func (store *Store) LatestBatch() (model.SnapshotBatch, error) {
	values, err := store.ListBatches()
	if err != nil {
		return model.SnapshotBatch{}, err
	}
	if len(values) == 0 {
		return model.SnapshotBatch{}, ErrNotFound
	}
	return values[len(values)-1], nil
}

func (store *Store) SaveReport(value model.ThroughputReport) error {
	if value.ID == "" || value.BatchSize <= 0 || value.DurationNanos < 0 {
		return fmt.Errorf("invalid throughput report")
	}
	return store.Update(func(tx *bolt.Tx) error { return putJSON(tx.Bucket([]byte("throughput_reports")), value.ID, value) })
}
func (store *Store) ListReports() ([]model.ThroughputReport, error) {
	var values []model.ThroughputReport
	err := store.View(func(tx *bolt.Tx) error {
		var err error
		values, err = listJSON[model.ThroughputReport](tx.Bucket([]byte("throughput_reports")))
		return err
	})
	sort.Slice(values, func(i, j int) bool {
		if values[i].BatchSize == values[j].BatchSize {
			return values[i].CreatedAt.Before(values[j].CreatedAt)
		}
		return values[i].BatchSize < values[j].BatchSize
	})
	return values, err
}
func (store *Store) SaveVerification(value model.VerificationRun) error {
	if value.ID == "" || value.BatchID == "" {
		return fmt.Errorf("invalid verification run")
	}
	return store.Update(func(tx *bolt.Tx) error { return putJSON(tx.Bucket([]byte("verification_runs")), value.ID, value) })
}
func (store *Store) Verification(id string) (model.VerificationRun, error) {
	var value model.VerificationRun
	err := store.View(func(tx *bolt.Tx) error { return getJSON(tx.Bucket([]byte("verification_runs")), id, &value) })
	return value, err
}
