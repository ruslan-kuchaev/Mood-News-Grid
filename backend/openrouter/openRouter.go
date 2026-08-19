package openrouter

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/sashabaranov/go-openai"
)

func RouterAI() {
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENROUTER_API_KEY is not set")
	}

	config := openai.DefaultConfig(apiKey)
	config.BaseURL = "https://openrouter.ai/api/v1"

	client := openai.NewClientWithConfig(config)

	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: "google/gemma-4-26b-a4b-it:free",
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: "Какого цвета небо? Ответь кратко на русском языке.",
				},
			},
		},
	)
	if err != nil {
		log.Fatalf("Ошибка запроса: %v", err)
	}

	if len(resp.Choices) > 0 {
		fmt.Println("Ответ нейросети:")
		fmt.Println(resp.Choices[0].Message.Content)
	}
}
