package middleware

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	maxAuditBodySize = 1024
)

var skippedPaths = map[string]bool{
	"/health":  true,
	"/metrics": true,
	"/docs":    true,
}

var sensitiveHeaders = map[string]bool{
	"authorization":  true,
	"cookie":         true,
	"set-cookie":     true,
	"proxy-authorization": true,
}

type responseCapture struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (rc *responseCapture) Write(b []byte) (int, error) {
	rc.body.Write(b)
	return rc.ResponseWriter.Write(b)
}

func (rc *responseCapture) WriteString(s string) (int, error) {
	rc.body.WriteString(s)
	return rc.ResponseWriter.WriteString(s)
}

func isDebugLevel() bool {
	return strings.EqualFold(os.Getenv("LOG_LEVEL"), "debug")
}

func isJSONContentType(ct string) bool {
	return strings.Contains(ct, "application/json")
}

func truncateBody(b []byte) string {
	if len(b) > maxAuditBodySize {
		return string(b[:maxAuditBodySize]) + "...[truncated]"
	}
	return string(b)
}

func redactHeaders(headers http.Header) map[string]string {
	out := make(map[string]string, len(headers))
	for k, v := range headers {
		if sensitiveHeaders[strings.ToLower(k)] {
			out[k] = "[REDACTED]"
		} else {
			out[k] = strings.Join(v, ", ")
		}
	}
	return out
}

func AuditLogMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !isDebugLevel() {
			c.Next()
			return
		}

		path := c.Request.URL.Path
		if skippedPaths[path] {
			c.Next()
			return
		}

		start := time.Now()

		var reqBody []byte
		ct := c.GetHeader("Content-Type")
		if c.Request.Body != nil && isJSONContentType(ct) {
			// Read the FULL body so downstream handlers see the complete payload.
			reqBody, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewReader(reqBody))
		}

		capture := &responseCapture{
			ResponseWriter: c.Writer,
			body:           &bytes.Buffer{},
		}
		c.Writer = capture

		c.Next()

		duration := time.Since(start)
		status := c.Writer.Status()
		respCT := c.Writer.Header().Get("Content-Type")
		respBody := capture.body.Bytes()

		attrs := []any{
			"method", c.Request.Method,
			"path", path,
			"status", status,
			"duration_ms", duration.Milliseconds(),
		}

		if len(reqBody) > 0 && isJSONContentType(ct) {
			attrs = append(attrs, "request_body", truncateBody(reqBody))
		}
		if len(respBody) > 0 && isJSONContentType(respCT) {
			attrs = append(attrs, "response_body", truncateBody(respBody))
		}

		if uid, exists := c.Get("user_id"); exists {
			attrs = append(attrs, "user_id", uid)
		}

		if rid := c.GetHeader(RequestIDHeader); rid != "" {
			attrs = append(attrs, "request_id", rid)
		}

		redacted := redactHeaders(c.Request.Header)
		attrs = append(attrs, "headers", redacted)

		slog.Debug("audit", attrs...)
	}
}
