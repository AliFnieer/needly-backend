package idempotency_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/AliFnieer/needly-backend/internal/idempotency"
	"github.com/AliFnieer/needly-backend/internal/testutil"
	"github.com/gin-gonic/gin"
)

// newTestRouter wires Middleware behind a fake auth step that reads the
// X-Test-User header, mirroring AuthMiddleware setting user_id as float64.
func newTestRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db := testutil.SetupTestDB(t)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		switch c.GetHeader("X-Test-User") {
		case "1":
			c.Set("user_id", float64(1))
		case "2":
			c.Set("user_id", float64(2))
		}
		c.Next()
	})
	r.Use(idempotency.Middleware(db))
	return r
}

func do(r *gin.Engine, method, path, key, user string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	if key != "" {
		req.Header.Set(idempotency.HeaderName, key)
	}
	if user != "" {
		req.Header.Set("X-Test-User", user)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func countingCreate(t *testing.T) (*gin.Engine, *int) {
	t.Helper()
	count := 0
	r := newTestRouter(t)
	r.POST("/things", func(c *gin.Context) {
		count++
		c.JSON(http.StatusCreated, gin.H{"call": count})
	})
	r.POST("/boom", func(c *gin.Context) {
		count++
		c.JSON(http.StatusInternalServerError, gin.H{"error": "boom"})
	})
	r.GET("/things", func(c *gin.Context) {
		count++
		c.JSON(http.StatusOK, gin.H{"call": count})
	})
	return r, &count
}

func TestMiddleware_ReplaysSameKeyInsteadOfReExecuting(t *testing.T) {
	r, count := countingCreate(t)

	first := do(r, http.MethodPost, "/things", "abc", "1")
	second := do(r, http.MethodPost, "/things", "abc", "1")

	if first.Code != http.StatusCreated || second.Code != http.StatusCreated {
		t.Fatalf("expected 201s, got %d and %d", first.Code, second.Code)
	}
	if !strings.Contains(first.Body.String(), `"call":1`) {
		t.Fatalf("unexpected first body: %s", first.Body.String())
	}
	if first.Body.String() != second.Body.String() {
		t.Fatalf("replayed body differs:\nfirst:  %s\nsecond: %s", first.Body.String(), second.Body.String())
	}
	if second.Header().Get("Idempotency-Replayed") != "true" {
		t.Fatal("expected Idempotency-Replayed header on replay")
	}
	if *count != 1 {
		t.Fatalf("handler must execute once, executed %d times", *count)
	}
}

func TestMiddleware_DifferentKeysExecuteTwice(t *testing.T) {
	r, _ := countingCreate(t)

	do(r, http.MethodPost, "/things", "key-1", "1")
	do(r, http.MethodPost, "/things", "key-2", "1")

	body := do(r, http.MethodPost, "/things", "key-3", "1").Body.String()
	if !strings.Contains(body, `"call":3`) {
		t.Fatalf("expected third execution, got %s", body)
	}
}

func TestMiddleware_NoKeyPassesThrough(t *testing.T) {
	r, count := countingCreate(t)

	first := do(r, http.MethodPost, "/things", "", "1")
	second := do(r, http.MethodPost, "/things", "", "1")

	if *count != 2 {
		t.Fatalf("requests without keys must execute every time, calls=%d", *count)
	}
	if !strings.Contains(first.Body.String(), `"call":1`) ||
		!strings.Contains(second.Body.String(), `"call":2`) {
		t.Fatalf("unexpected bodies: %s / %s", first.Body.String(), second.Body.String())
	}
}

func TestMiddleware_GETIgnored(t *testing.T) {
	r, count := countingCreate(t)

	do(r, http.MethodGet, "/things", "same-key", "1")
	do(r, http.MethodGet, "/things", "same-key", "1")
	last := do(r, http.MethodGet, "/things", "same-key", "1")

	if *count != 3 {
		t.Fatalf("GET must never be replayed, calls=%d", *count)
	}
	if last.Header().Get("Idempotency-Replayed") == "true" {
		t.Fatal("GET responses must not be replayed")
	}
}

func TestMiddleware_ServerErrorsNotStored(t *testing.T) {
	r, count := countingCreate(t)

	first := do(r, http.MethodPost, "/boom", "k", "1")
	if first.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", first.Code)
	}
	second := do(r, http.MethodPost, "/boom", "k", "1")
	if second.Header().Get("Idempotency-Replayed") == "true" {
		t.Fatal("500 responses must not be stored/replayed")
	}
	if *count != 2 {
		t.Fatalf("failed requests must stay retryable, calls=%d", *count)
	}
}

func TestMiddleware_UsersIsolated(t *testing.T) {
	r, _ := countingCreate(t)

	do(r, http.MethodPost, "/things", "shared", "1")
	do(r, http.MethodPost, "/things", "shared", "2")
	replayForUser1 := do(r, http.MethodPost, "/things", "shared", "1")

	if !strings.Contains(replayForUser1.Body.String(), `"call":1`) {
		t.Fatalf("user 2's record must not replay for user 1, got %s", replayForUser1.Body.String())
	}
	if replayForUser1.Header().Get("Idempotency-Replayed") != "true" {
		t.Fatal("user 1 should still get its own stored response replayed")
	}
}

func TestMiddleware_UnauthenticatedPassthrough(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := testutil.SetupTestDB(t)

	count := 0
	r := gin.New()
	r.Use(idempotency.Middleware(db))
	r.POST("/x", func(c *gin.Context) {
		count++
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	do(r, http.MethodPost, "/x", "k", "")
	do(r, http.MethodPost, "/x", "k", "")

	if count != 2 {
		t.Fatalf("unauthenticated requests must execute every time, calls=%d", count)
	}
}

func TestCleanupOlderThan_RemovesOnlyStaleRows(t *testing.T) {
	db := testutil.SetupTestDB(t)

	stale := idempotency.IdempotencyKey{
		UserID: 7, Route: "/api/v1/x", KeyHash: "aaaa",
		StatusCode: 201, ContentType: "application/json; charset=utf-8",
		ResponseBody: `{"ok":true}`, CreatedAt: time.Now().Add(-48 * time.Hour),
	}
	fresh := idempotency.IdempotencyKey{
		UserID: 7, Route: "/api/v1/y", KeyHash: "bbbb",
		StatusCode: 201, ContentType: "application/json; charset=utf-8",
		ResponseBody: `{"ok":true}`, CreatedAt: time.Now(),
	}
	for _, rec := range []idempotency.IdempotencyKey{stale, fresh} {
		if err := db.Create(&rec).Error; err != nil {
			t.Fatal(err)
		}
	}

	n, err := idempotency.CleanupOlderThan(db, time.Now().Add(-idempotency.Retention))
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("expected to delete 1 row, deleted %d", n)
	}

	var count int64
	db.Model(&idempotency.IdempotencyKey{}).Count(&count)
	if count != 1 {
		t.Fatalf("expected fresh row to survive, count=%d", count)
	}
}
