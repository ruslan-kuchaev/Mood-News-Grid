package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"backend/db"
	"backend/handler"
	"backend/middleware"
	"backend/openrouter"
	"backend/utils"

	"github.com/joho/godotenv"
)

var fastAI *openrouter.RouterAi

func main() {
	// Загружаем .env если он есть, но не падаем если его нет (для production)
	if err := godotenv.Load(); err != nil {
		log.Println("Файл .env не найден или не может быть загружен (используются переменные окружения системы)")
	}

	if err := db.Init(); err != nil {
		log.Fatalf("db init: %v", err)
	}
	defer db.Close()

	log.Println("Инициализация AI моделей...")
	fastAI = utils.MustInitAI(context.Background(), "nvidia/nemotron-3-ultra-550b-a55b:free")
	
	smartAI := utils.MustInitAI(context.Background(), "nvidia/nemotron-3-ultra-550b-a55b:free")

	if fastAI == nil || smartAI == nil {
		log.Println("ВНИМАНИЕ: Не удалось инициализировать AI модели. Сервер запустится, но функции AI будут недоступны.")
	} else {
		log.Printf("AI модели инициализированы успешно")
	}

	handler.SetFastAI(fastAI)
	handler.SetSmartAI(smartAI)

	// Создаем rate limiter: 20 запросов в минуту с одного IP
	rateLimiter := middleware.NewSimpleRateLimiter(20, 1*time.Minute)
	rateLimit := middleware.RateLimitMiddleware(rateLimiter)

	mux := http.NewServeMux()

	corsHandler := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-API-Key")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next(w, r)
		}
	}

	mux.HandleFunc("/", corsHandler(rateLimit(handler.NewsHandler)))
	mux.HandleFunc("/events", handler.SSEHandler)
	mux.HandleFunc("/facts/", corsHandler(handler.GetFactsHandler))
	mux.HandleFunc("/rewrite", corsHandler(rateLimit(handler.RewriteHandler)))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8088"
	}

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	go func() {
		log.Printf("Server started on http://localhost:%s", port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Println("Shutdown signal received, draining connections...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped cleanly")
}
