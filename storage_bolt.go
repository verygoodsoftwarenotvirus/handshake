package handshake

import (
	"bytes"
	"errors"
	"fmt"

	bolt "go.etcd.io/bbolt"
)

// NewBoltStorage takes StorageOptions as an argument and returns a reference to a BoltDB
// based implementation of the Storage interface.
func newBoltStorage(cfg Config, opts StorageOptions) (boltStorage, error) {
	tlb := defaultTLB
	fp := defaultBoltFilePath
	if opts.FilePath != "" {
		fp = opts.FilePath
	}
	db, err := bolt.Open(fp, 0666, nil)
	if err != nil {
		return boltStorage{}, err
	}

	// ensure that top level bucket exists
	if err := db.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists([]byte(tlb)); err != nil {
			return fmt.Errorf("error creating bucket: %s", err)
		}
		return nil
	}); err != nil {
		return boltStorage{}, err
	}

	// ensure that Config exists, and if not, initialize GlobalConfig
	if err := db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(tlb))
		blob := b.Get([]byte(globalConfigKey))
		if blob == nil {
			return b.Put([]byte(globalConfigKey), cfg.ToJSON())
		}
		return nil
	}); err != nil {
		return boltStorage{}, err
	}

	return boltStorage{db: db, tlb: tlb}, nil
}

// BoltStorage is a struct that conforms to the Storage interface for using
// BoltDB. DB is a reference to a boltDB instance and TLB stands for "top level bucket"
type boltStorage struct {
	db  *bolt.DB
	tlb string
}

// Get takes a key string and returns a byte slice or error from a BoltStorage struct. Get
// get retruns and empty byte slice if no key is found and/or the byte slice is blank. An error
// is returned if the key invalid in formatting, it is too long, or there is an underlying issue
// with boltDB
func (s boltStorage) Get(key string) (value []byte, err error) {
	err = s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(s.tlb))
		value = b.Get([]byte(key))
		return nil
	})
	return value, err
}

// Set takes a key string and value byte slice returns an error from a BoltStorage struct.
// Set treats both create and updates the same. Errors are returned if the key has invalid syntax
// and if key or value are too long.
func (s boltStorage) Set(key string, value []byte) (string, error) {
	return key, s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(s.tlb))
		return b.Put([]byte(key), value)
	})
}

// Delete takes a key string and deletes item, if it exists in storage, returns an error from a BoltStorage struct.
func (s boltStorage) Delete(key string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(s.tlb))
		return b.Delete([]byte(key))
	})
}

// List takes a path and returns a slice of key paths formatted as strings or an error.
func (s boltStorage) List(path string) (keys []string, err error) {
	p := []byte(path)
	err = s.db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket([]byte(s.tlb)).Cursor()
		for k, _ := c.Seek(p); k != nil && bytes.HasPrefix(k, p); k, _ = c.Next() {
			keys = append(keys, string(k))
		}
		return nil
	})
	return keys, err
}

// share is not configured on BoltStorage, since it is private storage.
// Therefore it returns an empty struct.
func (s boltStorage) share() (peerStorage, error) {
	return peerStorage{}, errors.New("this storage does not support shared configs")
}

// share is not configured on BoltStorage, since it is private storage.
// Therefore it returns an empty struct.
func (s boltStorage) export() (storageConfig, error) {
	return storageConfig{}, errors.New("this storage does not support exporting configs")
}

// Close is used to close the Bolt DB engine and returns an error
func (s boltStorage) Close() error {
	return s.db.Close()
}
