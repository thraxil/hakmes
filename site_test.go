package main

import "testing"

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
