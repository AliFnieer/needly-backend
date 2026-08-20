package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
)

func setLogLevel(t *testing.T, level string) {
	t.Helper()
	t.Setenv("LOG_LEVEL", level)
}

func newAuditRouter(t *testing.T, handler gin.HandlerFunc) *gin.Engine {
	t.Helper()
	router := gin.New()
	router.Use(handler)
	return router
}

func doAuditRequest(router *gin.Engine, method, path string, body *http.Request) *httptest.ResponseRecorder {
	if body == nil {
		body = httptest.NewRequest(method, path, nil)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, body)
	return w
}

func ensureDebugLogLevel(t *testing.T) {
	t.Helper()
	os.Setenv("LOG_LEVEL", "debug")
	t.Cleanup(func() { os.Unsetenv("LOG_LEVEL") })
}
