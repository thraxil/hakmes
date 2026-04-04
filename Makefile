.PHONY: all test clean coverage coverage-report coverage-html install install_deps lint

all: hakmes

hakmes: *.go
	CGO_ENABLED=0 go build -ldflags "-s -w" .

test: hakmes
	go test .

clean:
	rm -f hakmes coverage.out coverage.html

coverage: hakmes
	go test -coverprofile=coverage.out ./...

coverage-report: coverage
	go tool cover -func=coverage.out

coverage-html: coverage
	go tool cover -html=coverage.out -o coverage.html

install_deps:
	go mod download

install: hakmes
	cp hakmes /usr/local/bin/

lint:
	golangci-lint run
