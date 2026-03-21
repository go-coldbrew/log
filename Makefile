.PHONY: build test doc lint bench
HEADER="[![GoDoc](https://img.shields.io/badge/pkg.go.dev-doc-blue)](http://pkg.go.dev/github.com/go-coldbrew/log)"
build:
	go build ./...

test:
	go test -race ./...

doc:
	go install github.com/princjef/gomarkdoc/cmd/gomarkdoc@latest
	gomarkdoc ./...

lint:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	golangci-lint run

bench:
	go test -run=^$ -bench=. -benchmem ./...
