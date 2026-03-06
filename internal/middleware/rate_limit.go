package middleware

import (
	"net/http"
	"sync"
	"time"

	"backend/pkg/response"

	"github.com/gin-gonic/gin"
)

// ipAttempt хранит счётчик попыток и время окна.
type ipAttempt struct {
	count    int
	windowAt time.Time
}

// AuthRateLimiter ограничивает количество запросов на login/register с одного IP.
// Защита от брутфорса: sliding window — maxAttempts попыток в window.
type AuthRateLimiter struct {
	ips           map[string]*ipAttempt
	mu            sync.RWMutex
	maxAttempts   int
	window        time.Duration
	cleanupTicker *time.Ticker
}

// NewAuthRateLimiter создаёт лимитер: maxAttempts попыток в window.
// Пример: 10 попыток в минуту — NewAuthRateLimiter(10, time.Minute).
func NewAuthRateLimiter(maxAttempts int, window time.Duration) *AuthRateLimiter {
	if maxAttempts < 1 {
		maxAttempts = 5
	}
	if window < time.Second {
		window = time.Minute
	}
	rl := &AuthRateLimiter{
		ips:         make(map[string]*ipAttempt),
		maxAttempts: maxAttempts,
		window:      window,
	}
	// Периодическая очистка старых записей
	rl.cleanupTicker = time.NewTicker(window * 2)
	go rl.cleanup()
	return rl
}

func (rl *AuthRateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	a, ok := rl.ips[ip]
	if !ok {
		rl.ips[ip] = &ipAttempt{count: 1, windowAt: now}
		return true
	}

	// Окно истекло — сброс
	if now.Sub(a.windowAt) > rl.window {
		a.count = 1
		a.windowAt = now
		return true
	}

	a.count++
	return a.count <= rl.maxAttempts
}

func (rl *AuthRateLimiter) cleanup() {
	for range rl.cleanupTicker.C {
		rl.mu.Lock()
		now := time.Now()
		for ip, a := range rl.ips {
			if now.Sub(a.windowAt) > rl.window*2 {
				delete(rl.ips, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// Middleware возвращает gin middleware для ограничения запросов по IP.
func (rl *AuthRateLimiter) Middleware(responder *response.Responder) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if ip == "" {
			ip = "unknown"
		}

		if !rl.allow(ip) {
			responder.WriteError(c, http.StatusTooManyRequests, "Too many login attempts. Please try again later.")
			c.Abort()
			return
		}
		c.Next()
	}
}
