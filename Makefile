hakmes: *.go
	go build .

test: hakmes
	go test .

install_deps:
	go get github.com/kelseyhightower/envconfig
	go get github.com/boltdb/bolt/

install: hakmes
	cp hakmes /usr/local/bin/
