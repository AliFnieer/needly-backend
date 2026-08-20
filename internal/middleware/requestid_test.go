package middleware

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

var uuidRe = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func TestRequestIDMiddleware_GeneratesID(t *testing.T) {
	router := gin.New()
	router.Use(RequestIDMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	rid := w.Header().Get(RequestIDHeader)
	if rid == "" {
		t.Fatal("expected X-Request-ID to be set")
	}
	if !uuidRe.MatchString(rid) {
		t.Fatalf("expected valid UUIDv4, got %q", rid)
	}
}

func TestRequestIDMiddleware_HonorsIncomingID(t *testing.T) {
	incoming := "550e8400-e29b-41d4-a716-446655440000"
	router := gin.New()
	router.Use(RequestIDMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set(RequestIDHeader, incoming)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	rid := w.Header().Get(RequestIDHeader)
	if rid != incoming {
		t.Fatalf("expected %q, got %q", incoming, rid)
	}
}

func TestRequestIDMiddleware_SetsContextValue(t *testing.T) {
	var contextRID string
	router := gin.New()
	router.Use(RequestIDMiddleware())
	router.GET("/test", func(c *gin.Context) {
		val, ok := c.Get(RequestIDKey)
		if !ok {
			t.Fatal("request_id not found in context")
		}
		contextRID = val.(string)
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	headerRID := w.Header().Get(RequestIDHeader)
	if contextRID != headerRID {
		t.Fatalf("context request_id %q != header %q", contextRID, headerRID)
	}
}

func TestRequestIDMiddleware_UniquePerRequest(t *testing.T) {
	router := gin.New()
	router.Use(RequestIDMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	ids := make(map[string]bool)
	for i := 0; i < 50; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		rid := w.Header().Get(RequestIDHeader)
		if ids[rid] {
			t.Fatalf("duplicate request ID %q on iteration %d", rid, i)
		}
		ids[rid] = true
	}
}
