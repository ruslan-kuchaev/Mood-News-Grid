package handler

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

func SSEHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	clientChan := make(chan string, 10)

	sseMutex.Lock()
	sseClients[clientChan] = true
	sseMutex.Unlock()

	log.Println("[SSE] Новый клиент подключен")

	defer func() {
		sseMutex.Lock()
		delete(sseClients, clientChan)
		close(clientChan)
		sseMutex.Unlock()
		log.Println("[SSE] Клиент отключен")
	}()

	fmt.Fprintf(w, "data: {\"event\":\"connected\"}\n\n")
	flusher.Flush()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case message := <-clientChan:
			fmt.Fprint(w, message)
			flusher.Flush()
		case <-ticker.C:
			fmt.Fprintf(w, ": keepalive\n\n")
			flusher.Flush()
		}
	}
}
