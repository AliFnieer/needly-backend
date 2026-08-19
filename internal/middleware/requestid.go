package middleware

import (
	"crypto/rand"
	"fmt"

	"github.com/gin-gonic/gin"
)

const (
	// RequestIDHeader is the HTTP header used to propagate request IDs.
	RequestIDHeader = "X-Request-ID"

	// RequestIDKey is the Gin context key for the request ID.
	RequestIDKey = "request_id"
)

// RequestIDMiddleware ensures every request has a unique ID.
// It honours an incoming X-Request-ID header (for distributed tracing)
// and generates a UUIDv4 when none is provided.
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		rid := c.GetHeader(RequestIDHeader)
		if rid == "" {
			rid = generateUUIDv4()
		}

		c.Set(RequestIDKey, rid)
		c.Header(RequestIDHeader, rid)

		c.Next()
	}
}

// generateUUIDv4 produces a UUIDv4 string using crypto/rand.
func generateUUIDv4() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
