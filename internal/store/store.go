package store

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	bolt "go.etcd.io/bbolt"
)

var ErrNotFound = errors.New("entity not found")
var buckets = [][]byte{[]byte("sensor_readings"), []byte("alert_summaries"), []byte("snapshot_batches"), []byte("throughput_reports"), []byte("verification_runs"), []byte("minute_index"), []byte("metadata")}

type Store struct {
	db   *bolt.DB
	path string
}

func Open(path string) (*Store, error) {
	if path == "" {
		return nil, errors.New("store path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create store directory: %w", err)
	}
	db, err := bolt.Open(path, 0o600, &bolt.Options{Timeout: time.Second, NoGrowSync: true})
	if err != nil {
		return nil, fmt.Errorf("open store: %w", err)
	}
	value := &Store{db: db, path: path}
	if err := value.initialize(); err != nil {
		db.Close()
		return nil, err
	}
	return value, nil
}

func (store *Store) initialize() error {
	return store.db.Update(func(tx *bolt.Tx) error {
		for _, name := range buckets {
			if _, err := tx.CreateBucketIfNotExists(name); err != nil {
				return fmt.Errorf("create bucket %s: %w", name, err)
			}
		}
		metadata := tx.Bucket([]byte("metadata"))
		if metadata.Get([]byte("schema_version")) == nil {
			return metadata.Put([]byte("schema_version"), []byte("1"))
		}
		if string(metadata.Get([]byte("schema_version"))) != "1" {
			return errors.New("unsupported store schema")
		}
		return nil
	})
}
func (store *Store) Close() error {
	if store == nil || store.db == nil {
		return nil
	}
	err := store.db.Close()
	store.db = nil
	return err
}
func (store *Store) Path() string { return store.path }
func (store *Store) View(fn func(*bolt.Tx) error) error {
	if store == nil || store.db == nil {
		return errors.New("store is closed")
	}
	return store.db.View(fn)
}
func (store *Store) Update(fn func(*bolt.Tx) error) error {
	if store == nil || store.db == nil {
		return errors.New("store is closed")
	}
	return store.db.Update(fn)
}
func copyBytes(value []byte) []byte { return append([]byte(nil), value...) }
