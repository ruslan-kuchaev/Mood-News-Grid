package handler

import (
	"io"
	"net/http"
	"net/url"
	"os"
	"time"
)

func NewsHandler(w http.ResponseWriter, r *http.Request) {
	searchQuery := r.URL.Query().Get("q")
	if searchQuery == "" {
		http.Error(w, "Search query is required", http.StatusBadRequest)
		return
	}

	apiToken := os.Getenv("API_Token")

	baseURL, err := url.Parse("https://api.perigon.io/v1/articles/all")
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	params := url.Values{}
	params.Add("apiKey", apiToken)
	params.Add("q", searchQuery)
	params.Add("language", "ru")
	params.Add("sortBy", "date")
	params.Add("size", "10")
	baseURL.RawQuery = params.Encode()

	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, baseURL.String(), nil)
	if err != nil {
		http.Error(w, "Failed to build request", http.StatusInternalServerError)
		return
	}

	client := &http.Client{Timeout: 10 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		http.Error(w, "Upstream request failed", http.StatusBadGateway)
		return
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		http.Error(w, "Failed to read upstream response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(res.StatusCode)
	w.Write(body)
}
