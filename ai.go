package main

import (
	"context"

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
		Model: openai.F(options.aiModelLLM),
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
		Model:          openai.F(options.aiModelEmbedding),
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
