package middleware

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

// LoggerMiddleware logs request details: method, path, status, latency.
func LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		path := c.Request.URL.Path
		if raw := c.Request.URL.RawQuery; raw != "" {
			path = path + "?" + raw
		}

		// Process request
		c.Next()

		// Log after request is processed
		latency := time.Since(start)
		status := c.Writer.Status()
		method := c.Request.Method
		clientIP := c.ClientIP()

		log.Printf("[%s] %s %s -> %d (%s)",
			clientIP,
			method,
			path,
			status,
			latency,
		)
	}
}