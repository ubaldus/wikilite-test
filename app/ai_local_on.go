// Copyright (C) by Ubaldo Porcheddu <ubaldo@eja.it>

//go:build aiLocal

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
	"unsafe"
)

func localAiEnabled() bool {
	return true
}

func localAiInit(modelPath string) error {
	cModelPath := C.CString(modelPath)
	cThreadNumber := C.int(options.aiThreads)
	defer C.free(unsafe.Pointer(cModelPath))

	if result := C.llama_embeddings_init(cModelPath, cThreadNumber); result != 0 {
		return fmt.Errorf("failed to initialize llama embeddings: error code %d", int(result))
	}

	return nil
}

func localAiEmbeddings(input string) ([]float32, error) {
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

	goSlice := (*[1 << 30]C.float)(unsafe.Pointer(cEmbedding))[:totalFloats:totalFloats]
	result := make([]float32, totalFloats)
	for i, v := range goSlice {
		result[i] = float32(v)
	}

	return result, nil
}
