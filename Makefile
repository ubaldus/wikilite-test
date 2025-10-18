GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)
EXT_LDFLAGS := 

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

.PHONY: all lint clean static

all: lint wikilite

static: EXT_LDFLAGS := -static $(EXT_LDFLAGS)
static: lint wikilite

lint:
	@gofmt -w ./app

clean:
	@rm -rf build
	@rm -f wikilite wikilite.exe


ifeq ($(LOCAL_EMBEDDINGS_SUPPORTED),true)
EXT_LDFLAGS := -L$(CURDIR)/build/bin $(EXT_LDFLAGS)

wikilite: build/bin/libembedding_wrapper.a $(shell find app -type f)
	@echo "Building wikilite with local embeddings support..."
	go build -v -tags "fts5 aiLocal" -ldflags="-s -w -extldflags '$(EXT_LDFLAGS)'" -o $(TARGET) ./app

else

wikilite: $(shell find app -type f)
	@echo "Building wikilite without local embeddings support..."
	go build -v -tags "fts5" -ldflags="-s -w -extldflags '$(EXT_LDFLAGS)'" -o $(TARGET) ./app

endif

build/bin/libembedding_wrapper.a: CMakeLists.txt $(shell find src -type f)
	@mkdir -p build
	@cd build && cmake ..
	@cmake --build build --config Release -j

