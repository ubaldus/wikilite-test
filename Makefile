.PHONY: clean test lint wikilite release-dry-run release

PACKAGE_NAME := github.com/eja/wikilite
GOLANG_CROSS_VERSION := v1.22.2
GOPATH ?= '$(HOME)/go'

all: lint wikilite

clean:
	@rm -f wikilite wikilite.exe

lint:
	@gofmt -w .

test:
	@go mod tidy
	@go mod verify
	@go vet ./...
	@go test -v ./test

wikilite:
	@go build -tags "fts5" -ldflags "-s -w" -o wikilite .
	@strip wikilite
