package main

type Site struct {
	CaskBase  string
	ChunkSize int64
}

func NewSite(cask_base string, chunk_size int64) *Site {
	return &Site{CaskBase: cask_base, ChunkSize: chunk_size}
}
