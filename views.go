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

func indexHandler(w http.ResponseWriter, r *http.Request, s *site) {
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

func infoHandler(w http.ResponseWriter, r *http.Request, s *site) {
	p := infoPage{
		Title:     "Hakmes status",
		ChunkSize: s.ChunkSize,
		CaskBase:  s.CaskBase,
	}
	t, _ := template.New("status").Parse(statusTemplate)
	t.Execute(w, p)
}

type postResponse struct {
	Key       string   `json:"key"`
	Extension string   `json:"extension"`
	MimeType  string   `json:"mimetype"`
	Size      int64    `json:"size"`
	Chunks    []string `json:"chunks"`
}

func postFileHandler(w http.ResponseWriter, r *http.Request, s *site) {
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
	key, err := keyFromString("sha1:" + fmt.Sprintf("%x", h.Sum(nil)))
	if err != nil {
		log.Println(err)
		http.Error(w, "bad hash", 500)
		return
	}
	f.Seek(0, 0)
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
	numChunks := 0
	var chunkKeys []string
	buf := make([]byte, s.ChunkSize)
	for {
		// upload each chunk to cask
		nr, er := f.Read(buf)
		if nr > 0 {
			numChunks++
			key, err := sendChunkToCask(buf[0:nr], s)
			if err != nil {
				log.Println(err.Error())
				http.Error(w, "failed to write to cask", 500)
				return
			}
			chunkKeys = append(chunkKeys, key.String())
		}
		if er == io.EOF {
			break
		}
	}
	log.Printf("%d chunks\n", numChunks)
	// write db entry
	pr = postResponse{
		Key:       key.String(),
		Size:      size,
		Extension: extension,
		MimeType:  mimetype,
		Chunks:    chunkKeys,
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

func sendChunkToCask(chunk []byte, s *site) (key, error) {
	resp, err := postFile(bytes.NewBuffer(chunk), s.CaskPostURL())
	if err != nil {
		log.Println("Couldn't send chunk to cask")
		log.Println(err.Error())
		return key{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		log.Println("didn't get a 200 from Cask")
		return key{}, errors.New("cask failed")
	}
	b, _ := ioutil.ReadAll(resp.Body)
	var cr caskresponse
	err = json.Unmarshal(b, &cr)
	if err != nil {
		return key{}, err
	}
	if !cr.Success {
		return key{}, errors.New("cask could not store to enough nodes")
	}
	k, err := keyFromString(cr.Key)
	return *k, nil
}

func postFile(f io.Reader, targetURL string) (*http.Response, error) {
	bodyBuf := bytes.NewBufferString("")
	bodyWriter := multipart.NewWriter(bodyBuf)
	fileWriter, err := bodyWriter.CreateFormFile("file", "file.dat")
	if err != nil {
		panic(err.Error())
	}
	io.Copy(fileWriter, f)
	// .Close() finishes setting it up
	// do not defer this or it will make and empty POST request
	bodyWriter.Close()
	contentType := bodyWriter.FormDataContentType()
	c := http.Client{}
	req, err := http.NewRequest("POST", targetURL, bodyBuf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	//	req.Header.Set("X-Cask-Cluster-Secret", secret)

	return c.Do(req)
}

func retrieveHandler(w http.ResponseWriter, r *http.Request, s *site) {
	parts := strings.Split(r.URL.String(), "/")
	if len(parts) == 4 {
		key := parts[2]
		k, err := keyFromString(key)
		if err != nil {
			http.Error(w, "invalid key", 400)
			return
		}

		// If-None-Match is *always* safe to handle since the key
		// is the hash of the content. It just has to be the same
		// as the hash in the path.
		if inm := r.Header.Get("If-None-Match"); inm != "" {
			if inm == "\""+key+"\"" {
				h := w.Header()
				delete(h, "Content-Type")
				delete(h, "Content-Length")
				w.WriteHeader(http.StatusNotModified)
				return
			}
		}
		metadata, found := s.Get(k)
		if found != true {
			http.Error(w, "file not found", 404)
		}

		log.Println(metadata.MimeType)
		w.Header().Set("Content-Type", metadata.MimeType)
		w.Header().Set("ETag", "\""+key+"\"")
		for _, key := range metadata.Chunks {
			data, err := getChunkFromCask(key, s.CaskRetrieveBase())
			if err != nil {
				http.Error(w, "cask retrieve failed", 500)
				return
			}
			w.Write(data)
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			} else {
				log.Println("not a flusher")
			}
		}
	} else {
		http.Error(w, "bad request", 400)
	}
}

func getChunkFromCask(key, caskBase string) ([]byte, error) {
	url := caskBase + key + "/"
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

var statusTemplate = `
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
