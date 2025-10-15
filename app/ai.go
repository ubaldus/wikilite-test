// Copyright (C) by Ubaldo Porcheddu <ubaldo@eja.it>

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/ollama/ollama/llama"
	"github.com/ollama/ollama/llm"
)

type embeddingRequest struct {
	Model          string      `json:"model"`
	Input          interface{} `json:"input"`
	EncodingFormat string      `json:"encoding_format,omitempty"`
}

type embeddingResponse struct {
	Object string `json:"object"`
	Data   []struct {
		Object    string    `json:"object"`
		Index     int       `json:"index"`
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

var aiLocal struct {
	model     *llama.Model
	context   *llama.Context
	batchSize int
	isLocal   bool
}

func aiModelIsLocal(value string) bool {
	if _, err := os.Stat(value); err == nil {
		return true
	}
	if _, err := os.Stat(value + ".gguf"); err == nil {
		options.aiModel = value + ".gguf"
		return true
	}
	return false
}

func aiInit() (err error) {
	aiModelPath := options.aiModel
	if _, err := os.Stat(aiModelPath); err == nil {
		aiLocal.isLocal = true
	} else if _, err := os.Stat(aiModelPath + ".gguf"); err == nil {
		aiLocal.isLocal = true
		aiModelPath += ".gguf"
	}
	if aiLocal.isLocal {
		originalStderr := os.Stderr
		if os.Stderr, err = MuteStderr(); err != nil {
			return
		}

		ggml, err := llm.LoadModel(aiModelPath, 0)
		if err != nil {
			return err
		}

		blockCount := int(ggml.KV().BlockCount() + 1)
		aiLocal.model, err = llama.LoadModelFromFile(aiModelPath, llama.ModelParams{NumGpuLayers: blockCount, VocabOnly: false})
		if err != nil {
			return err
		}

		aiLocal.batchSize = 512
		kvSize := int(ggml.KV().ContextLength())
		aiLocal.context, err = llama.NewContextWithModel(aiLocal.model, llama.NewContextParams(kvSize, aiLocal.batchSize, 1, 1, false, ""))
		if err != nil {
			return err
		}
		os.Stderr = originalStderr
	}

	if _, err := aiEmbeddings("test"); err != nil {
		return fmt.Errorf("AI error loading embedding model: %v", err)
	}

	return nil
}

func aiEmbeddings(input string) (output []float32, err error) {
	if aiLocal.isLocal {
		tokens, err := aiLocal.model.Tokenize(input, true, true)
		if err != nil {
			return nil, fmt.Errorf("failed to tokenize prompt: %w", err)
		}

		batchSize := aiLocal.batchSize
		if len(tokens) > batchSize {
			batchSize = len(tokens)
		}

		batch, err := llama.NewBatch(batchSize, 1, 0)
		if err != nil {
			return nil, fmt.Errorf("failed to create new batch: %w", err)
		}
		defer batch.Free()

		aiLocal.context.KvCacheClear()

		for i, token := range tokens {
			isLastToken := i == len(tokens)-1
			batch.Add(token, nil, i, isLastToken, 0)
		}

		if err := aiLocal.context.Decode(batch); err != nil {
			return nil, fmt.Errorf("failed to decode batch: %w", err)
		}

		embeddings := aiLocal.context.GetEmbeddingsSeq(0)
		if embeddings == nil {
			return nil, fmt.Errorf("failed to get embeddings, result was nil")
		}

		return embeddings, nil

	} else {
		url := options.aiApiUrl
		payload := embeddingRequest{
			Model:          options.aiModel,
			Input:          input,
			EncodingFormat: "float",
		}

		body, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal embedding request: %v", err)
		}

		req, err := http.NewRequestWithContext(context.TODO(), "POST", url, bytes.NewBuffer(body))
		if err != nil {
			return nil, fmt.Errorf("failed to create HTTP request: %v", err)
		}

		req.Header.Set("Content-Type", "application/json")
		if options.aiApiKey != "" {
			req.Header.Set("Authorization", "Bearer "+options.aiApiKey)
		}

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("HTTP request failed: %v", err)
		}
		defer resp.Body.Close()

		var apiResp embeddingResponse
		if decodeErr := json.NewDecoder(resp.Body).Decode(&apiResp); decodeErr != nil {
			return nil, fmt.Errorf("failed to decode response (status %d): %v", resp.StatusCode, decodeErr)
		}

		if resp.StatusCode != http.StatusOK {
			if apiResp.Error.Message != "" {
				return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, apiResp.Error.Message)
			}
			return nil, fmt.Errorf("API request failed with status %d", resp.StatusCode)
		}

		if len(apiResp.Data) == 0 {
			return nil, fmt.Errorf("no embeddings returned")
		}

		return apiResp.Data[0].Embedding, nil
	}
}
