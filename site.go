package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/boltdb/bolt"
)

type Site struct {
	CaskBase  string
	ChunkSize int64
	db        *bolt.DB
}

func NewSite(cask_base string, chunk_size int64, db *bolt.DB) *Site {
	return &Site{CaskBase: cask_base, ChunkSize: chunk_size, db: db}
}

func (s Site) CaskPostURL() string {
	if strings.HasSuffix(s.CaskBase, "/") {
		return s.CaskBase
	} else {
		return s.CaskBase + "/"
	}
}

func (s Site) EnsureBuckets() {
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

func (s Site) Add(p postResponse) {
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
