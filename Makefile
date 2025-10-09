.PHONY: clean test lint wikilite release-dry-run release

PACKAGE_NAME := github.com/eja/wikilite
GOLANG_CROSS_VERSION := v1.24
GOPATH ?= '$(HOME)/go'
LOCAL_INCLUDE_PATH := $(CURDIR)/include

all: lint wikilite

clean:
	@rm -f wikilite wikilite.exe

lint:
	@gofmt -w ./app

wikilite:
	@CGO_CXXFLAGS_ALLOW=".*" CGO_CXXFLAGS="-I$(LOCAL_INCLUDE_PATH)" \
	 CGO_CFLAGS_ALLOW=".*" CGO_CFLAGS="-I$(LOCAL_INCLUDE_PATH)" \
	 go build -tags "fts5" -ldflags "-s -w" -o wikilite ./app
