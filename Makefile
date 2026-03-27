.PHONY: build test doc lint bench
HEADER="[![GoDoc](https://img.shields.io/badge/pkg.go.dev-doc-blue)](http://pkg.go.dev/github.com/go-coldbrew/log)"
build:
	go build ./...

test:
	go test -race ./...

doc:
	go tool gomarkdoc ./...

lint:
	go tool golangci-lint run
	go tool govulncheck ./...

bench:
	go test -run=^$$ -bench=. -benchmem ./...
