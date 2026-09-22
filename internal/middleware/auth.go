package middleware

import (
	"net/http"
	"strings"

	"github.com/AliFnieer/needly-backend/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const (
	// IssuerNeedlyAPI is the expected JWT issuer for this service.
	IssuerNeedlyAPI = "needly-api"
)

// AuthMiddlewareWS accepts the JWT as a `?token=` query parameter in addition
// to the Authorization header. WebSocket clients (mobile/browser) cannot set
// headers on the handshake, so when the header is absent the query token is
// copied into the header before delegating to AuthMiddleware. REST routes must
// keep using AuthMiddleware only.
func AuthMiddlewareWS(cfg *config.Config) gin.HandlerFunc {
	auth := AuthMiddleware(cfg)
	return func(c *gin.Context) {
		if c.GetHeader("Authorization") == "" {
			if token := c.Query("token"); token != "" {
				c.Request.Header.Set("Authorization", "Bearer "+token)
			}
		}
		auth(c)
	}
}

// AuthMiddleware validates the JWT token in the Authorization header.
func AuthMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "authorization header is required",
			})
			c.Abort()
			return
		}

		// Expected format: "Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid authorization header format",
			})
			c.Abort()
			return
		}

		tokenString := parts[1]

		// Parse and validate the token
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// Verify signing method — only accept HS256 to prevent algorithm confusion
			if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(cfg.JWT.Secret), nil
		})
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid or expired token",
			})
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid token claims",
			})
			c.Abort()
			return
		}

		// Validate issuer claim — must match configured issuer
		issuer, _ := claims["iss"].(string)
		if issuer != cfg.JWT.Issuer {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid token issuer",
			})
			c.Abort()
			return
		}

		// Store user ID and email in context for downstream handlers
		c.Set("user_id", claims["user_id"])
		c.Set("user_email", claims["email"])

		c.Next()
	}
}
