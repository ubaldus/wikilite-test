GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)

TARGET := wikilite
ifeq ($(GOOS),windows)
	TARGET := wikilite.exe
endif

LOCAL_EMBEDDINGS_SUPPORTED := false
ifeq ($(GOOS),darwin)
  LOCAL_EMBEDDINGS_SUPPORTED := true
endif
ifeq ($(GOOS),linux)
  ifeq ($(GOARCH),amd64)
    LOCAL_EMBEDDINGS_SUPPORTED := true
  endif
  ifeq ($(GOARCH),arm64)
    LOCAL_EMBEDDINGS_SUPPORTED := true
  endif
endif

.PHONY: all lint clean

all: lint wikilite

lint:
	@gofmt -w ./app

clean:
	@rm -rf build
	@rm -f wikilite wikilite.exe

ifeq ($(LOCAL_EMBEDDINGS_SUPPORTED),true)

wikilite: build/bin/libembedding_wrapper.a $(shell find app -type f)
	@echo "Building wikilite with local embeddings support..."
	go build -v -tags "fts5 aiLocal" -ldflags="-s -w -extldflags '-L$(CURDIR)/build/bin'" -o $(TARGET) ./app

else

wikilite: $(shell find app -type f)
	@echo "Building wikilite without local embeddings support..."
	go build -v -tags "fts5" -ldflags="-s -w" -o $(TARGET) ./app

endif

build/bin/libembedding_wrapper.a: CMakeLists.txt $(shell find src -type f)
	@mkdir -p build
	@cd build && cmake ..
	@cmake --build build --config Release -j
