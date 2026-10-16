package middleware

import (
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

// SimpleRateLimiter - простой in-memory rate limiter
type SimpleRateLimiter struct {
	requests map[string][]time.Time
	mu       sync.Mutex
	limit    int
	window   time.Duration
}

func NewSimpleRateLimiter(limit int, window time.Duration) *SimpleRateLimiter {
	limiter := &SimpleRateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}

	// Очистка старых записей каждую минуту
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			limiter.cleanup()
		}
	}()

	return limiter
}

func (rl *SimpleRateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	for ip, times := range rl.requests {
		var valid []time.Time
		for _, t := range times {
			if now.Sub(t) < rl.window {
				valid = append(valid, t)
			}
		}
		if len(valid) == 0 {
			delete(rl.requests, ip)
		} else {
			rl.requests[ip] = valid
		}
	}
}

func (rl *SimpleRateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	// Фильтруем старые запросы
	var valid []time.Time
	for _, t := range rl.requests[ip] {
		if now.Sub(t) < rl.window {
			valid = append(valid, t)
		}
	}

	if len(valid) >= rl.limit {
		return false
	}

	valid = append(valid, now)
	rl.requests[ip] = valid
	return true
}

// RateLimitMiddleware - middleware для rate limiting
func RateLimitMiddleware(limiter *SimpleRateLimiter) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ip := r.RemoteAddr
			if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
				ip = forwarded
			}

			if !limiter.Allow(ip) {
				log.Printf("[RateLimit] Превышен лимит запросов для IP: %s", ip)
				http.Error(w, "Rate limit exceeded. Please try again later.", http.StatusTooManyRequests)
				return
			}

			next(w, r)
		}
	}
}

// SimpleAuthMiddleware - простая проверка API ключа (опционально)
func SimpleAuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	apiKey := os.Getenv("API_AUTH_KEY")

	// Если ключ не установлен, не требуем авторизацию (для development)
	if apiKey == "" {
		return next
	}

	return func(w http.ResponseWriter, r *http.Request) {
		providedKey := r.Header.Get("X-API-Key")
		if providedKey != apiKey {
			log.Printf("[Auth] Неверный API ключ от IP: %s", r.RemoteAddr)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}
