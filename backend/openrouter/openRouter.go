package openrouter

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/sashabaranov/go-openai"
)

type RouterAi struct {
	client *openai.Client
	model  string
}

func NewRouterAi(model string) *RouterAi {
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		log.Println("OPENROUTER_API_KEY is not set")
		return nil
	}

	config := openai.DefaultConfig(apiKey)
	config.BaseURL = "https://openrouter.ai/api/v1"

	client := openai.NewClientWithConfig(config)
	return &RouterAi{
		client: client,
		model:  model,
	}
}

func (r *RouterAi) Check(ctx context.Context) error {
	log.Println("Checking OpenRouter API...", r.model)

	resp, err := r.client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: r.model,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    "user",
					Content: "Привет как ты?",
				},
			},
		},
	)
	if err != nil {
		log.Printf("Ошибка проверки OpenRouter API: %v", err)
		return err
	}
	log.Println("OpenRouter API is up and running")
	log.Println(resp)
	return nil
}

func (r *RouterAi) ExtractFacts(ctx context.Context, title, content string) (string, error) {
	prompt := `Проанализируй следующую новость и извлеки ключевые факты в формате JSON.

Структура JSON должна быть строго такой:
{
  "dates": ["дата1", "дата2"],
  "people": ["имя1", "имя2"],
  "organizations": ["организация1", "организация2"],
  "locations": ["место1", "место2"],
  "main_thesis": "основная мысль новости в одном предложении",
  "key_numbers": [{"value": "число", "context": "что это число означает"}],
  "keywords": ["ключевое слово1", "ключевое слово2"],
  "categories": ["категория1", "категория2"]
}

Правила:
- Если какой-то категории нет в тексте, оставь пустой массив []
- main_thesis должен быть кратким (1-2 предложения максимум)
- keywords - это самые важные термины и понятия из новости (5-10 слов)
- Извлекай только факты, не добавляй свои интерпретации
- Возвращай только JSON, без дополнительного текста

Заголовок: ` + title + `
Текст новости: ` + content

	resp, err := r.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: r.model,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: 0.3,
	})
	if err != nil {
		return "", fmt.Errorf("AI request failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from AI")
	}

	return resp.Choices[0].Message.Content, nil
}

func (r *RouterAi) Rewrite(ctx context.Context, prompt string) (string, error) {
	resp, err := r.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: r.model,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: 0.7,
	})

	if err != nil {
		return "", fmt.Errorf("AI request failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from AI")
	}

	return resp.Choices[0].Message.Content, nil
}
