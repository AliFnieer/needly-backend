package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	// CurrentAPIVersion is the current supported API version.
	CurrentAPIVersion = "2025-07-01"

	// SupportedVersions lists all API versions this server handles.
	// The oldest version is first; the newest is last.
	SupportedVersions = "2025-07-01"
)

// APIVersionMiddleware validates the API-Version header. If the header is
// missing the request proceeds (backwards compatible). If the header is
// present but not in the supported set, the request is rejected with 400.
func APIVersionMiddleware() gin.HandlerFunc {
	supported := map[string]bool{
		CurrentAPIVersion: true,
	}

	return func(c *gin.Context) {
		version := strings.TrimSpace(c.GetHeader("API-Version"))
		if version == "" {
			// No header — default to current version (backwards compatible).
			c.Set("api_version", CurrentAPIVersion)
			c.Header("API-Version", CurrentAPIVersion)
			c.Next()
			return
		}

		if !supported[version] {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":     "unsupported API version",
				"supported": SupportedVersions,
				"current":   CurrentAPIVersion,
			})
			c.Abort()
			return
		}

		c.Set("api_version", version)
		c.Header("API-Version", version)
		c.Next()
	}
}
