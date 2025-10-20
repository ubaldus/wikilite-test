GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)
EXT_LDFLAGS := 

TARGET := wikilite

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
ifeq ($(GOOS),windows)
  ifeq ($(GOARCH),amd64)
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


ifeq ($(GOOS),windows)
TARGET := wikilite.exe
LIBRARY_NAME := embedding_wrapper.lib
LIBRARY_PATH := build/bin/Release/$(LIBRARY_NAME)
SYSTEM_LIBS := -lws2_32 -lbcrypt -ladvapi32 -luser32 -lole32 -loleaut32
WINDOWS_EXT_LDFLAGS := -L$(CURDIR)/build/bin/Release -lembedding_wrapper -lcommon -lllama -lggml $(SYSTEM_LIBS)
else
LIBRARY_NAME := libembedding_wrapper.a
LIBRARY_PATH := build/bin/$(LIBRARY_NAME)
WINDOWS_EXT_LDFLAGS :=
endif

ifeq ($(LOCAL_EMBEDDINGS_SUPPORTED),true)

ifeq ($(GOOS),windows)
EXT_LDFLAGS := $(WINDOWS_EXT_LDFLAGS) $(EXT_LDFLAGS)
else
EXT_LDFLAGS := -L$(CURDIR)/build/bin -lembedding_wrapper -lcommon -lllama -lggml $(EXT_LDFLAGS)
endif

wikilite: $(LIBRARY_PATH) $(shell find app -type f)
	@echo "Building wikilite with local embeddings support..."
	go build -v -tags "fts5 aiLocal" -ldflags="-s -w -extldflags '$(EXT_LDFLAGS)'" -o $(TARGET) ./app

else

wikilite: $(shell find app -type f)
	@echo "Building wikilite without local embeddings support..."
	go build -v -tags "fts5" -ldflags="-s -w -extldflags '$(EXT_LDFLAGS)'" -o $(TARGET) ./app

endif

$(LIBRARY_PATH): CMakeLists.txt $(shell find src -type f)
	@mkdir -p build
	@cd build && cmake ..
	@cmake --build build --config Release -j
