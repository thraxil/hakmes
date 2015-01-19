package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
	"text/template"
)

func indexHandler(w http.ResponseWriter, r *http.Request, s *Site) {
	if r.Method == "GET" {
		infoHandler(w, r, s)
		return
	}
	if r.Method == "POST" {
		postFileHandler(w, r, s)
		return
	}
	http.Error(w, "method not supported", 405)
}

type infoPage struct {
	Title     string
	ChunkSize int64
	CaskBase  string
}

func infoHandler(w http.ResponseWriter, r *http.Request, s *Site) {
	p := infoPage{
		Title:     "Hakmes status",
		ChunkSize: s.ChunkSize,
		CaskBase:  s.CaskBase,
	}
	t, _ := template.New("status").Parse(status_template)
	t.Execute(w, p)
}

type postResponse struct {
	Key       string   `json:"key"`
	Extension string   `json:"extension"`
	MimeType  string   `json:"mimetype"`
	Size      int64    `json:"size"`
	Chunks    []string `json:"chunks"`
}

func postFileHandler(w http.ResponseWriter, r *http.Request, s *Site) {
	// get file from request
	f, fh, err := r.FormFile("file")
	if err != nil {
		log.Println("failure reading file from request")
		log.Println(err.Error())
		http.Error(w, "couldn't read file", 500)
		return
	}
	defer f.Close()
	log.Println("read in file")
	// TODO: force lowercase
	extension := filepath.Ext(fh.Filename)

	mimetype := fh.Header.Get("Content-Type")
	if mimetype == "" {
		mimetype = "application/octet-stream"
	}

	// calculate SHA1
	h := sha1.New()
	size, err := io.Copy(h, f)
	if err != nil {
		log.Println("couldn't copy buffer")
		log.Println(err.Error())
		http.Error(w, "couldn't calculate hash", 500)
		return
	}
	key, err := KeyFromString("sha1:" + fmt.Sprintf("%x", h.Sum(nil)))
	if err != nil {
		log.Println(err)
		http.Error(w, "bad hash", 500)
		return
	}
	f.Seek(0, 0)
	log.Println("calculated hash")
	// if we already have an entry for that hash, we're done
	pr, found := s.Get(key)
	if found {
		log.Println("already had an entry for this key in the database")
		b, err := json.Marshal(pr)
		if err != nil {
			http.Error(w, "json error", 500)
			return
		}
		w.Write(b)
		return
	}
	// split into chunks
	num_chunks := 0
	chunk_keys := make([]string, 0)
	buf := make([]byte, s.ChunkSize)
	for {
		// upload each chunk to cask
		nr, er := f.Read(buf)
		if nr > 0 {
			num_chunks++
			key, err := sendChunkToCask(buf[0:nr], s)
			if err != nil {
				log.Println(err.Error())
				http.Error(w, "failed to write to cask", 500)
				return
			}
			chunk_keys = append(chunk_keys, key.String())
		}
		if er == io.EOF {
			break
		}
	}
	log.Printf("%d chunks\n", num_chunks)
	log.Printf("%v\n", chunk_keys)
	// write db entry
	pr = postResponse{
		Key:       key.String(),
		Size:      size,
		Extension: extension,
		MimeType:  mimetype,
		Chunks:    chunk_keys,
	}
	s.Add(pr)
	// return hash and info

	b, err := json.Marshal(pr)
	if err != nil {
		http.Error(w, "json error", 500)
		return
	}
	w.Write(b)
}

type caskresponse struct {
	Key     string `json:"key"`
	Success bool   `json:"success"`
}

func sendChunkToCask(chunk []byte, s *Site) (Key, error) {
	log.Printf("sendChunkToCask()\n")
	resp, err := postFile(bytes.NewBuffer(chunk), s.CaskPostURL())
	if err != nil {
		log.Println("Couldn't send chunk to cask")
		log.Println(err.Error())
		return Key{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		log.Println("didn't get a 200 from Cask")
		return Key{}, errors.New("cask failed")
	}
	b, _ := ioutil.ReadAll(resp.Body)
	var cr caskresponse
	err = json.Unmarshal(b, &cr)
	if err != nil {
		return Key{}, err
	}
	if !cr.Success {
		return Key{}, errors.New("cask could not store to enough nodes")
	}
	k, err := KeyFromString(cr.Key)
	return *k, nil
}

func postFile(f io.Reader, target_url string) (*http.Response, error) {
	body_buf := bytes.NewBufferString("")
	body_writer := multipart.NewWriter(body_buf)
	file_writer, err := body_writer.CreateFormFile("file", "file.dat")
	if err != nil {
		panic(err.Error())
	}
	io.Copy(file_writer, f)
	// .Close() finishes setting it up
	// do not defer this or it will make and empty POST request
	body_writer.Close()
	content_type := body_writer.FormDataContentType()
	c := http.Client{}
	req, err := http.NewRequest("POST", target_url, body_buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", content_type)
	//	req.Header.Set("X-Cask-Cluster-Secret", secret)

	return c.Do(req)
}

func retrieveHandler(w http.ResponseWriter, r *http.Request, s *Site) {
	parts := strings.Split(r.URL.String(), "/")
	if len(parts) == 4 {
		key := parts[2]
		k, err := KeyFromString(key)
		if err != nil {
			http.Error(w, "invalid key", 400)
			return
		}
		metadata, found := s.Get(k)
		if found != true {
			http.Error(w, "file not found", 404)
		}

		log.Println(metadata.MimeType)
		w.Header().Set("Content-Type", metadata.MimeType)
		for _, key := range metadata.Chunks {
			data, err := getChunkFromCask(key, s.CaskRetrieveBase())
			if err != nil {
				http.Error(w, "cask retrieve failed", 500)
				return
			}
			w.Write(data)
			if f, ok := w.(http.Flusher); ok {
				log.Println("*flush*")
				f.Flush()
			} else {
				log.Println("not a flusher")
			}
		}
	} else {
		http.Error(w, "bad request", 400)
	}
}

func getChunkFromCask(key, cask_base string) ([]byte, error) {
	log.Printf("getChunkFromCask(%s)\n", key)
	url := cask_base + key + "/"
	c := http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	//	req.Header.Set("X-Cask-Cluster-Secret", secret)
	resp, err := c.Do(req)
	defer resp.Body.Close()
	if err != nil {
		return nil, err
	}
	if resp.Status != "200 OK" {
		return nil, errors.New("404, probably")
	}
	b, _ := ioutil.ReadAll(resp.Body)
	return b, nil
}

func faviconHandler(w http.ResponseWriter, r *http.Request) {
	// just ignore this crap
}

var status_template = `
<html>
<head>
<title>{{.Title}}</title>
<link rel="stylesheet" href="//maxcdn.bootstrapcdn.com/bootstrap/3.3.1/css/bootstrap.min.css" />
</head>
<body>
<div class="container">
<h1>Hakmes</h1>
<table class="table">
<tr><th>Cask Base</th><td>{{.CaskBase}}</td></tr>
<tr><th>Chunk Size</th><td>{{.ChunkSize}}</td></tr>
</table>
</div>
</html>
`
