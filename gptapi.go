package main

import (
	"context"
	"errors"
	"os"

	openai "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/responses"
)

var openaiClient *openai.Client

func InitGPT(ctx context.Context) error {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return errors.New("OPENAI_API_KEY is empty")
	}

	c := openai.NewClient(option.WithAPIKey(apiKey))
	openaiClient = &c
	return nil
}

func CallGPT(ctx context.Context, prompt string) (string, error) {
	if openaiClient == nil {
		return "", errors.New("openai client not initialized")
	}

	resp, err := openaiClient.Responses.New(ctx, responses.ResponseNewParams{
		Model: openai.ChatModel("gpt-oss-20b"),
		Input: responses.ResponseNewParamsInputUnion{
			OfString: openai.String(prompt),
		},
	})
	if err != nil {
		return "", err
	}

	return resp.OutputText(), nil
}
