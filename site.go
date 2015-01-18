package main

import "strings"

type Site struct {
	CaskBase  string
	ChunkSize int64
}

func NewSite(cask_base string, chunk_size int64) *Site {
	return &Site{CaskBase: cask_base, ChunkSize: chunk_size}
}

func (s Site) CaskPostURL() string {
	if strings.HasSuffix(s.CaskBase, "/") {
		return s.CaskBase
	} else {
		return s.CaskBase + "/"
	}
}
