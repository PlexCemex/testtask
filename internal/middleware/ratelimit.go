package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimiter ограничивает число запросов на пользователя (по IP, если нет auth).
type RateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*rate.Limiter
	rpm      int
}

func NewRateLimiter(requestsPerMinute int) *RateLimiter {
	return &RateLimiter{
		visitors: make(map[string]*rate.Limiter),
		rpm:      requestsPerMinute,
	}
}

func (rl *RateLimiter) getLimiter(key string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, exists := rl.visitors[key]
	if !exists {
		limiter = rate.NewLimiter(rate.Limit(float64(rl.rpm)/60.0), rl.rpm)
		rl.visitors[key] = limiter
	}
	return limiter
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.RemoteAddr
		if userID, ok := UserID(r.Context()); ok {
			key = "user:" + strconv.FormatInt(userID, 10)
		}

		limiter := rl.getLimiter(key)
		if !limiter.Allow() {
			http.Error(w, `{"error":"rate limit exceeded"}`, http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Cleanup сбрасывает карту лимитеров целиком, чтобы она не росла бесконечно.
// TODO: сделать по-нормальному - удалять только протухшие ключи, а не всё разом
func (rl *RateLimiter) Cleanup(interval time.Duration) {
	go func() {
		for range time.Tick(interval) {
			rl.mu.Lock()
			rl.visitors = make(map[string]*rate.Limiter)
			rl.mu.Unlock()
		}
	}()
}
