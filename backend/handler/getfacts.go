package handler

import (
	"backend/validator"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func GetFactsHandler(w http.ResponseWriter, r *http.Request) {
	urlHash := strings.TrimPrefix(r.URL.Path, "/facts/")
	if urlHash == "" {
		http.Error(w, "URL hash is required", http.StatusBadRequest)
		return
	}

	// Валидация urlHash для предотвращения path traversal
	if !validator.IsValidHash(urlHash) {
		log.Printf("[Facts] Недопустимый формат url_hash: %s", urlHash)
		http.Error(w, "Invalid URL hash format", http.StatusBadRequest)
		return
	}

	factFile := filepath.Join("facts", urlHash+".json")

	data, err := os.ReadFile(factFile)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "Facts not found or not ready yet", http.StatusNotFound)
			return
		}
		log.Printf("[Facts] Ошибка чтения файла %s: %v", factFile, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	var fact NewsFact
	if err := json.Unmarshal(data, &fact); err != nil {
		log.Printf("[Facts] Ошибка парсинга JSON из %s: %v", factFile, err)
		http.Error(w, "Invalid fact data", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(fact)
}
