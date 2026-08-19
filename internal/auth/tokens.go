package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/AliFnieer/needly-backend/internal/config"
	"github.com/golang-jwt/jwt/v5"
)

// TokenPair holds a short-lived access token and a long-lived refresh token.
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

// generateAccessToken creates a signed JWT access token for the given user.
func generateAccessToken(user *User, cfg *config.Config) (string, error) {
	jti, err := generateRandomHex(16)
	if err != nil {
		return "", fmt.Errorf("failed to generate jti: %w", err)
	}

	expiration := time.Duration(cfg.JWT.ExpirationHours) * time.Hour

	claims := jwt.MapClaims{
		"user_id": user.ID,
		"email":   user.Email,
		"iss":     cfg.JWT.Issuer,
		"exp":     time.Now().Add(expiration).Unix(),
		"iat":     time.Now().Unix(),
		"jti":     jti,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.JWT.Secret))
}

// generateRefreshToken produces a cryptographically random refresh token string.
func generateRefreshToken() (string, error) {
	return generateRandomHex(32)
}

// hashToken returns the hex-encoded SHA-256 hash of a token string.
func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// generateRandomHex produces n random bytes encoded as a hex string.
func generateRandomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("crypto/rand failed: %w", err)
	}
	return hex.EncodeToString(b), nil
}
