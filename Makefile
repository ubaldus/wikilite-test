.PHONY: clean test lint wikilite release-dry-run release

PACKAGE_NAME := github.com/eja/wikilite
GOLANG_CROSS_VERSION := v1.20
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
	@go build -ldflags "-s -w" -o wikilite *.go
	@strip wikilite

release-dry-run:
	docker run \
		--rm \
		--privileged \
		-e CGO_ENABLED=1 \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v `pwd`:/go/src/$(PACKAGE_NAME) \
		-v ${GOPATH}/pkg:/go/pkg \
		-w /go/src/$(PACKAGE_NAME) \
		ghcr.io/goreleaser/goreleaser-cross:${GOLANG_CROSS_VERSION} \
		--clean --skip-validate --skip-publish --snapshot

release:
	@if [ ! -f ".release-env" ]; then \
		echo "\033[91m.release-env is required for release\033[0m";\
		exit 1;\
	fi
	docker run \
		--rm \
		--privileged \
		-e CGO_ENABLED=1 \
		--env-file .release-env \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v `pwd`:/go/src/$(PACKAGE_NAME) \
		-w /go/src/$(PACKAGE_NAME) \
		ghcr.io/goreleaser/goreleaser-cross:${GOLANG_CROSS_VERSION} \
		release --clean --skip-validate
