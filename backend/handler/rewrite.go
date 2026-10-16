package handler

import (
	"backend/openrouter"
	"backend/validator"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type RewriteRequest struct {
	URLHash string `json:"url_hash"`
	Emotion string `json:"emotion"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

type RewriteResponse struct {
	OriginalContent  string `json:"original_content"`
	RewrittenContent string `json:"rewritten_content"`
	Emotion          string `json:"emotion"`
}

var smartAI *openrouter.RouterAi

func SetSmartAI(ai *openrouter.RouterAi) {
	smartAI = ai
}

func RewriteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Ограничиваем размер тела запроса для защиты от DoS
	r.Body = http.MaxBytesReader(w, r.Body, 10*1024*1024) // 10 MB

	var req RewriteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[Rewrite] Ошибка парсинга запроса: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.URLHash == "" || req.Emotion == "" {
		http.Error(w, "url_hash and emotion are required", http.StatusBadRequest)
		return
	}

	// Валидация urlHash для предотвращения path traversal
	if !validator.IsValidHash(req.URLHash) {
		log.Printf("[Rewrite] Недопустимый формат url_hash: %s", req.URLHash)
		http.Error(w, "Invalid URL hash format", http.StatusBadRequest)
		return
	}

	// Валидация emotion (whitelist)
	if !validator.IsValidEmotion(req.Emotion) {
		log.Printf("[Rewrite] Недопустимая эмоция: %s", req.Emotion)
		http.Error(w, "Invalid emotion. Allowed: happy, sad, angry", http.StatusBadRequest)
		return
	}

	log.Printf("[Rewrite] Запрос на переписывание: url_hash=%s, emotion=%s", req.URLHash, req.Emotion)

	factsFile := filepath.Join("facts", req.URLHash+".json")
	factsData, err := os.ReadFile(factsFile)
	if err != nil {
		log.Printf("[Rewrite] Не удалось прочитать facts для %s: %v", req.URLHash, err)
		http.Error(w, "Facts not found", http.StatusNotFound)
		return
	}

	var facts NewsFact
	if err := json.Unmarshal(factsData, &facts); err != nil {
		log.Printf("[Rewrite] Ошибка парсинга facts: %v", err)
		http.Error(w, "Invalid facts data", http.StatusInternalServerError)
		return
	}

	cachedFile := filepath.Join("facts", fmt.Sprintf("%s_rewritten_%s.json", req.URLHash, req.Emotion))
	if data, err := os.ReadFile(cachedFile); err == nil {
		var cached RewriteResponse
		if json.Unmarshal(data, &cached) == nil {
			log.Printf("[Rewrite] Возвращаем кэшированный результат для %s/%s", req.URLHash, req.Emotion)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(cached)
			return
		}
	}

	if smartAI == nil {
		log.Println("[Rewrite] SmartAI не инициализирован")
		http.Error(w, "AI service not available", http.StatusServiceUnavailable)
		return
	}

	rewrittenText, err := rewriteWithEmotion(r.Context(), smartAI, req.Title, req.Content, req.Emotion, facts)
	if err != nil {
		log.Printf("[Rewrite] Ошибка переписывания: %v", err)
		http.Error(w, fmt.Sprintf("Failed to rewrite: %v", err), http.StatusInternalServerError)
		return
	}

	response := RewriteResponse{
		OriginalContent:  req.Content,
		RewrittenContent: rewrittenText,
		Emotion:          req.Emotion,
	}

	responseData, _ := json.MarshalIndent(response, "", "  ")
	if err := os.WriteFile(cachedFile, responseData, 0644); err != nil {
		log.Printf("[Rewrite] Ошибка записи кэша: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	log.Printf("[Rewrite] Успешно переписан текст для %s с эмоцией %s", req.URLHash, req.Emotion)
}

func rewriteWithEmotion(ctx context.Context, ai *openrouter.RouterAi, title, content, emotion string, facts NewsFact) (string, error) {
	emotionMap := map[string]string{
		"happy": "позитивной, радостной, воодушевляющей",
		"sad":   "грустной, меланхоличной, сочувствующей",
		"angry": "критичной, возмущенной, резкой",
	}

	emotionDesc, ok := emotionMap[emotion]
	if !ok {
		emotionDesc = "нейтральной"
	}

	factsJSON, _ := json.MarshalIndent(facts, "", "  ")

	prompt := fmt.Sprintf(`Перепиши следующую новость с %s интонацией.

КРИТИЧЕСКИ ВАЖНО:
1. НЕ ИЗМЕНЯЙ даты, время, числа, имена людей, названия организаций и мест
2. СОХРАНИ основную мысль и все факты
3. ИЗМЕНИ только стиль изложения и эмоциональную окраску
4. Используй факты ниже как справочник того, что должно остаться неизменным
5. Верни ТОЛЬКО переписанный текст новости, БЕЗ заголовков, БЕЗ JSON, БЕЗ дополнительных пояснений

ФАКТЫ (НЕ МЕНЯТЬ):
%s

Заголовок: %s

Оригинальный текст:
%s

Переписанный текст:`, emotionDesc, string(factsJSON), title, content)

	resp, err := ai.Rewrite(ctx, prompt)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(resp), nil
}
