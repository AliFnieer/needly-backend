package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// SecurityMiddleware sets standard security headers on every response.
func SecurityMiddleware(ginMode string) gin.HandlerFunc {
	isRelease := ginMode == gin.ReleaseMode

	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "0")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Permissions-Policy", "camera=(), microphone=(), geolocation=()")

		// HSTS only in production (not for localhost dev)
		if isRelease {
			c.Header("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		}

		// CSP allows unpkg.com for Swagger UI
		c.Header("Content-Security-Policy", strings.Join([]string{
			"default-src 'self'",
			"script-src 'self' 'unsafe-inline' https://unpkg.com",
			"style-src 'self' 'unsafe-inline' https://unpkg.com",
			"img-src 'self' data: https://unpkg.com",
		}, "; "))

		c.Next()
	}
}

// ClientIPHeader extracts the real client IP from common proxy headers,
// falling back to RemoteAddr.
func ClientIPHeader(c *gin.Context) string {
	if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	if xri := c.GetHeader("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	return c.ClientIP()
}
