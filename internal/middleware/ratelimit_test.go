package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AliFnieer/needly-backend/internal/config"
	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// TestRateLimiterDisabled verifies the no-op middleware passes requests through.
func TestRateLimiterDisabled(t *testing.T) {
	cfg := &config.RateLimitConfig{
		Enabled: false,
	}
	rl := NewRateLimiter(nil, cfg)

	router := gin.New()
	router.Use(rl.Middleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

// TestRateLimiterNilClient verifies middleware fails open when client is nil.
func TestRateLimiterNilClient(t *testing.T) {
	rl := NewRateLimiter(nil, nil)

	router := gin.New()
	router.Use(rl.Middleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

// TestRateLimiterClientKeyUser verifies user_id is preferred for the key.
func TestRateLimiterClientKeyUser(t *testing.T) {
	cfg := &config.RateLimitConfig{
		Enabled:       true,
		Requests:      10,
		WindowSeconds: 60,
	}
	rl := NewRateLimiter(nil, cfg)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)
	c.Set("user_id", float64(42))

	key := rl.clientKey(c)
	if key != "user:42" {
		t.Fatalf("expected key %q, got %q", "user:42", key)
	}
}

// TestRateLimiterClientKeyIP verifies IP fallback when no user is present.
func TestRateLimiterClientKeyIP(t *testing.T) {
	cfg := &config.RateLimitConfig{
		Enabled:       true,
		Requests:      10,
		WindowSeconds: 60,
	}
	rl := NewRateLimiter(nil, cfg)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)
	c.Request.RemoteAddr = "203.0.113.7:12345"

	key := rl.clientKey(c)
	if key == "" {
		t.Fatal("expected a non-empty client key")
	}
	if key != "ip:"+c.ClientIP() && key != "ip:203.0.113.7:12345" {
		t.Fatalf("unexpected client key %q", key)
	}
}

// TestRateLimiterClientKeyStringUser verifies string user IDs are supported.
func TestRateLimiterClientKeyStringUser(t *testing.T) {
	cfg := &config.RateLimitConfig{
		Enabled:       true,
		Requests:      10,
		WindowSeconds: 60,
	}
	rl := NewRateLimiter(nil, cfg)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)
	c.Set("user_id", "abc-123")

	key := rl.clientKey(c)
	if key != "user:abc-123" {
		t.Fatalf("expected key %q, got %q", "user:abc-123", key)
	}
}