package main

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/boltdb/bolt"
)

func TestInfoHandler(t *testing.T) {
	s := &site{
		CaskBase:  "http://cask.example.com",
		ChunkSize: 1024,
	}
	mux := getMux(s)
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Check the response body
	expected := "Hakmes status"
	if !contains(rr.Body.String(), expected) {
		t.Errorf("handler returned unexpected body: got %v want it to contain %v",
			rr.Body.String(), expected)
	}
}

func contains(s, substr string) bool {
	return (len(s) >= len(substr)) && (s[0:len(substr)] == substr || (len(s) > 0 && contains(s[1:], substr)))
}

func TestPostFileHandler(t *testing.T) {
	// Mock Cask server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cr := caskresponse{
			Key:     "sha1:1234567890123456789012345678901234567890",
			Success: true,
		}
		b, _ := json.Marshal(cr)
		if _, err := w.Write(b); err != nil {
			t.Errorf("error writing to response: %v", err)
		}
	}))
	defer ts.Close()

	// Temp DB
	tmpDB := "test_post.db"
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

	s := newSite(ts.URL, 1024, db)
	s.EnsureBuckets()
	mux := getMux(s)

	// Create multi-part form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "test.txt")
	if _, err := io.WriteString(part, "hello world"); err != nil {
		t.Errorf("error writing to part: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Errorf("error closing writer: %v", err)
	}

	req, err := http.NewRequest("POST", "/", body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	var pr postResponse
	err = json.Unmarshal(rr.Body.Bytes(), &pr)
	if err != nil {
		t.Fatal(err)
	}

	if pr.Size != 11 {
		t.Errorf("Expected size 11, got %d", pr.Size)
	}
}

func TestRetrieveHandler(t *testing.T) {
	// Mock Cask server for GET
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write([]byte("hello world")); err != nil {
			t.Errorf("error writing to response: %v", err)
		}
	}))
	defer ts.Close()

	// Temp DB
	tmpDB := "test_retrieve.db"
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

	s := newSite(ts.URL, 1024, db)
	s.EnsureBuckets()
	mux := getMux(s)

	pr := postResponse{
		Key:       "sha1:1234567890123456789012345678901234567890",
		Extension: ".txt",
		MimeType:  "text/plain",
		Size:      11,
		Chunks:    []string{"sha1:chunk1"},
	}
	s.Add(pr)

	req, err := http.NewRequest("GET", "/file/sha1:1234567890123456789012345678901234567890/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v. body: %s",
			status, http.StatusOK, rr.Body.String())
	}

	if rr.Body.String() != "hello world" {
		t.Errorf("Expected 'hello world', got %q", rr.Body.String())
	}

	// Test Not Found
	req, _ = http.NewRequest("GET", "/file/sha1:0000000000000000000000000000000000000000/", nil)
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("Expected 404 for missing file, got %v", status)
	}

	// Test If-None-Match
	req, _ = http.NewRequest("GET", "/file/sha1:1234567890123456789012345678901234567890/", nil)
	req.Header.Set("If-None-Match", "\"sha1:1234567890123456789012345678901234567890\"")
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusNotModified {
		t.Errorf("Expected 304 for If-None-Match, got %v", status)
	}
}

func TestFaviconHandler(t *testing.T) {
	s := &site{}
	mux := getMux(s)
	req, err := http.NewRequest("GET", "/favicon.ico", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	// it just ignores it, so 200 is expected default if nothing written
	if rr.Code != http.StatusOK {
		t.Errorf("Expected 200, got %v", rr.Code)
	}
}

func TestPostFileHandlerExisting(t *testing.T) {
	// Temp DB
	tmpDB := "test_post_existing.db"
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
	mux := getMux(s)

	// Pre-add an entry
	// SHA1 of "hello world" is 2aae6c35c94fcfb415dbe95f408b9ce91ee846ed
	key := "sha1:2aae6c35c94fcfb415dbe95f408b9ce91ee846ed"
	pr := postResponse{
		Key:       key,
		Size:      11,
		Extension: ".txt",
		MimeType:  "text/plain",
		Chunks:    []string{"sha1:chunk1"},
	}
	s.Add(pr)

	// Create multi-part form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "test.txt")
	if _, err := io.WriteString(part, "hello world"); err != nil {
		t.Errorf("error writing to part: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Errorf("error closing writer: %v", err)
	}

	req, _ := http.NewRequest("POST", "/", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	var prRetrieved postResponse
	err = json.Unmarshal(rr.Body.Bytes(), &prRetrieved)
	if err != nil {
		t.Fatal(err)
	}
	if prRetrieved.Key != key {
		t.Errorf("Expected key %s, got %s", key, prRetrieved.Key)
	}
}

func TestFileInfoHandler(t *testing.T) {
	// Temp DB
	tmpDB := "test_file_info.db"
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
	mux := getMux(s)

	pr := postResponse{
		Key:       "sha1:1234567890123456789012345678901234567890",
		Extension: ".txt",
		MimeType:  "text/plain",
		Size:      11,
		Chunks:    []string{"sha1:chunk1"},
	}
	s.Add(pr)

	req, err := http.NewRequest("GET", "/info/sha1:1234567890123456789012345678901234567890/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	var retrieved postResponse
	err = json.Unmarshal(rr.Body.Bytes(), &retrieved)
	if err != nil {
		t.Fatal(err)
	}
	if retrieved.Key != pr.Key {
		t.Errorf("Expected key %s, got %s", pr.Key, retrieved.Key)
	}

	// Test Not Found
	req, _ = http.NewRequest("GET", "/info/sha1:0000000000000000000000000000000000000000/", nil)
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("Expected 404 for missing file info, got %v", status)
	}

	// Test Invalid Key
	req, _ = http.NewRequest("GET", "/info/invalid/", nil)
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("Expected 400 for invalid key, got %v", status)
	}

	// Test Bad Request
	req, _ = http.NewRequest("GET", "/bad/", nil)
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("Expected 404 for bad request (no route), got %v", status)
	}

	// Test If-None-Match
	req, _ = http.NewRequest("GET", "/info/sha1:1234567890123456789012345678901234567890/", nil)
	req.Header.Set("If-None-Match", "\"sha1:1234567890123456789012345678901234567890\"")
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusNotModified {
		t.Errorf("Expected 304 for If-None-Match, got %v", status)
	}
}

func TestPostFileHandlerNoFile(t *testing.T) {
	s := &site{}
	mux := getMux(s)
	req, _ := http.NewRequest("POST", "/", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected 500 for missing file, got %v", rr.Code)
	}
}

func TestPostFileHandlerCaskFail(t *testing.T) {
	// Mock Cask server returning 500
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "cask error", http.StatusInternalServerError)
	}))
	defer ts.Close()

	// Temp DB
	tmpDB := "test_post_cask_fail.db"
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

	s := newSite(ts.URL, 1024, db)
	s.EnsureBuckets()
	mux := getMux(s)

	// Create multi-part form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "test.txt")
	if _, err := io.WriteString(part, "hello world"); err != nil {
		t.Errorf("error writing string: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Errorf("error closing writer: %v", err)
	}

	req, _ := http.NewRequest("POST", "/", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusInternalServerError {
		t.Errorf("Expected 500 for cask fail, got %v", status)
	}
}

func TestPostFileHandlerCaskSuccessFalse(t *testing.T) {
	// Mock Cask server returning success: false
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cr := caskresponse{
			Key:     "sha1:1234567890123456789012345678901234567890",
			Success: false,
		}
		b, _ := json.Marshal(cr)
		if _, err := w.Write(b); err != nil {
			t.Errorf("error writing response: %v", err)
		}
	}))
	defer ts.Close()

	// Temp DB
	tmpDB := "test_post_cask_success_false.db"
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

	s := newSite(ts.URL, 1024, db)
	s.EnsureBuckets()
	mux := getMux(s)

	// Create multi-part form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "test.txt")
	if _, err := io.WriteString(part, "hello world"); err != nil {
		t.Errorf("error writing string: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Errorf("error closing writer: %v", err)
	}

	req, _ := http.NewRequest("POST", "/", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusInternalServerError {
		t.Errorf("Expected 500 for cask success: false, got %v", status)
	}
}

func TestPostFileHandlerCaskInvalidJSON(t *testing.T) {
	// Mock Cask server returning invalid JSON
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write([]byte("not json")); err != nil {
			t.Errorf("error writing response: %v", err)
		}
	}))
	defer ts.Close()

	// Temp DB
	tmpDB := "test_post_cask_invalid_json.db"
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

	s := newSite(ts.URL, 1024, db)
	s.EnsureBuckets()
	mux := getMux(s)

	// Create multi-part form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "test.txt")
	if _, err := io.WriteString(part, "hello world"); err != nil {
		t.Errorf("error writing string: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Errorf("error closing writer: %v", err)
	}

	req, _ := http.NewRequest("POST", "/", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusInternalServerError {
		t.Errorf("Expected 500 for cask invalid JSON, got %v", status)
	}
}

func TestRetrieveHandlerRange(t *testing.T) {
	// Mock Cask server for GET
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "chunk1") {
			_, _ = w.Write([]byte("01234"))
		} else if strings.Contains(r.URL.Path, "chunk2") {
			_, _ = w.Write([]byte("56789"))
		}
	}))
	defer ts.Close()

	// Temp DB
	tmpDB := "test_retrieve_range.db"
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

	s := newSite(ts.URL, 5, db)
	s.EnsureBuckets()
	mux := getMux(s)

	key := "sha1:1234567890123456789012345678901234567890"
	pr := postResponse{
		Key:       key,
		Extension: ".txt",
		MimeType:  "text/plain",
		Size:      10,
		Chunks:    []string{"sha1:chunk1", "sha1:chunk2"},
	}
	s.Add(pr)

	// Test Range: bytes=2-6 (should get "23456")
	req, _ := http.NewRequest("GET", "/file/"+key+"/", nil)
	req.Header.Set("Range", "bytes=2-6")

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusPartialContent {
		t.Errorf("Expected 206 Partial Content, got %v", status)
	}

	if rr.Body.String() != "23456" {
		t.Errorf("Expected '23456', got %q", rr.Body.String())
	}

	contentRange := rr.Header().Get("Content-Range")
	expectedContentRange := "bytes 2-6/10"
	if contentRange != expectedContentRange {
		t.Errorf("Expected Content-Range %q, got %q", expectedContentRange, contentRange)
	}
}

func TestRetrieveHandlerCaskFail(t *testing.T) {
	// Mock Cask server for GET failing
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer ts.Close()

	// Temp DB
	tmpDB := "test_retrieve_fail.db"
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

	s := newSite(ts.URL, 1024, db)
	s.EnsureBuckets()
	mux := getMux(s)

	pr := postResponse{
		Key:       "sha1:1234567890123456789012345678901234567890",
		Extension: ".txt",
		MimeType:  "text/plain",
		Size:      11,
		Chunks:    []string{"sha1:chunk1"},
	}
	s.Add(pr)

	req, _ := http.NewRequest("GET", "/file/sha1:1234567890123456789012345678901234567890/", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusInternalServerError {
		t.Errorf("Expected 500 for cask fail, got %v", status)
	}
}

func TestRetrieveHandlerInvalidKey(t *testing.T) {
	s := &site{}
	mux := getMux(s)
	req, _ := http.NewRequest("GET", "/file/invalid/", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("Expected 400 for invalid key, got %v", status)
	}
}

func TestRetrieveHandlerBadRequest(t *testing.T) {
	s := &site{}
	mux := getMux(s)
	req, _ := http.NewRequest("GET", "/invalid/", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("Expected 404 for bad request (no route), got %v", status)
	}
}
