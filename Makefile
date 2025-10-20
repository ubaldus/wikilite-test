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

# Platform-specific variables
ifeq ($(GOOS),windows)
TARGET := wikilite.exe
# For MinGW builds, libraries are in build/bin/ with .a extension and no 'lib' prefix
LIBRARY_PATH := build/bin/embedding_wrapper.a
SYSTEM_LIBS := -lws2_32 -lbcrypt -ladvapi32 -luser32 -lole32 -loleaut32
# Use -l: syntax for exact filename matching with MinGW
WINDOWS_EXT_LDFLAGS := -L$(CURDIR)/build/bin -l:embedding_wrapper.a -l:common.a -l:llama.a -l:ggml.a -l:ggml-base.a -l:ggml-cpu.a $(SYSTEM_LIBS)
else
LIBRARY_PATH := build/bin/libembedding_wrapper.a
WINDOWS_EXT_LDFLAGS :=
endif

ifeq ($(LOCAL_EMBEDDINGS_SUPPORTED),true)
# For Windows, use the Windows-specific flags, for others use standard flags
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
	@cd build && cmake -G "MinGW Makefiles" ..
	@cmake --build build -j
