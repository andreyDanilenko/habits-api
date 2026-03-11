package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"backend/pkg/response"

	"github.com/gin-gonic/gin"
)

func TestAuthRateLimiter_BlocksAfterMaxAttempts(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Короткое окно для быстрого теста: 3 попытки за 100ms
	rl := NewAuthRateLimiter(3, 100*time.Millisecond)
	defer rl.Stop() // добавим метод Stop для остановки cleanup

	responder := response.NewResponder()
	r := gin.New()
	r.Use(rl.Middleware(responder, IPKeyExtractor))
	r.POST("/auth/login", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	// Фиксированный IP для теста
	req := func() *http.Request {
		r := httptest.NewRequest("POST", "/auth/login", nil)
		r.RemoteAddr = "192.168.1.100:12345"
		return r
	}

	// 3 запроса — должны проходить
	for i := 0; i < 3; i++ {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req())
		if w.Code != http.StatusOK {
			t.Errorf("request %d: expected 200, got %d", i+1, w.Code)
		}
	}

	// 4-й запрос — должен вернуть 429
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req())
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("request 4: expected 429, got %d", w.Code)
	}
}

func TestAuthRateLimiter_ResetsAfterWindow(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Окно 1 сек — достаточно для стабильного теста
	window := time.Second
	rl := NewAuthRateLimiter(2, window)
	defer rl.Stop()

	responder := response.NewResponder()
	r := gin.New()
	r.Use(rl.Middleware(responder, IPKeyExtractor))
	r.POST("/auth/login", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	makeReq := func() int {
		req := httptest.NewRequest("POST", "/auth/login", nil)
		req.RemoteAddr = "10.0.0.1:12345"
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w.Code
	}

	// 2 запроса — OK
	if makeReq() != http.StatusOK || makeReq() != http.StatusOK {
		t.Fatal("first 2 requests should succeed")
	}
	// 3-й — 429
	if makeReq() != http.StatusTooManyRequests {
		t.Fatal("3rd request should be rate limited")
	}
	// Ждём истечения окна
	time.Sleep(window + 100*time.Millisecond)
	// Снова должен пройти
	code := makeReq()
	if code != http.StatusOK {
		t.Fatalf("after window reset, expected 200, got %d", code)
	}
}

func TestAuthRateLimiter_DifferentIPsIndependent(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rl := NewAuthRateLimiter(2, time.Minute)
	defer rl.Stop()

	responder := response.NewResponder()
	r := gin.New()
	r.Use(rl.Middleware(responder, IPKeyExtractor))
	r.POST("/auth/login", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	makeReq := func(ip string) int {
		req := httptest.NewRequest("POST", "/auth/login", nil)
		req.RemoteAddr = ip + ":12345"
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w.Code
	}

	// IP1: 2 запроса OK
	if makeReq("1.1.1.1") != http.StatusOK || makeReq("1.1.1.1") != http.StatusOK {
		t.Fatal("IP1: first 2 should succeed")
	}
	// IP1: 3-й — 429
	if makeReq("1.1.1.1") != http.StatusTooManyRequests {
		t.Fatal("IP1: 3rd should be limited")
	}
	// IP2: свой лимит, 2 запроса OK
	if makeReq("2.2.2.2") != http.StatusOK || makeReq("2.2.2.2") != http.StatusOK {
		t.Fatal("IP2: first 2 should succeed (independent limit)")
	}
}

func TestAuthRateLimiter_LoginPerEmail(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 2 попытки на (IP, email) в минуту
	rl := NewAuthRateLimiter(2, time.Minute)
	defer rl.Stop()

	responder := response.NewResponder()
	r := gin.New()
	r.Use(rl.Middleware(responder, LoginKeyExtractor))
	r.POST("/auth/login", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	makeReq := func(email string) int {
		body := `{"email":"` + email + `","password":"x"}`
		req := httptest.NewRequest("POST", "/auth/login", strings.NewReader(body))
		req.RemoteAddr = "1.1.1.1:12345"
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w.Code
	}

	// user1@test.com: 2 OK
	if makeReq("user1@test.com") != http.StatusOK || makeReq("user1@test.com") != http.StatusOK {
		t.Fatal("user1: first 2 should succeed")
	}
	// user1@test.com: 3-й — 429
	if makeReq("user1@test.com") != http.StatusTooManyRequests {
		t.Fatal("user1: 3rd should be limited")
	}
	// user2@test.com: свой лимит, 2 OK
	if makeReq("user2@test.com") != http.StatusOK || makeReq("user2@test.com") != http.StatusOK {
		t.Fatal("user2: first 2 should succeed (independent per email)")
	}
}

func TestAuthRateLimiter_Headers(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rl := NewAuthRateLimiter(3, time.Minute)
	defer rl.Stop()

	responder := response.NewResponder()
	r := gin.New()
	r.Use(rl.Middleware(responder, IPKeyExtractor))
	r.POST("/auth/login", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest("POST", "/auth/login", nil)
	req.RemoteAddr = "1.1.1.1:12345"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Header().Get("X-RateLimit-Limit") != "3" {
		t.Errorf("X-RateLimit-Limit: got %s", w.Header().Get("X-RateLimit-Limit"))
	}
	if w.Header().Get("X-RateLimit-Remaining") != "2" {
		t.Errorf("X-RateLimit-Remaining: got %s", w.Header().Get("X-RateLimit-Remaining"))
	}
}
