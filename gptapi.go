package main

import (
	"context"
	"errors"

	"os"

	"google.golang.org/genai"
)

var geminiClient *genai.Client

func InitGemini(ctx context.Context) error {
	apiKey := os.Getenv("API_KEY")
	if apiKey == "" {
		return errors.New("GEMINI_API_KEY is empty")
	}

	c, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return err
	}

	geminiClient = c
	return nil
}

func CallGPT(ctx context.Context, prompt string) (string, error) {
	if geminiClient == nil {
		return "", errors.New("gemini client not initialized")
	}

	resp, err := geminiClient.Models.GenerateContent(
		ctx,
		"gemini-3-flash-preview",
		genai.Text(prompt),
		nil,
	)
	if err != nil {
		return "", err
	}

	return resp.Text(), nil
}
