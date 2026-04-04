package main

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/boltdb/bolt"
)

// MetadataStore defines the interface for backend storage operations.
type MetadataStore interface {
	EnsureBuckets()
	Add(p postResponse)
	Get(k *key) (postResponse, bool)
	All(fn func(postResponse)) error
}

// BoltStore is the BoltDB implementation of MetadataStore.
type BoltStore struct {
	db *bolt.DB
}

func NewBoltStore(db *bolt.DB) *BoltStore {
	return &BoltStore{db: db}
}

func (s *BoltStore) EnsureBuckets() {
	err := s.db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("Files"))
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
}

func (s *BoltStore) Add(p postResponse) {
	data, err := json.Marshal(p)
	if err != nil {
		log.Println("error marshaling to json")
		log.Println(err.Error())
		return
	}
	err = s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Files"))
		err := b.Put([]byte(p.Key), data)
		return err
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Println("wrote it...")
}

func (s *BoltStore) Get(k *key) (postResponse, bool) {
	var v []byte
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Files"))
		if b == nil {
			return nil
		}
		v = b.Get([]byte(k.String()))
		return nil
	})
	if err != nil {
		log.Println("error retrieving entry")
		log.Println(err.Error())
		return postResponse{}, false
	}
	if v == nil {
		return postResponse{}, false
	}
	var pr postResponse
	err = json.Unmarshal(v, &pr)
	if err != nil {
		log.Println("error unmarshaling json")
		log.Println(err.Error())
		return pr, false
	}
	return pr, true
}

func (s *BoltStore) All(fn func(postResponse)) error {
	return s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Files"))
		if b == nil {
			return nil
		}
		return b.ForEach(func(k, v []byte) error {
			var pr postResponse
			err := json.Unmarshal(v, &pr)
			if err != nil {
				return err
			}
			fn(pr)
			return nil
		})
	})
}

// MemoryStore is an in-memory implementation of MetadataStore for testing.
type MemoryStore struct {
	data map[string]postResponse
	mu   sync.RWMutex
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		data: make(map[string]postResponse),
	}
}

func (m *MemoryStore) EnsureBuckets() {
	// No-op for in-memory store
}

func (m *MemoryStore) Add(p postResponse) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[p.Key] = p
	log.Println("wrote it...")
}

func (m *MemoryStore) Get(k *key) (postResponse, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.data[k.String()]
	return p, ok
}

func (m *MemoryStore) All(fn func(postResponse)) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, p := range m.data {
		fn(p)
	}
	return nil
}
