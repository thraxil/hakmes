hakmes: *.go
	go build .

test: hakmes
	go test .

coverage: hakmes
	go test -coverprofile=coverage.out ./...

coverage-report: coverage
	go tool cover -func=coverage.out

coverage-html: coverage
	go tool cover -html=coverage.out -o coverage.html

install_deps:
	go get github.com/kelseyhightower/envconfig
	go get github.com/boltdb/bolt/

install: hakmes
	cp hakmes /usr/local/bin/

lint:
	golangci-lint run
