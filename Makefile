.PHONY: clean test lint wikilite release-dry-run release

PACKAGE_NAME := github.com/eja/wikilite
GOLANG_CROSS_VERSION := v1.24
GOPATH ?= '$(HOME)/go'

all: lint wikilite

clean:
	@rm -f wikilite wikilite.exe

lint:
	@gofmt -w ./app

wikilite:
	@go build -tags "fts5" -ldflags "-s -w" -o wikilite ./app
	@strip wikilite
