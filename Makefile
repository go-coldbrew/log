.PHONY: build test doc lint bench
HEADER="[![GoDoc](https://img.shields.io/badge/pkg.go.dev-doc-blue)](http://pkg.go.dev/github.com/go-coldbrew/log)"
build:
	go build ./...

test:
	go test ./... -race

doc:
	go install github.com/princjef/gomarkdoc/cmd/gomarkdoc@latest
	gomarkdoc ./...

lint:
	golangci-lint run

bench:
	go test -bench=. -benchmem ./...
