package middleware

import (
	"net/http"
	"sync"
	"time"
)

// RateLimiter 简单令牌桶限流器（单机版）
type RateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	rate     int // 每秒允许的请求数
	burst    int // 最大突发请求数
	cleanup  time.Duration
}

type visitor struct {
	tokens    float64
	lastCheck time.Time
}

// NewRateLimiter 创建限流器
func NewRateLimiter(rate, burst int) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*visitor),
		rate:     rate,
		burst:    burst,
		cleanup:  time.Minute,
	}
	go rl.cleanupVisitors()
	return rl
}

// RateLimitMiddleware 限流中间件
func RateLimitMiddleware(limit *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			uid := GetUserID(r.Context())
			if uid == "" {
				uid = r.RemoteAddr
			}

			if !limit.allow(uid) {
				http.Error(w, `{"error":"too many requests"}`, http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (rl *RateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, exists := rl.visitors[key]
	now := time.Now()

	if !exists {
		rl.visitors[key] = &visitor{
			tokens:    float64(rl.burst) - 1,
			lastCheck: now,
		}
		return true
	}

	elapsed := now.Sub(v.lastCheck).Seconds()
	v.tokens += elapsed * float64(rl.rate)
	if v.tokens > float64(rl.burst) {
		v.tokens = float64(rl.burst)
	}
	v.lastCheck = now

	if v.tokens < 1 {
		return false
	}

	v.tokens--
	return true
}

func (rl *RateLimiter) cleanupVisitors() {
	for {
		time.Sleep(rl.cleanup)
		rl.mu.Lock()
		for k, v := range rl.visitors {
			if time.Since(v.lastCheck) > rl.cleanup {
				delete(rl.visitors, k)
			}
		}
		rl.mu.Unlock()
	}
}
