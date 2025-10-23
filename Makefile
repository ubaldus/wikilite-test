TARGET := wikilite
GOARCH := $(shell go env GOARCH)
GOOS := $(shell go env GOOS)
EXT_LDFLAGS :=

LOCAL_EMBEDDINGS_SUPPORTED := false
ifeq ($(GOOS),darwin)
  LOCAL_EMBEDDINGS_SUPPORTED := true
endif
ifeq ($(GOOS),linux)
  EXT_LDFLAGS := -static
  ifeq ($(GOARCH),amd64)
    LOCAL_EMBEDDINGS_SUPPORTED := true
  endif
  ifeq ($(GOARCH),arm64)
    LOCAL_EMBEDDINGS_SUPPORTED := true
  endif
endif
ifeq ($(GOOS),windows)
  TARGET := wikilite.exe
  LOCAL_EMBEDDINGS_SUPPORTED := true
endif

.PHONY: all lint clean static

all: lint wikilite

lint:
	@gofmt -w ./app

clean:
	@rm -rf build
	@rm -f wikilite wikilite.exe


LIBRARY_PATH := build/bin/libembedding_wrapper.a

ifeq ($(LOCAL_EMBEDDINGS_SUPPORTED),true)
wikilite: $(LIBRARY_PATH) $(shell find app -type f)
	@echo "Building wikilite with local embeddings support..."
	go build -v -tags "fts5 aiLocal" -ldflags="-s -w -extldflags '$(EXT_LDFLAGS)'" -o $(TARGET) ./app
else
wikilite: $(shell find app -type f)
	@echo "Building wikilite without local embeddings support..."
	go build -v -tags "fts5" -ldflags="-s -w -extldflags '$(EXT_LDFLAGS)'" -o $(TARGET) ./app
endif

CMAKE_GENERATOR :=
ifeq ($(GOOS),windows)
  CMAKE_GENERATOR := -G "MinGW Makefiles"
endif

$(LIBRARY_PATH): CMakeLists.txt $(shell find src -type f)
	@mkdir -p build
	@cd build && cmake .. $(CMAKE_GENERATOR)
	@cmake --build build -j
