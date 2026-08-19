package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	// DefaultRequestTimeout is the default maximum duration for a request.
	DefaultRequestTimeout = 30 * time.Second
)

// RequestTimeoutMiddleware adds a deadline to each request context.
// Long-lived connections (WebSocket upgrades) should be excluded.
func RequestTimeoutMiddleware(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip timeout for WebSocket upgrades — they are long-lived
		if isWebSocketUpgrade(c) {
			c.Next()
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)
		c.Next()

		// If context was cancelled by timeout before handler finished, ensure 504
		if ctx.Err() == context.DeadlineExceeded && !c.Writer.Written() {
			c.AbortWithStatusJSON(http.StatusGatewayTimeout, gin.H{
				"error": "request timed out",
			})
		}
	}
}

// isWebSocketUpgrade checks if the request is a WebSocket upgrade.
func isWebSocketUpgrade(c *gin.Context) bool {
	upgrade := c.GetHeader("Upgrade")
	return upgrade == "websocket" || upgrade == "WebSocket"
}
