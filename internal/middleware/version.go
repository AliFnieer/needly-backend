package middleware

import (
	"github.com/gin-gonic/gin"
)

const (
	// APIVersion is the current API version.
	APIVersion = "v1"
	// APIVersionHeader is the response header indicating the API version.
	APIVersionHeader = "X-API-Version"
)

// VersionMiddleware sets the X-API-Version response header on every request.
func VersionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header(APIVersionHeader, APIVersion)
		c.Next()
	}
}
