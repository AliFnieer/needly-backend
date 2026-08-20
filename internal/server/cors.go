package server

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

// corsMiddleware handles Cross-Origin Resource Sharing (CORS).
// It enforces a strict origin whitelist: requests from unknown origins are
// rejected. When the allowed list is empty, all cross-origin requests are
// rejected (fail-closed). The wildcard "*" is only permitted in debug/test
// mode; in release mode it is ignored.
func (s *Server) corsMiddleware() gin.HandlerFunc {
	origins := s.cfg.CORS.AllowedOrigins
	allowAll := false

	if len(origins) == 0 {
		// No origins configured — reject all cross-origin requests.
		slog.Warn("CORS: no allowed origins configured, rejecting all cross-origin requests")
	} else if len(origins) == 1 && origins[0] == "*" {
		if s.cfg.Server.GinMode == gin.ReleaseMode {
			// Wildcard blocked in production — reject everything.
			slog.Warn("CORS: wildcard origin is not allowed in release mode, rejecting all cross-origin requests")
		} else {
			allowAll = true
		}
	}

	return func(c *gin.Context) {
		// Always handle preflight first, even if origin is rejected.
		if c.Request.Method == http.MethodOptions {
			if !allowAll && len(origins) == 0 {
				c.AbortWithStatus(http.StatusNoContent)
				return
			}
		}

		origin := c.GetHeader("Origin")

		// Non-browser requests without Origin are allowed through.
		if origin == "" {
			c.Next()
			return
		}

		allowed := false
		if allowAll {
			allowed = true
		} else {
			for _, allowedOrigin := range origins {
				if allowedOrigin == origin {
					allowed = true
					break
				}
			}
		}

		if !allowed {
			// Silently reject — do not set any CORS headers so the
			// browser blocks the response.
			if c.Request.Method == http.MethodOptions {
				c.AbortWithStatus(http.StatusNoContent)
				return
			}
			c.Next()
			return
		}

		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
