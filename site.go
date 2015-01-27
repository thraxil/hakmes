package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/boltdb/bolt"
)

type site struct {
	CaskBase  string
	ChunkSize int64
	db        *bolt.DB
}

func newSite(caskBase string, chunkSize int64, db *bolt.DB) *site {
	return &site{CaskBase: caskBase, ChunkSize: chunkSize, db: db}
}

func (s site) CaskPostURL() string {
	if strings.HasSuffix(s.CaskBase, "/") {
		return s.CaskBase
	}
	return s.CaskBase + "/"
}

func (s site) CaskRetrieveBase() string {
	if strings.HasSuffix(s.CaskBase, "/") {
		return s.CaskBase + "file/"
	}
	return s.CaskBase + "/file/"
}

func (s site) EnsureBuckets() {
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

func (s site) Add(p postResponse) {
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

func (s site) Get(k *key) (postResponse, bool) {
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
