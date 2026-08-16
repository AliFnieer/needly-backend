package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// corsMiddleware handles Cross-Origin Resource Sharing (CORS).
func (s *Server) corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		// Allow if origin is in the allowed list, or allow all if list is empty
		allowed := len(s.cfg.CORS.AllowedOrigins) == 0
		for _, allowedOrigin := range s.cfg.CORS.AllowedOrigins {
			if allowedOrigin == "*" || allowedOrigin == origin {
				allowed = true
				break
			}
		}

		if allowed {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
			c.Header("Access-Control-Max-Age", "86400")
		}

		// Handle preflight requests
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}