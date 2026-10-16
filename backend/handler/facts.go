package handler

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"backend/db"
	"backend/openrouter"
)

// Структура для сохранения готовых фактов на диск
type NewsFact struct {
	URL           string     `json:"url"`
	Title         string     `json:"title"`
	Dates         []string   `json:"dates"`
	People        []string   `json:"people"`
	Organizations []string   `json:"organizations"`
	Locations     []string   `json:"locations"`
	MainThesis    string     `json:"main_thesis"`
	KeyNumbers    []KeyValue `json:"key_numbers"`
	Keywords      []string   `json:"keywords"`
	Categories    []string   `json:"categories"`
	ProcessedAt   time.Time  `json:"processed_at"`
}

type KeyValue struct {
	Value   string `json:"value"`
	Context string `json:"context"`
}

// Структура для парсинга сырого ответа от LLM
type FactsResponse struct {
	Dates         []string   `json:"dates"`
	People        []string   `json:"people"`
	Organizations []string   `json:"organizations"`
	Locations     []string   `json:"locations"`
	MainThesis    string     `json:"main_thesis"`
	KeyNumbers    []KeyValue `json:"key_numbers"`
	Keywords      []string   `json:"keywords"`
	Categories    []string   `json:"categories"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

var (
	sseClients = make(map[chan string]bool)
	sseMutex   sync.Mutex
)

const maxRetries = 3

// hashURL генерирует MD5-хэш ссылки для безопасного имени файла
func hashURL(url string) string {
	hash := md5.Sum([]byte(url))
	return hex.EncodeToString(hash[:])
}

// notifySSEClients рассылает клиентам событие о готовности фактов
func notifySSEClients(urlHash string) {
	sseMutex.Lock()
	defer sseMutex.Unlock()

	log.Printf("[SSE] Отправка события fact_ready для %s (подключено клиентов: %d)", urlHash, len(sseClients))

	message := fmt.Sprintf("data: {\"event\":\"fact_ready\",\"url_hash\":\"%s\"}\n\n", urlHash)
	for client := range sseClients {
		select {
		case client <- message:
			log.Printf("[SSE] Событие отправлено клиенту")
		default:
			log.Printf("[SSE] Канал клиента заблокирован, закрываем")
			close(client)
			delete(sseClients, client)
		}
	}
}

func notifySSEError(title, errorMsg string) {
	sseMutex.Lock()
	defer sseMutex.Unlock()

	message := fmt.Sprintf("data: {\"event\":\"processing_error\",\"title\":\"%s\",\"error\":\"%s\"}\n\n",
		escapeJSON(title), escapeJSON(errorMsg))
	for client := range sseClients {
		select {
		case client <- message:
		default:
			close(client)
			delete(sseClients, client)
		}
	}
}

func escapeJSON(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	s = strings.ReplaceAll(s, "\t", "\\t")
	return s
}

// ProcessNewsWithAI — точка входа для запуска обработки списка новостей
func ProcessNewsWithAI(ctx context.Context, articles []db.NewsArticle, ai *openrouter.RouterAi) error {
	return processWithRetry(ctx, articles, ai, 1)
}

// processWithRetry выполняет обработку с семафором и повторными попытками для упавших статей
func processWithRetry(ctx context.Context, articles []db.NewsArticle, ai *openrouter.RouterAi, attempt int) error {
	if len(articles) == 0 {
		return nil
	}

	factsDir := "facts"
	if err := os.MkdirAll(factsDir, 0755); err != nil {
		return fmt.Errorf("ошибка создания папки %s: %w", factsDir, err)
	}

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 3) // Ограничение: максимум 3 одновременных запроса к AI

	var (
		failedArticles []db.NewsArticle
		failedMutex    sync.Mutex
	)

	// Потокобезопасное добавление упавшей новости в список для ретрая
	recordFailure := func(a db.NewsArticle) {
		failedMutex.Lock()
		failedArticles = append(failedArticles, a)
		failedMutex.Unlock()
	}

	for _, article := range articles {
		semaphore <- struct{}{}
		wg.Add(1)

		go func(a db.NewsArticle) {
			defer func() {
				<-semaphore
				wg.Done()
			}()

			log.Printf("[AI] Попытка %d/%d | Обработка новости: %s", attempt, maxRetries, a.Title)

			factsJSON, err := ai.ExtractFacts(ctx, a.Title, a.Content)
			if err != nil {
				log.Printf("[AI] Ошибка вызова AI для %s: %v", a.URL, err)
				recordFailure(a)
				return
			}

			// Очистка markdown-блоков из ответа нейросети
			factsJSON = strings.TrimSpace(factsJSON)
			if strings.HasPrefix(factsJSON, "```json") {
				factsJSON = strings.TrimPrefix(factsJSON, "```json")
				factsJSON = strings.TrimSuffix(factsJSON, "```")
			} else if strings.HasPrefix(factsJSON, "```") {
				factsJSON = strings.TrimPrefix(factsJSON, "```")
				factsJSON = strings.TrimSuffix(factsJSON, "```")
			}
			factsJSON = strings.TrimSpace(factsJSON)

			var facts FactsResponse
			if err := json.Unmarshal([]byte(factsJSON), &facts); err != nil {
				log.Printf("[AI] Ошибка парсинга JSON для %s: %v. Ответ: %s", a.URL, err, factsJSON)
				recordFailure(a)
				return
			}

			newsFact := NewsFact{
				URL:           a.URL,
				Title:         a.Title,
				Dates:         facts.Dates,
				People:        facts.People,
				Organizations: facts.Organizations,
				Locations:     facts.Locations,
				MainThesis:    facts.MainThesis,
				KeyNumbers:    facts.KeyNumbers,
				Keywords:      facts.Keywords,
				Categories:    facts.Categories,
				ProcessedAt:   time.Now(),
			}

			urlHash := hashURL(a.URL)
			factFile := filepath.Join(factsDir, urlHash+".json")

			data, err := json.MarshalIndent(newsFact, "", "  ")
			if err != nil {
				log.Printf("[AI] Ошибка сериализации JSON для %s: %v", a.URL, err)
				recordFailure(a)
				return
			}

			if err := os.WriteFile(factFile, data, 0644); err != nil {
				log.Printf("[AI] Ошибка записи в файл для %s: %v", a.URL, err)
				recordFailure(a)
				return
			}

			log.Printf("[AI] Успешно обработана: %s (hash: %s)", a.Title, urlHash)
			notifySSEClients(urlHash)
		}(article)
	}

	wg.Wait()

	// Обработка неудачных попыток
	if len(failedArticles) > 0 {
		if attempt < maxRetries {
			log.Printf("[AI] Не удалось обработать %d новостей. Повтор через 2 секунды (попытка %d/%d)...", len(failedArticles), attempt+1, maxRetries)

			select {
			case <-time.After(2 * time.Second):
			case <-ctx.Done():
				return ctx.Err()
			}

			return processWithRetry(ctx, failedArticles, ai, attempt+1)
		}

		log.Printf("[AI] Завершено с ошибками: %d новостей не удалось обработать после %d попыток", len(failedArticles), maxRetries)

		for _, failed := range failedArticles {
			notifySSEError(failed.Title, "Не удалось обработать после нескольких попыток")
		}
	} else {
		log.Println("[AI] Все новости успешно обработаны")
	}

	return nil
}
