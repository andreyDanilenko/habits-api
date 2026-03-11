package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"backend/pkg/response"

	"github.com/gin-gonic/gin"
)

// attempt хранит счётчик попыток и время окна.
type attempt struct {
	count    int
	windowAt time.Time
}

// AuthRateLimiter ограничивает количество запросов на auth-эндпоинты.
// Защита от брутфорса: fixed window — maxAttempts попыток в window.
type AuthRateLimiter struct {
	keys          map[string]*attempt
	mu            sync.RWMutex
	maxAttempts   int
	window        time.Duration
	cleanupTicker *time.Ticker
	stopCh        chan struct{}
	stopped       atomic.Bool
}

// AllowResult результат проверки лимита.
type AllowResult struct {
	Allowed    bool
	Remaining  int
	RetryAfter time.Duration
}

// KeyExtractor извлекает ключ для rate limit из контекста (например, IP или IP+email).
type KeyExtractor func(c *gin.Context) string

// IPKeyExtractor возвращает ключ по IP.
func IPKeyExtractor(c *gin.Context) string {
	ip := c.ClientIP()
	if ip == "" {
		ip = "unknown"
	}
	return ip
}

// LoginKeyExtractor возвращает ключ IP:email для login — защита от брутфорса по конкретному email.
// Читает body, извлекает email, восстанавливает body для handler.
func LoginKeyExtractor(c *gin.Context) string {
	ip := c.ClientIP()
	if ip == "" {
		ip = "unknown"
	}
	email := extractEmailFromBody(c)
	if email == "" {
		return ip
	}
	return ip + ":" + strings.ToLower(strings.TrimSpace(email))
}

func extractEmailFromBody(c *gin.Context) string {
	if c.Request.Body == nil {
		return ""
	}
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return ""
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(body))
	var req struct {
		Email string `json:"email"`
	}
	if json.Unmarshal(body, &req) != nil {
		return ""
	}
	return req.Email
}

// NewAuthRateLimiter создаёт лимитер: maxAttempts попыток в window.
func NewAuthRateLimiter(maxAttempts int, window time.Duration) *AuthRateLimiter {
	if maxAttempts < 1 {
		maxAttempts = 5
	}
	if window < time.Second {
		window = time.Minute
	}
	rl := &AuthRateLimiter{
		keys:        make(map[string]*attempt),
		maxAttempts: maxAttempts,
		window:      window,
		stopCh:      make(chan struct{}),
	}
	rl.cleanupTicker = time.NewTicker(window * 2)
	go rl.cleanup()
	return rl
}

// Stop останавливает cleanup-горутину.
func (rl *AuthRateLimiter) Stop() {
	if rl.stopped.Swap(true) {
		return
	}
	rl.cleanupTicker.Stop()
	close(rl.stopCh)
}

func (rl *AuthRateLimiter) allowWithInfo(key string) AllowResult {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	a, ok := rl.keys[key]
	if !ok {
		rl.keys[key] = &attempt{count: 1, windowAt: now}
		return AllowResult{
			Allowed:    true,
			Remaining:  rl.maxAttempts - 1,
			RetryAfter: rl.window,
		}
	}

	if now.Sub(a.windowAt) >= rl.window {
		a.count = 1
		a.windowAt = now
		return AllowResult{
			Allowed:    true,
			Remaining:  rl.maxAttempts - 1,
			RetryAfter: rl.window,
		}
	}

	a.count++
	remaining := rl.maxAttempts - a.count
	if remaining < 0 {
		remaining = 0
	}
	retryAfter := rl.window - now.Sub(a.windowAt)
	if retryAfter < 0 {
		retryAfter = 0
	}
	return AllowResult{
		Allowed:    a.count <= rl.maxAttempts,
		Remaining:  remaining,
		RetryAfter: retryAfter,
	}
}

func (rl *AuthRateLimiter) cleanup() {
	for {
		select {
		case <-rl.stopCh:
			return
		case <-rl.cleanupTicker.C:
			rl.mu.Lock()
			now := time.Now()
			for k, a := range rl.keys {
				if now.Sub(a.windowAt) > rl.window*2 {
					delete(rl.keys, k)
				}
			}
			rl.mu.Unlock()
		}
	}
}

const (
	headerRateLimitLimit     = "X-RateLimit-Limit"
	headerRateLimitRemaining = "X-RateLimit-Remaining"
	headerRetryAfter         = "Retry-After"
)

// AuthRateLimitConfig конфигурация rate limit для auth-маршрутов.
// Разные лимиты: login жёстче (брутфорс по email), register/refresh мягче.
type AuthRateLimitConfig struct {
	LoginLimiter    *AuthRateLimiter // 5/min per (IP, email)
	RegisterLimiter *AuthRateLimiter // 10/min per IP
	RefreshLimiter  *AuthRateLimiter // 30/min per IP
	LogoutLimiter   *AuthRateLimiter // 30/min per IP
}

// Middleware возвращает gin middleware с поддержкой KeyExtractor и заголовков X-RateLimit-*.
func (rl *AuthRateLimiter) Middleware(responder *response.Responder, extractor KeyExtractor) gin.HandlerFunc {
	if extractor == nil {
		extractor = IPKeyExtractor
	}
	return func(c *gin.Context) {
		key := extractor(c)
		res := rl.allowWithInfo(key)

		c.Header(headerRateLimitLimit, strconv.Itoa(rl.maxAttempts))
		c.Header(headerRateLimitRemaining, strconv.Itoa(res.Remaining))

		if !res.Allowed {
			sec := int(res.RetryAfter.Seconds())
			if sec < 1 {
				sec = 1
			}
			c.Header(headerRetryAfter, strconv.Itoa(sec))
			responder.WriteError(c, http.StatusTooManyRequests, "Too many attempts. Please try again later.")
			c.Abort()
			return
		}
		c.Next()
	}
}
