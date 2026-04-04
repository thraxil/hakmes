package main

import (
	"bufio"
	"encoding/json"
	"io"
	"log"
	"strings"
)

type site struct {
	CaskBase  string
	ChunkSize int64
	store     MetadataStore
}

func newSite(caskBase string, chunkSize int64, store MetadataStore) *site {
	return &site{CaskBase: caskBase, ChunkSize: chunkSize, store: store}
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
	s.store.EnsureBuckets()
}

func (s site) Add(p postResponse) {
	s.store.Add(p)
}

func (s site) Get(k *key) (postResponse, bool) {
	return s.store.Get(k)
}

func (s site) All(fn func(postResponse)) error {
	return s.store.All(fn)
}

func (s site) Ingest(r io.Reader) error {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		var pr postResponse
		err := json.Unmarshal(scanner.Bytes(), &pr)
		if err != nil {
			log.Printf("error unmarshaling line: %v", err)
			continue
		}
		k, err := keyFromString(pr.Key)
		if err != nil {
			log.Printf("invalid key %s: %v", pr.Key, err)
			continue
		}
		_, found := s.Get(k)
		if found {
			continue
		}
		s.Add(pr)
	}
	return scanner.Err()
}
