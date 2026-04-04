package main

import (
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
