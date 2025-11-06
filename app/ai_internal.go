// Copyright (C) by Ubaldo Porcheddu <ubaldo@eja.it>

//go:build aiInternal

package main

/*
#cgo CFLAGS: -I../src
#cgo linux LDFLAGS: -L../build/bin -lembedding_wrapper -lcommon -lllama -lggml -lggml-base -lggml-cpu -lstdc++ -lpthread -lm -ldl
#cgo darwin LDFLAGS: -L../build/bin -lembedding_wrapper -lcommon -lllama -lggml -lggml-base -lggml-cpu -lstdc++ -framework Accelerate
#cgo windows LDFLAGS: -L../build/bin -lembedding_wrapper -lcommon -lllama -lggml -lggml-base -lstdc++ -lws2_32 -lbcrypt -ladvapi32
#include "llama_embeddings.h"
#include <stdlib.h>
*/
import "C"

import (
	"fmt"
	"os"
	"runtime"
	"sync"
	"unsafe"
)

var aiInternalInitOnce = sync.OnceValue(aiInternalInit)

func aiInternal() bool {
	return true
}

func aiInternalInit() error {
	modelPath := "memory:"
	modelData := db.AiModelLoad()
	if modelData == nil {
		return fmt.Errorf("No internal AI model available")
	}

	if runtime.GOOS == "windows" {
		tmpFile, err := os.CreateTemp("", "model-*.gguf")
		if err != nil {
			return err
		}
		defer tmpFile.Close()
		defer os.Remove(tmpFile.Name())

		if _, err := tmpFile.Write(modelData); err != nil {
			return err
		}
		modelPath = tmpFile.Name()
		modelData = nil
	}

	cModelPath := C.CString(modelPath)
	cThreadNumber := C.int(options.aiThreads)
	defer C.free(unsafe.Pointer(cModelPath))

	if modelData != nil {
		cBuffer := C.CBytes(modelData)
		defer C.free(cBuffer)
		C.llama_copy_memory_buffer(cBuffer, C.size_t(len(modelData)))
		cModelPath = C.CString(modelPath)
		defer C.free(unsafe.Pointer(cModelPath))
	}

	if result := C.llama_embeddings_init(cModelPath, cThreadNumber); result != 0 {
		return fmt.Errorf("failed to initialize llama embeddings: error code %d", int(result))
	}

	return nil
}

func aiEmbeddings(input string) ([]float32, error) {
	if options.aiApi {
		return aiApiEmbeddings(input)
	} else {
		if err := aiInternalInitOnce(); err != nil {
			return nil, err
		}
		return aiInternalEmbeddings(input)
	}
}

func aiInternalEmbeddings(input string) ([]float32, error) {
	if C.llama_embeddings_get_dimension() < 1 {
		return nil, fmt.Errorf("model not initialized or already freed")
	}

	cText := C.CString(input)
	defer C.free(unsafe.Pointer(cText))

	var nEmb int
	nEmbPtr := (*C.int)(unsafe.Pointer(&nEmb))

	cEmbedding := C.llama_embeddings_get(cText, nEmbPtr)

	if cEmbedding == nil {
		return nil, fmt.Errorf("failed to generate embedding for text: '%s'", input)
	}
	defer C.llama_embeddings_free_output(cEmbedding)

	if nEmb <= 0 {
		return nil, fmt.Errorf("embedding generation returned invalid size: %d", nEmb)
	}

	embeddingDim := int(C.llama_embeddings_get_dimension())
	if embeddingDim <= 0 {
		return nil, fmt.Errorf("failed to get embedding dimension")
	}
	totalFloats := nEmb * embeddingDim

	goSlice := unsafe.Slice((*C.float)(unsafe.Pointer(cEmbedding)), totalFloats)
	result := make([]float32, totalFloats)
	for i, v := range goSlice {
		result[i] = float32(v)
	}

	return result, nil
}
