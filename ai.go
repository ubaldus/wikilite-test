// Copyright (C) 2024 by Ubaldo Porcheddu <ubaldo@eja.it>

package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"math/bits"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/shared"
)

func aiInstruct(input string) (output string, err error) {
	client := openai.NewClient(
		option.WithAPIKey(options.aiApiKey),
		option.WithBaseURL(options.aiUrl),
	)
	chatCompletion, err := client.Chat.Completions.New(context.TODO(), openai.ChatCompletionNewParams{
		Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(input),
		}),
		Model: openai.F(options.aiLlmModel),
	})
	if err != nil {
		return "", err
	}
	return chatCompletion.Choices[0].Message.Content, nil
}

func aiEmbeddings(input string) (output []float32, err error) {
	client := openai.NewClient(
		option.WithAPIKey(options.aiApiKey),
		option.WithBaseURL(options.aiUrl),
	)
	response, err := client.Embeddings.New(context.TODO(), openai.EmbeddingNewParams{
		Model:          openai.F(options.aiEmbeddingModel),
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
