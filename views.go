package main

import (
	"net/http"
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

func postFileHandler(w http.ResponseWriter, r *http.Request, s *Site) {

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
