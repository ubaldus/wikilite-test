// Copyright (C) 2024-2025 by Ubaldo Porcheddu <ubaldo@eja.it>

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"github.com/ollama/ollama/llama"
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
}

func aiInit() error {
	if options.aiApiUrl == "" {
		aiModelPath := filepath.Join(options.aiModelPath, options.aiModel) + ".gguf"
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
			aiLocal.context, err = llama.NewContextWithModel(aiLocal.model, llama.NewContextParams(options.aiModelContextSize, aiLocal.batchSize, 1, runtime.NumCPU(), false, ""))
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
