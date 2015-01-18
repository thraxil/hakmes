package main

import "net/http"

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

func infoHandler(w http.ResponseWriter, r *http.Request, s *Site) {

}

func postFileHandler(w http.ResponseWriter, r *http.Request, s *Site) {

}

func faviconHandler(w http.ResponseWriter, r *http.Request) {
	// just ignore this crap
}
