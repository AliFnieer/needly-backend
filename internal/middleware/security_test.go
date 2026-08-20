package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestSecurityMiddleware_ProductionHSTS(t *testing.T) {
	router := gin.New()
	router.Use(SecurityMiddleware(gin.ReleaseMode))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	hsts := w.Header().Get("Strict-Transport-Security")
	if hsts != "max-age=63072000; includeSubDomains" {
		t.Fatalf("expected HSTS header in release mode, got %q", hsts)
	}
}

func TestSecurityMiddleware_DevelopmentNoHSTS(t *testing.T) {
	router := gin.New()
	router.Use(SecurityMiddleware(gin.DebugMode))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	hsts := w.Header().Get("Strict-Transport-Security")
	if hsts != "" {
		t.Fatalf("expected no HSTS header in debug mode, got %q", hsts)
	}
}

func TestSecurityMiddleware_AlwaysSetsNosniff(t *testing.T) {
	for _, mode := range []string{gin.ReleaseMode, gin.DebugMode, gin.TestMode} {
		router := gin.New()
		router.Use(SecurityMiddleware(mode))
		router.GET("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if v := w.Header().Get("X-Content-Type-Options"); v != "nosniff" {
			t.Fatalf("mode=%s: expected nosniff, got %q", mode, v)
		}
	}
}

func TestSecurityMiddleware_AlwaysSetsFrameDeny(t *testing.T) {
	for _, mode := range []string{gin.ReleaseMode, gin.DebugMode, gin.TestMode} {
		router := gin.New()
		router.Use(SecurityMiddleware(mode))
		router.GET("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if v := w.Header().Get("X-Frame-Options"); v != "DENY" {
			t.Fatalf("mode=%s: expected DENY, got %q", mode, v)
		}
	}
}

func TestSecurityMiddleware_SetsAdditionalHeaders(t *testing.T) {
	router := gin.New()
	router.Use(SecurityMiddleware(gin.DebugMode))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	checks := map[string]string{
		"X-XSS-Protection":  "0",
		"Referrer-Policy":    "strict-origin-when-cross-origin",
		"Permissions-Policy": "camera=(), microphone=(), geolocation=()",
	}
	for header, expected := range checks {
		if v := w.Header().Get(header); v != expected {
			t.Fatalf("expected %s=%q, got %q", header, expected, v)
		}
	}
}

func TestSecurityMiddleware_SetsCSP(t *testing.T) {
	router := gin.New()
	router.Use(SecurityMiddleware(gin.DebugMode))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	csp := w.Header().Get("Content-Security-Policy")
	if csp == "" {
		t.Fatal("expected Content-Security-Policy header")
	}
}

func TestSecurityMiddleware_HandlerStillRuns(t *testing.T) {
	router := gin.New()
	router.Use(SecurityMiddleware(gin.DebugMode))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != "ok" {
		t.Fatalf("expected body ok, got %q", w.Body.String())
	}
}
