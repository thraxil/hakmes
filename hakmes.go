package main

import (
	_ "expvar"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/boltdb/bolt"
	"github.com/kelseyhightower/envconfig"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func makeHandler(fn func(http.ResponseWriter, *http.Request, *site), s *site) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s\n", r.Method, r.URL.String())
		fn(w, r, s)
	}
}

type config struct {
	Port    int
	SSLCert string `envconfig:"SSL_CERT"`
	SSLKey  string `envconfig:"SSL_Key"`

	CaskBase  string `envconfig:"CASK_BASE"`
	ChunkSize int64  `envconfig:"CHUNK_SIZE"`

	DBPath string `envconfig:"DB_PATH"`
}

func main() {
	var c config
	err := envconfig.Process("hakmes", &c)
	if err != nil {
		log.Fatal(err.Error())
	}

	db, err := bolt.Open(c.DBPath, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	s := newSite(c.CaskBase, c.ChunkSize, db)
	s.EnsureBuckets()

	log.Println("=== Hakmes starting ===============")
	log.Printf("running on http://localhost:%d\n", c.Port)
	log.Println("using Cask at " + c.CaskBase)
	log.Printf("chunk size: %d bytes\n", c.ChunkSize)
	log.Println("===================================")

	http.HandleFunc("/", prometheus.InstrumentHandler("root", makeHandler(indexHandler, s)))
	http.HandleFunc("/file/", prometheus.InstrumentHandler("file", makeHandler(retrieveHandler, s)))
	http.HandleFunc("/info/", prometheus.InstrumentHandler("info", makeHandler(fileInfoHandler, s)))
	http.HandleFunc("/favicon.ico", faviconHandler)
	http.Handle("/metrics", promhttp.Handler())

	if c.SSLCert != "" && c.SSLKey != "" {
		log.Fatal(http.ListenAndServeTLS(fmt.Sprintf(":%d", c.Port), c.SSLCert, c.SSLKey, nil))
	} else {
		log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", c.Port), nil))
	}
}
