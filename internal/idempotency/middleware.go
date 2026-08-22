// Package idempotency provides middleware that makes POST endpoints safe to
// retry. Mobile clients operating offline queue mutations and replay them on
// reconnect; without idempotency keys a request that succeeded server-side
// but timed out client-side would be applied twice (e.g. duplicate items).
package idempotency

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// HeaderName is the HTTP header carrying the client-generated idempotency key.
const HeaderName = "Idempotency-Key"

// maxKeyLength bounds the raw header value before hashing.
const maxKeyLength = 255

// Middleware replays the stored response for a previously seen
// (user, route, Idempotency-Key) combination instead of re-executing the
// handler. It only acts on POST requests that carry the header; everything
// else passes through untouched. Responses with status >= 500 are never
// stored so transient failures stay retryable.
func Middleware(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method != http.MethodPost {
			c.Next()
			return
		}

		key := strings.TrimSpace(c.GetHeader(HeaderName))
		if key == "" || len(key) > maxKeyLength {
			c.Next()
			return
		}

		userID, ok := userIDFromContext(c)
		if !ok {
			c.Next()
			return
		}

		route := c.FullPath()
		if route == "" {
			route = c.Request.URL.Path
		}
		keyHash := hashKey(key)

		var stored IdempotencyKey
		err := db.Where("user_id = ? AND route = ? AND key_hash = ?",
			userID, route, keyHash).First(&stored).Error
		if err == nil {
			c.Header("Idempotency-Replayed", "true")
			c.Data(stored.StatusCode, stored.ContentType, []byte(stored.ResponseBody))
			c.Abort()
			return
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			slog.Error("idempotency lookup failed; executing handler anyway", "error", err)
			c.Next()
			return
		}

		blw := &captureWriter{ResponseWriter: c.Writer, status: http.StatusOK}
		c.Writer = blw
		c.Next()

		if blw.status >= 500 {
			return
		}

		rec := IdempotencyKey{
			UserID:       userID,
			Route:        route,
			KeyHash:      keyHash,
			StatusCode:   blw.status,
			ContentType:  blw.Header().Get("Content-Type"),
			ResponseBody: blw.body.String(),
		}
		if err := db.Create(&rec).Error; err != nil && !errors.Is(err, gorm.ErrDuplicatedKey) {
			slog.Warn("failed to store idempotency record", "error", err)
		}
	}
}

func userIDFromContext(c *gin.Context) (uint, bool) {
	v, exists := c.Get("user_id")
	if !exists {
		return 0, false
	}
	f, ok := v.(float64)
	if !ok || f <= 0 {
		return 0, false
	}
	return uint(f), true
}

func hashKey(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

type captureWriter struct {
	gin.ResponseWriter
	body   bytes.Buffer
	status int
}

func (w *captureWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *captureWriter) Write(b []byte) (int, error) {
	n, err := w.body.Write(b)
	written, werr := w.ResponseWriter.Write(b)
	if err == nil {
		err = werr
	}
	if n > written {
		n = written
	}
	return n, err
}

func (w *captureWriter) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}
