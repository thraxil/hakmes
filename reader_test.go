package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHakmesReader(t *testing.T) {
	// Mock Cask server
	chunks := [][]byte{
		[]byte("chunk0-"),
		[]byte("chunk1-"),
		[]byte("chunk2"),
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for i, chunk := range chunks {
			key := fmt.Sprintf("sha1:chunk%d", i)
			if r.URL.Path == "/file/"+key+"/" {
				_, _ = w.Write(chunk)
				return
			}
		}
		http.Error(w, "not found", 404)
	}))
	defer ts.Close()

	s := &site{
		CaskBase:  ts.URL,
		ChunkSize: 7, // match chunk size above
	}

	metadata := postResponse{
		Key:  "sha1:file",
		Size: 20,
		Chunks: []string{
			"sha1:chunk0",
			"sha1:chunk1",
			"sha1:chunk2",
		},
	}

	reader := newHakmesReader(s, metadata)

	// Test sequential read
	buf := make([]byte, 20)
	n, err := io.ReadFull(reader, buf)
	if err != nil {
		t.Fatalf("read full failed: %v", err)
	}
	if n != 20 {
		t.Errorf("expected 20 bytes, got %d", n)
	}
	if string(buf) != "chunk0-chunk1-chunk2" {
		t.Errorf("unexpected content: %q", string(buf))
	}

	// Test Seek and Read
	_, err = reader.Seek(7, io.SeekStart)
	if err != nil {
		t.Fatalf("seek failed: %v", err)
	}
	_, err = reader.Read(buf[:7])
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if string(buf[:7]) != "chunk1-" {
		t.Errorf("expected 'chunk1-', got %q", string(buf[:7]))
	}

	// Test ReadAt
	bufAt := make([]byte, 6)
	_, err = reader.ReadAt(bufAt, 14)
	if err != nil {
		t.Fatalf("ReadAt failed: %v", err)
	}
	if string(bufAt) != "chunk2" {
		t.Errorf("expected 'chunk2', got %q", string(bufAt))
	}

	// Test cross-chunk read
	_, err = reader.Seek(5, io.SeekStart)
	if err != nil {
		t.Fatalf("seek failed: %v", err)
	}
	crossBuf := make([]byte, 4)
	_, err = reader.Read(crossBuf)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if string(crossBuf) != "0-ch" {
		t.Errorf("expected '0-ch', got %q", string(crossBuf))
	}
}
