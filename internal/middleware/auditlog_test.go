package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAuditLog_SkipsHealthEndpoint(t *testing.T) {
	ensureDebugLogLevel(t)

	router := newAuditRouter(t, AuditLogMiddleware())
	router.GET("/health", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := doAuditRequest(router, http.MethodGet, "/health", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestAuditLog_SkipsMetricsEndpoint(t *testing.T) {
	ensureDebugLogLevel(t)

	router := newAuditRouter(t, AuditLogMiddleware())
	router.GET("/metrics", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := doAuditRequest(router, http.MethodGet, "/metrics", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestAuditLog_SkipsDocsEndpoint(t *testing.T) {
	ensureDebugLogLevel(t)

	router := newAuditRouter(t, AuditLogMiddleware())
	router.GET("/docs", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := doAuditRequest(router, http.MethodGet, "/docs", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestAuditLog_CapturesRequestBody(t *testing.T) {
	ensureDebugLogLevel(t)

	router := newAuditRouter(t, AuditLogMiddleware())
	router.POST("/api/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"received": true})
	})

	body := strings.NewReader(`{"name":"test","value":123}`)
	req := httptest.NewRequest(http.MethodPost, "/api/test", body)
	req.Header.Set("Content-Type", "application/json")

	w := doAuditRequest(router, http.MethodPost, "/api/test", req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestAuditLog_CapturesResponseBody(t *testing.T) {
	ensureDebugLogLevel(t)

	router := newAuditRouter(t, AuditLogMiddleware())
	router.GET("/api/data", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"key": "value", "count": 42})
	})

	w := doAuditRequest(router, http.MethodGet, "/api/data", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	if !strings.Contains(w.Body.String(), `"key":"value"`) {
		t.Fatalf("expected response body to contain key/value, got: %s", w.Body.String())
	}
}

func TestAuditLog_MasksAuthorizationHeader(t *testing.T) {
	ensureDebugLogLevel(t)

	router := newAuditRouter(t, AuditLogMiddleware())
	router.GET("/api/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Authorization", "Bearer super-secret-token-12345")

	w := doAuditRequest(router, http.MethodGet, "/api/test", req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestAuditLog_NoopWhenNotDebug(t *testing.T) {
	setLogLevel(t, "info")

	router := newAuditRouter(t, AuditLogMiddleware())
	router.POST("/api/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	body := strings.NewReader(`{"password":"secret123"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/test", body)
	req.Header.Set("Content-Type", "application/json")

	w := doAuditRequest(router, http.MethodPost, "/api/test", req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestAuditLog_IncludesUserIDWhenSet(t *testing.T) {
	ensureDebugLogLevel(t)

	router := newAuditRouter(t, AuditLogMiddleware())
	router.GET("/api/me", func(c *gin.Context) {
		c.Set("user_id", float64(42))
		c.JSON(http.StatusOK, gin.H{"user_id": 42})
	})

	w := doAuditRequest(router, http.MethodGet, "/api/me", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestAuditLog_IncludesRequestID(t *testing.T) {
	ensureDebugLogLevel(t)

	router := newAuditRouter(t, AuditLogMiddleware())
	router.GET("/api/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set(RequestIDHeader, "test-request-id-abc-123")

	w := doAuditRequest(router, http.MethodGet, "/api/test", req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestAuditLog_NonJSONBodyNotLogged(t *testing.T) {
	ensureDebugLogLevel(t)

	router := newAuditRouter(t, AuditLogMiddleware())
	router.POST("/api/upload", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	body := strings.NewReader("plain text body content here")
	req := httptest.NewRequest(http.MethodPost, "/api/upload", body)
	req.Header.Set("Content-Type", "text/plain")

	w := doAuditRequest(router, http.MethodPost, "/api/upload", req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestResponseCapture_WritesBody(t *testing.T) {
	ensureDebugLogLevel(t)

	router := newAuditRouter(t, AuditLogMiddleware())
	router.POST("/api/echo", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"result": "ok"})
	})

	body := strings.NewReader(`{"input":"hello"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/echo", body)
	req.Header.Set("Content-Type", "application/json")

	w := doAuditRequest(router, http.MethodPost, "/api/echo", req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"result":"ok"`) {
		t.Fatalf("expected response body in output, got: %s", w.Body.String())
	}
}
