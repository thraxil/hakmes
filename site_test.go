package main

import (
	"os"
	"testing"

	"github.com/boltdb/bolt"
)

func TestNewSite(t *testing.T) {
	s := newSite("http://example.com", 1024, nil)
	if s.CaskBase != "http://example.com" {
		t.Errorf("s.CaskBase = %q, want %q", s.CaskBase, "http://example.com")
	}
	if s.ChunkSize != 1024 {
		t.Errorf("s.ChunkSize = %d, want %d", s.ChunkSize, 1024)
	}
}

func TestCaskPostURL(t *testing.T) {
	cases := []struct {
		caskBase string
		expected string
	}{
		{"http://example.com", "http://example.com/"},
		{"http://example.com/", "http://example.com/"},
	}

	for _, c := range cases {
		s := &site{CaskBase: c.caskBase}
		if s.CaskPostURL() != c.expected {
			t.Errorf("CaskPostURL(%q) = %q, want %q", c.caskBase, s.CaskPostURL(), c.expected)
		}
	}
}

func TestCaskRetrieveBase(t *testing.T) {
	cases := []struct {
		caskBase string
		expected string
	}{
		{"http://example.com", "http://example.com/file/"},
		{"http://example.com/", "http://example.com/file/"},
	}

	for _, c := range cases {
		s := &site{CaskBase: c.caskBase}
		if s.CaskRetrieveBase() != c.expected {
			t.Errorf("CaskRetrieveBase(%q) = %q, want %q", c.caskBase, s.CaskRetrieveBase(), c.expected)
		}
	}
}

func TestSiteDBMethods(t *testing.T) {
	tmpDB := "test.db"
	db, err := bolt.Open(tmpDB, 0600, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Errorf("error closing db: %v", err)
		}
		if err := os.Remove(tmpDB); err != nil {
			t.Errorf("error removing tmp db: %v", err)
		}
	}()

	s := newSite("http://example.com", 1024, db)
	s.EnsureBuckets()

	pr := postResponse{
		Key:       "sha1:1234567890123456789012345678901234567890",
		Extension: ".txt",
		MimeType:  "text/plain",
		Size:      10,
		Chunks:    []string{"sha1:chunk1"},
	}

	s.Add(pr)

	k, _ := keyFromString(pr.Key)
	retrieved, found := s.Get(k)
	if !found {
		t.Errorf("Expected to find entry for %s", pr.Key)
	}
	if retrieved.Key != pr.Key {
		t.Errorf("Retrieved key = %s, want %s", retrieved.Key, pr.Key)
	}

	k2, _ := keyFromString("sha1:0000000000000000000000000000000000000000")
	_, found = s.Get(k2)
	if found {
		t.Errorf("Expected not to find entry for %s", k2.String())
	}
}
