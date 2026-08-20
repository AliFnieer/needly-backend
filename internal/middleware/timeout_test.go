package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestRequestTimeoutMiddleware_NormalRequest(t *testing.T) {
	router := gin.New()
	router.Use(RequestTimeoutMiddleware(5 * time.Second))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestRequestTimeoutMiddleware_SkipsWebSocket(t *testing.T) {
	router := gin.New()
	router.Use(RequestTimeoutMiddleware(1 * time.Millisecond))
	router.GET("/test", func(c *gin.Context) {
		// If context deadline was applied, ctx.Err() would be set
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Upgrade", "websocket")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for websocket, got %d", w.Code)
	}
}

func TestRequestTimeoutMiddleware_SkipsWebSocketCaseInsensitive(t *testing.T) {
	router := gin.New()
	router.Use(RequestTimeoutMiddleware(1 * time.Millisecond))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Upgrade", "WebSocket")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for WebSocket header, got %d", w.Code)
	}
}

func TestRequestTimeoutMiddleware_Timeout(t *testing.T) {
	router := gin.New()
	router.Use(RequestTimeoutMiddleware(50 * time.Millisecond))
	router.GET("/test", func(c *gin.Context) {
		select {
		case <-c.Request.Context().Done():
		case <-time.After(5 * time.Second):
		}
		// Don't write a response - let the middleware check the deadline
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusGatewayTimeout {
		t.Fatalf("expected 504, got %d", w.Code)
	}
}

func TestRequestTimeoutMiddleware_ContextHasDeadline(t *testing.T) {
	timeout := 10 * time.Second
	router := gin.New()
	router.Use(RequestTimeoutMiddleware(timeout))
	router.GET("/test", func(c *gin.Context) {
		deadline, ok := c.Request.Context().Deadline()
		if !ok {
			t.Error("expected context to have a deadline")
			c.String(http.StatusInternalServerError, "no deadline")
			return
		}
		remaining := time.Until(deadline)
		// Should be roughly 10s (within 1s tolerance)
		if remaining < 9*time.Second || remaining > 11*time.Second {
			t.Errorf("expected deadline ~10s away, got %v", remaining)
			c.String(http.StatusInternalServerError, "bad deadline")
			return
		}
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestRequestTimeoutMiddleware_NoUpgradeHeaderPassesThrough(t *testing.T) {
	router := gin.New()
	router.Use(RequestTimeoutMiddleware(5 * time.Second))
	router.GET("/test", func(c *gin.Context) {
		// Verify no websocket skip
		upgrade := c.GetHeader("Upgrade")
		if upgrade == "websocket" || upgrade == "WebSocket" {
			t.Error("should not have websocket header")
		}
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}
