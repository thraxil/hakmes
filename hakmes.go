package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/boltdb/bolt"
	"github.com/kelseyhightower/envconfig"
)

func makeHandler(fn func(http.ResponseWriter, *http.Request, *Site), s *Site) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s\n", r.Method, r.URL.String())
		fn(w, r, s)
	}
}

type Config struct {
	Port     int
	SSL_Cert string `envconfig:"SSL_CERT"`
	SSL_Key  string `envconfig:"SSL_Key"`

	CaskBase  string `envconfig:"CASK_BASE"`
	ChunkSize int64  `envconfig:"CHUNK_SIZE"`

	DBPath string `envconfig:"DB_PATH"`
}

func main() {
	var c Config
	err := envconfig.Process("hakmes", &c)
	if err != nil {
		log.Fatal(err.Error())
	}

	db, err := bolt.Open(c.DBPath, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	s := NewSite(c.CaskBase, c.ChunkSize, db)
	s.EnsureBuckets()

	log.Println("=== Hakmes starting ===============")
	log.Printf("running on http://localhost:%d\n", c.Port)
	log.Println("using Cask at " + c.CaskBase)
	log.Printf("chunk size: %d bytes\n", c.ChunkSize)
	log.Println("===================================")

	http.HandleFunc("/", makeHandler(indexHandler, s))
	http.HandleFunc("/file/", makeHandler(retrieveHandler, s))
	http.HandleFunc("/favicon.ico", faviconHandler)

	if c.SSL_Cert != "" && c.SSL_Key != "" {
		log.Fatal(http.ListenAndServeTLS(fmt.Sprintf(":%d", c.Port), c.SSL_Cert, c.SSL_Key, nil))
	} else {
		log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", c.Port), nil))
	}
}
