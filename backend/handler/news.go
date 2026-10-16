package handler

import (
	"backend/db"
	"backend/openrouter"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
)

var fastAI *openrouter.RouterAi

func SetFastAI(ai *openrouter.RouterAi) {
	fastAI = ai
}

type worldNewsResponse struct {
	Offset    int                `json:"offset"`
	Number    int                `json:"number"`
	Available int                `json:"available"`
	News      []worldNewsArticle `json:"news"`
}

type worldNewsArticle struct {
	ID          int      `json:"id"`
	Title       string   `json:"title"`
	Text        string   `json:"text"`
	URL         string   `json:"url"`
	Image       string   `json:"image"`
	PublishDate string   `json:"publish_date"`
	Author      string   `json:"author"`
	Authors     []string `json:"authors"`
	Language    string   `json:"language"`
}

type newsResponse struct {
	Status           int           `json:"status"`
	NumResults       int           `json:"numResults"`
	Articles         []newsArticle `json:"articles"`
	ProcessingErrors []string      `json:"processing_errors,omitempty"`
}

type newsArticle struct {
	Title    string `json:"title"`
	Author   string `json:"author"`
	PubDate  string `json:"pubdate"`
	Language string `json:"language"`
	Content  string `json:"content"`
	URL      string `json:"url"`
	URLHash  string `json:"url_hash"`
}

func NewsHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("=== [NewsHandler] Получен новый HTTP-запрос ===")

	deleteJsonArticle()

	apiToken := os.Getenv("API_Token")
	if apiToken == "" {
		log.Println("[NewsHandler] ВНИМАНИЕ: Переменная окружения API_Token пустая или не задана!")
		http.Error(w, "API_Token environment variable is missing", http.StatusInternalServerError)
		return
	}

	searchQuery := r.URL.Query().Get("q")
	if searchQuery == "" {
		log.Println("[NewsHandler] Ошибка: параметр поиска 'q' пустой")
		http.Error(w, "Search query is required", http.StatusBadRequest)
		return
	}
	log.Printf("[NewsHandler] Поисковый запрос: %q", searchQuery)

	baseURL, err := url.Parse("https://api.worldnewsapi.com/search-news")
	if err != nil {
		log.Printf("[NewsHandler] Ошибка парсинга URL: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	params := url.Values{}
	params.Add("text", searchQuery)
	params.Add("language", "ru")
	params.Add("sort", "publish-time")
	params.Add("sort-direction", "DESC")
	params.Add("number", "10")
	baseURL.RawQuery = params.Encode()

	log.Printf("[NewsHandler] Отправка запроса к World News API: %s", baseURL.String())

	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, baseURL.String(), nil)
	if err != nil {
		log.Printf("[NewsHandler] Ошибка создания запроса: %v", err)
		http.Error(w, "Failed to build request", http.StatusInternalServerError)
		return
	}

	req.Header.Set("x-api-key", apiToken)

	client := &http.Client{Timeout: 10 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		log.Printf("[NewsHandler] Ошибка соединения с World News API: %v", err)
		http.Error(w, "Upstream request failed", http.StatusBadGateway)
		return
	}
	defer res.Body.Close()

	log.Printf("[NewsHandler] Получен ответ от World News API. Код статуса: %d", res.StatusCode)

	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Printf("[NewsHandler] Ошибка чтения тела ответа: %v", err)
		http.Error(w, "Failed to read upstream response", http.StatusInternalServerError)
		return
	}

	if res.StatusCode != http.StatusOK {
		log.Printf("[NewsHandler] World News API вернул ошибку (Код %d): %s", res.StatusCode, string(body))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(res.StatusCode)
		w.Write(body)
		return
	}

	var parsed worldNewsResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		log.Printf("[NewsHandler] Ошибка JSON парсинга ответа: %v. Ответ API: %s", err, string(body))
		http.Error(w, "Failed to parse upstream response", http.StatusInternalServerError)
		return
	}

	log.Printf("[NewsHandler] Успешно получено новостей от API: %d (Всего доступно по запросу: %d)", len(parsed.News), parsed.Available)

	dbArticles, jsonArticles := normalizeArticles(parsed.News)

	log.Printf("[NewsHandler] Сохранение %d записей в базу данных...", len(dbArticles))
	if err := db.SaveArticles(r.Context(), dbArticles); err != nil {
		log.Printf("[NewsHandler] Ошибка при сохранении в БД: %v", err)
		http.Error(w, "Failed to save articles", http.StatusInternalServerError)
		return
	}
	log.Println("[NewsHandler] Новости успешно сохранены в БД")

	if fastAI != nil {
		log.Println("[NewsHandler] Запуск AI обработки новостей...")
		go func() {
			// Используем контекст с таймаутом для фоновой задачи
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()

			if err := ProcessNewsWithAI(ctx, dbArticles, fastAI); err != nil {
				if err == context.DeadlineExceeded {
					log.Printf("[NewsHandler] AI обработка превысила таймаут")
				} else {
					log.Printf("[NewsHandler] Ошибка AI обработки: %v", err)
				}
			}
		}()
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(newsResponse{
		Status:     http.StatusOK,
		NumResults: parsed.Available,
		Articles:   jsonArticles,
	}); err != nil {
		log.Printf("[NewsHandler] Ошибка кодирования JSON для клиента: %v", err)
	}

	log.Println("=== [NewsHandler] Запрос успешно обработан ===")
}

func normalizeArticles(items []worldNewsArticle) ([]db.NewsArticle, []newsArticle) {
	dbArticles := make([]db.NewsArticle, 0, len(items))
	jsonArticles := make([]newsArticle, 0, len(items))

	const worldNewsDateFormat = "2006-01-02 15:04:05"

	for _, item := range items {
		title := orUnknown(item.Title)
		content := orUnknown(item.Text)
		language := orUnknown(item.Language)
		urlStr := orUnknown(item.URL)

		authorRaw := item.Author
		if authorRaw == "" && len(item.Authors) > 0 {
			authorRaw = strings.Join(item.Authors, ", ")
		}
		author := orUnknown(authorRaw)

		pubDateStr := orUnknown(item.PublishDate)
		pubDate, err := time.Parse(worldNewsDateFormat, item.PublishDate)
		if err != nil {
			pubDate, err = time.Parse(time.RFC3339, item.PublishDate)
			if err != nil {
				pubDate = time.Time{}
			}
		}

		dbArticles = append(dbArticles, db.NewsArticle{
			Title:    title,
			Author:   author,
			PubDate:  pubDate,
			Language: language,
			Content:  content,
			URL:      urlStr,
		})

		// Генерация url_hash: если URL пустой или "не известно", используем UUID
		var urlHash string
		if urlStr == unknownValue || strings.TrimSpace(urlStr) == "" {
			urlHash = strings.ReplaceAll(uuid.New().String(), "-", "")
		} else {
			hash := md5.Sum([]byte(urlStr))
			urlHash = hex.EncodeToString(hash[:])
		}

		jsonArticles = append(jsonArticles, newsArticle{
			Title:    title,
			Author:   author,
			PubDate:  pubDateStr,
			Language: language,
			Content:  content,
			URL:      urlStr,
			URLHash:  urlHash,
		})
	}

	return dbArticles, jsonArticles
}

const unknownValue = "не известно"

func orUnknown(value string) string {
	if strings.TrimSpace(value) == "" {
		return unknownValue
	}
	return value
}

func deleteJsonArticle() {
	dir := "facts"

	if err := os.RemoveAll(dir); err != nil {
		log.Printf("[DeleteJsonArticle] Ошибка удаления директории %s: %v", dir, err)
	}
}
