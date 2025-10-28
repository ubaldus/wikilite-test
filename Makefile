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
	@git submodule update --init --force llama.cpp


CMAKE_GENERATOR :=
ifeq ($(GOOS),windows)
  CMAKE_GENERATOR := -G "MinGW Makefiles"
endif

ifeq ($(LOCAL_EMBEDDINGS_SUPPORTED),true)
wikilite: CMakeLists.txt $(shell find app -type f) $(shell find src -type f)
	@if ! grep -q "ggml_set_memory_buffer" llama.cpp/ggml/src/ggml.c; then patch -p1 < src/llama_cpp.patch; fi
	@mkdir -p build
	@cd build && cmake .. $(CMAKE_GENERATOR)
	@cmake --build build -j
	go build -v -tags "fts5 aiInternal" -ldflags="-s -w -extldflags '$(EXT_LDFLAGS)'" -o $(TARGET) ./app
else
wikilite: $(shell find app -type f)
	go build -v -tags "fts5" -ldflags="-s -w -extldflags '$(EXT_LDFLAGS)'" -o $(TARGET) ./app
endif
