package main

import (
	"errors"
	"fmt"
	"io"
	"log"
)

type hakmesReader struct {
	s        *site
	metadata postResponse
	offset   int64

	// Cache the last fetched chunk to avoid redundant Cask requests
	lastChunkIndex int
	lastChunkData  []byte
}

func newHakmesReader(s *site, metadata postResponse) *hakmesReader {
	return &hakmesReader{
		s:              s,
		metadata:       metadata,
		offset:         0,
		lastChunkIndex: -1,
	}
}

func (h *hakmesReader) Verify() error {
	if len(h.metadata.Chunks) == 0 {
		return nil
	}
	_, err := h.getChunk(0)
	return err
}

func (h *hakmesReader) Read(p []byte) (n int, err error) {
	if h.offset >= h.metadata.Size {
		return 0, io.EOF
	}

	n, err = h.ReadAt(p, h.offset)
	h.offset += int64(n)
	return n, err
}

func (h *hakmesReader) ReadAt(p []byte, off int64) (n int, err error) {
	if off < 0 || off >= h.metadata.Size {
		return 0, io.EOF
	}

	remaining := h.metadata.Size - off
	if int64(len(p)) > remaining {
		p = p[:remaining]
	}

	totalRead := 0
	for totalRead < len(p) {
		currentOff := off + int64(totalRead)
		chunkIdx := int(currentOff / h.s.ChunkSize)
		offsetInChunk := currentOff % h.s.ChunkSize

		if chunkIdx >= len(h.metadata.Chunks) {
			break
		}

		chunkData, err := h.getChunk(chunkIdx)
		if err != nil {
			return totalRead, err
		}

		n := copy(p[totalRead:], chunkData[offsetInChunk:])
		totalRead += n
	}

	return totalRead, nil
}

func (h *hakmesReader) getChunk(idx int) ([]byte, error) {
	if h.lastChunkIndex == idx {
		return h.lastChunkData, nil
	}

	chunkKey := h.metadata.Chunks[idx]
	data, err := getChunkFromCask(chunkKey, h.s.CaskRetrieveBase())
	if err != nil {
		log.Printf("error fetching chunk %d (%s): %v", idx, chunkKey, err)
		return nil, fmt.Errorf("cask retrieve failed: %w", err)
	}

	h.lastChunkIndex = idx
	h.lastChunkData = data
	return data, nil
}

func (h *hakmesReader) Seek(offset int64, whence int) (int64, error) {
	var newOffset int64
	switch whence {
	case io.SeekStart:
		newOffset = offset
	case io.SeekCurrent:
		newOffset = h.offset + offset
	case io.SeekEnd:
		newOffset = h.metadata.Size + offset
	default:
		return 0, errors.New("invalid whence")
	}

	if newOffset < 0 {
		return 0, errors.New("negative offset")
	}
	h.offset = newOffset
	return h.offset, nil
}

// We need to move getChunkFromCask or make it accessible if it's in views.go
// It is currently in views.go and is not exported. Since everything is in package main,
// it should be fine.
