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

func (s Site) CaskRetrieveBase() string {
	if strings.HasSuffix(s.CaskBase, "/") {
		return s.CaskBase + "file/"
	} else {
		return s.CaskBase + "/file/"
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

func (s Site) Get(k *Key) (postResponse, bool) {
	var v []byte
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Files"))
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
