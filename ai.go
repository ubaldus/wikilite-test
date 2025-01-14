// Copyright (C) 2024-2025 by Ubaldo Porcheddu <ubaldo@eja.it>

package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"math/bits"
	"os"
	"runtime"

	"github.com/ollama/ollama/llama"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/shared"
)

var aiLocal struct {
	model     *llama.Model
	context   *llama.Context
	batchSize int
}

func aiInit() error {
	if options.aiApiUrl == "" {
		aiModelPath := options.aiModelFile
		if aiModelPath == "" {
			aiModelPath = options.aiModel + ".gguf"
		}
		if _, err := os.Stat(aiModelPath); err != nil {
			return err
		} else {
			originalStderr := os.Stderr
			if os.Stderr, err = MuteStderr(); err != nil {
				return err
			}
			aiLocal.batchSize = 512
			llama.BackendInit()
			aiLocal.model, err = llama.LoadModelFromFile(aiModelPath, llama.ModelParams{UseMmap: true})
			if err != nil {
				return err
			}
			aiLocal.context, err = llama.NewContextWithModel(aiLocal.model, llama.NewContextParams(2048, aiLocal.batchSize, 1, runtime.NumCPU(), false, ""))
			if err != nil {
				return err
			}
			os.Stderr = originalStderr
		}
	}

	if _, err := aiEmbeddings("test"); err != nil {
		return fmt.Errorf("AI error loading embedding model: %v", err)
	}

	return nil
}

func aiEmbeddings(input string) (output []float32, err error) {
	if options.aiApiUrl == "" {
		tokens, err := aiLocal.model.Tokenize(input, true, true)
		if err != nil {
			return nil, fmt.Errorf("failed to tokenize text: %v", err)
		}

		var embeddings []float32
		seqId := 0

		for i := 0; i < len(tokens); i += aiLocal.batchSize {
			end := i + aiLocal.batchSize
			if end > len(tokens) {
				end = len(tokens)
			}

			batchTokens := tokens[i:end]
			batch, err := llama.NewBatch(len(batchTokens), 1, 0)
			if err != nil {
				return nil, fmt.Errorf("failed to create batch: %v", err)
			}

			for j, token := range batchTokens {
				isLast := (i + j + 1) == len(tokens)
				batch.Add(token, nil, j, isLast, seqId)
			}

			if err := aiLocal.context.Decode(batch); err != nil {
				batch.Free()
				return nil, fmt.Errorf("failed to decode batch: %v", err)
			}

			if i+len(batchTokens) == len(tokens) {
				batchEmbeddings := aiLocal.context.GetEmbeddingsSeq(seqId)
				if batchEmbeddings != nil {
					embeddings = batchEmbeddings
				}
			}

			batch.Free()
		}

		if embeddings == nil || len(embeddings) == 0 {
			return nil, fmt.Errorf("failed to get embeddings")
		}

		return embeddings, nil

	} else {

		client := openai.NewClient(
			option.WithAPIKey(options.aiApiKey),
			option.WithBaseURL(options.aiApiUrl),
		)
		response, err := client.Embeddings.New(context.TODO(), openai.EmbeddingNewParams{
			Model:          openai.F(options.aiModel),
			Input:          openai.F[openai.EmbeddingNewParamsInputUnion](shared.UnionString(input)),
			EncodingFormat: openai.F(openai.EmbeddingNewParamsEncodingFormatFloat),
		})

		if err != nil {
			return nil, err
		}

		for _, embedding := range response.Data {
			for _, value := range embedding.Embedding {
				output = append(output, float32(value))
			}
		}
	}
	return
}

func aiL2Distance(a, b []float32) (float32, error) {
	if len(a) != len(b) {
		return 0, fmt.Errorf("vectors must have the same length")
	}

	var sum float32
	for i := 0; i < len(a); i++ {
		diff := a[i] - b[i]
		sum += diff * diff
	}

	return float32(math.Sqrt(float64(sum))), nil
}

func aiHammingDistance(a, b []byte) (float32, error) {
	if len(a) != len(b) {
		return 0, fmt.Errorf("bit arrays must have the same length")
	}

	diffCount := 0
	for i := 0; i < len(a); i++ {
		diffBits := a[i] ^ b[i]
		diffCount += bits.OnesCount8(diffBits)
	}

	return float32(diffCount), nil
}

func aiQuantizeBinary(values []float32) []byte {
	numBytes := (len(values) + 7) / 8
	packedData := make([]byte, numBytes)

	for i, value := range values {
		if value >= 0 {
			packedData[i/8] |= 1 << (i % 8)
		}
	}

	return packedData
}

func aiBytesToFloat32(bytes []byte) []float32 {
	if len(bytes)%4 != 0 {
		panic("input byte slice length must be a multiple of 4")
	}

	float32s := make([]float32, 0, len(bytes)/4)
	for i := 0; i < len(bytes); i += 4 {
		bits := binary.LittleEndian.Uint32(bytes[i : i+4])
		float32s = append(float32s, math.Float32frombits(bits))
	}

	return float32s
}

func aiFloat32ToBytes(values []float32) []byte {
	bytes := make([]byte, 4*len(values))

	for i, value := range values {
		bits := math.Float32bits(value)
		binary.LittleEndian.PutUint32(bytes[4*i:4*(i+1)], bits)
	}

	return bytes
}
