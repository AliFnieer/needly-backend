package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/AliFnieer/needly-backend/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func testAuthConfig() *config.Config {
	return &config.Config{
		JWT: config.JWTConfig{
			Secret:          "test-secret-key-for-unit-tests",
			ExpirationHours: 1,
			Issuer:          "needly-api",
		},
	}
}

func makeSignedToken(claims jwt.MapClaims, secret string) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := token.SignedString([]byte(secret))
	if err != nil {
		panic(err)
	}
	return s
}

func doRequest(router *gin.Engine, req *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func TestAuthMiddleware_MissingHeader(t *testing.T) {
	cfg := testAuthConfig()
	router := gin.New()
	router.Use(AuthMiddleware(cfg))
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := doRequest(router, httptest.NewRequest(http.MethodGet, "/protected", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "authorization header is required") {
		t.Fatalf("unexpected body: %s", w.Body.String())
	}
}

func TestAuthMiddleware_InvalidFormat(t *testing.T) {
	cfg := testAuthConfig()
	router := gin.New()
	router.Use(AuthMiddleware(cfg))
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "NotBearer sometoken")
	w := doRequest(router, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "invalid authorization header format") {
		t.Fatalf("unexpected body: %s", w.Body.String())
	}
}

func TestAuthMiddleware_ExpiredToken(t *testing.T) {
	cfg := testAuthConfig()
	claims := jwt.MapClaims{
		"user_id": float64(1),
		"email":   "test@example.com",
		"iss":     cfg.JWT.Issuer,
		"exp":     time.Now().Add(-1 * time.Hour).Unix(),
		"iat":     time.Now().Add(-2 * time.Hour).Unix(),
	}
	tokenStr := makeSignedToken(claims, cfg.JWT.Secret)

	router := gin.New()
	router.Use(AuthMiddleware(cfg))
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	w := doRequest(router, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAuthMiddleware_InvalidSignature(t *testing.T) {
	claims := jwt.MapClaims{
		"user_id": float64(1),
		"email":   "test@example.com",
		"iss":     "needly-api",
		"exp":     time.Now().Add(time.Hour).Unix(),
		"iat":     time.Now().Unix(),
	}
	tokenStr := makeSignedToken(claims, "wrong-secret-key")

	cfg := testAuthConfig()
	router := gin.New()
	router.Use(AuthMiddleware(cfg))
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	w := doRequest(router, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAuthMiddleware_WrongIssuer(t *testing.T) {
	cfg := testAuthConfig()
	claims := jwt.MapClaims{
		"user_id": float64(1),
		"email":   "test@example.com",
		"iss":     "wrong-issuer",
		"exp":     time.Now().Add(time.Hour).Unix(),
		"iat":     time.Now().Unix(),
	}
	tokenStr := makeSignedToken(claims, cfg.JWT.Secret)

	router := gin.New()
	router.Use(AuthMiddleware(cfg))
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	w := doRequest(router, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "invalid token issuer") {
		t.Fatalf("unexpected body: %s", w.Body.String())
	}
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	cfg := testAuthConfig()
	claims := jwt.MapClaims{
		"user_id": float64(42),
		"email":   "user@example.com",
		"iss":     cfg.JWT.Issuer,
		"exp":     time.Now().Add(time.Hour).Unix(),
		"iat":     time.Now().Unix(),
	}
	tokenStr := makeSignedToken(claims, cfg.JWT.Secret)

	router := gin.New()
	router.Use(AuthMiddleware(cfg))
	router.GET("/protected", func(c *gin.Context) {
		uid, _ := c.Get("user_id")
		email, _ := c.Get("user_email")
		c.JSON(http.StatusOK, gin.H{
			"user_id": uid,
			"email":   email,
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	w := doRequest(router, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "42") {
		t.Fatalf("expected user_id 42 in response, got: %s", body)
	}
	if !strings.Contains(body, "user@example.com") {
		t.Fatalf("expected email in response, got: %s", body)
	}
}

func TestAuthMiddleware_EmptyBearerToken(t *testing.T) {
	cfg := testAuthConfig()
	router := gin.New()
	router.Use(AuthMiddleware(cfg))
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer ")
	w := doRequest(router, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAuthMiddleware_BearerCaseInsensitive(t *testing.T) {
	cfg := testAuthConfig()
	claims := jwt.MapClaims{
		"user_id": float64(1),
		"email":   "test@example.com",
		"iss":     cfg.JWT.Issuer,
		"exp":     time.Now().Add(time.Hour).Unix(),
		"iat":     time.Now().Unix(),
	}
	tokenStr := makeSignedToken(claims, cfg.JWT.Secret)

	router := gin.New()
	router.Use(AuthMiddleware(cfg))
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "bearer "+tokenStr)
	w := doRequest(router, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for lowercase bearer, got %d", w.Code)
	}
}
