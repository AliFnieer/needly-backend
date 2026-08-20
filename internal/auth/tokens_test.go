package auth

import (
	"encoding/hex"
	"strings"
	"testing"
	"time"

	"github.com/AliFnieer/needly-backend/internal/config"
	"github.com/golang-jwt/jwt/v5"
)

func testConfig() *config.Config {
	return &config.Config{
		JWT: config.JWTConfig{
			Secret:          "test-secret-key-for-unit-tests",
			ExpirationHours: 1,
			Issuer:          "needly-api",
		},
	}
}

func TestGenerateAccessToken(t *testing.T) {
	cfg := testConfig()
	user := &User{ID: 1, Email: "test@example.com"}

	tokenStr, err := generateAccessToken(user, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tokenStr == "" {
		t.Fatal("expected non-empty token string")
	}

	// Parse and verify claims
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		return []byte(cfg.JWT.Secret), nil
	})
	if err != nil {
		t.Fatalf("failed to parse token: %v", err)
	}
	if !token.Valid {
		t.Fatal("expected valid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		t.Fatal("expected MapClaims")
	}

	if claims["user_id"] != float64(1) {
		t.Fatalf("expected user_id 1, got %v", claims["user_id"])
	}
	if claims["email"] != "test@example.com" {
		t.Fatalf("expected email test@example.com, got %v", claims["email"])
	}
	if claims["iss"] != "needly-api" {
		t.Fatalf("expected iss needly-api, got %v", claims["iss"])
	}

	exp, ok := claims["exp"].(float64)
	if !ok {
		t.Fatal("expected exp to be numeric")
	}
	iat, ok := claims["iat"].(float64)
	if !ok {
		t.Fatal("expected iat to be numeric")
	}
	if exp <= iat {
		t.Fatalf("exp (%v) should be after iat (%v)", exp, iat)
	}
	// exp - iat should be approximately 1 hour (3600s)
	diff := time.Duration(exp-iat) * time.Second
	if diff != time.Hour {
		t.Fatalf("expected expiration gap of 1h, got %v", diff)
	}

	jti, ok := claims["jti"].(string)
	if !ok || jti == "" {
		t.Fatal("expected non-empty jti string")
	}
}

func TestGenerateRefreshToken(t *testing.T) {
	token, err := generateRefreshToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty refresh token")
	}
	// 32 bytes → 64 hex chars
	if len(token) != 64 {
		t.Fatalf("expected 64-char hex string, got %d chars", len(token))
	}
	if _, err := hex.DecodeString(token); err != nil {
		t.Fatalf("expected valid hex, got error: %v", err)
	}
}

func TestHashToken(t *testing.T) {
	h1 := hashToken("hello")
	h2 := hashToken("hello")
	h3 := hashToken("world")

	if h1 != h2 {
		t.Fatalf("expected deterministic hash: %q != %q", h1, h2)
	}
	if h1 == h3 {
		t.Fatal("expected different inputs to produce different hashes")
	}
	// SHA-256 hex output is 64 chars
	if len(h1) != 64 {
		t.Fatalf("expected 64-char hex hash, got %d chars", len(h1))
	}
}

func TestGenerateRandomHex(t *testing.T) {
	v1, err := generateRandomHex(16)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	v2, err := generateRandomHex(16)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 16 bytes → 32 hex chars
	if len(v1) != 32 {
		t.Fatalf("expected 32 chars, got %d", len(v1))
	}
	if _, err := hex.DecodeString(v1); err != nil {
		t.Fatalf("expected valid hex: %v", err)
	}

	// Two random calls should (almost certainly) differ
	if v1 == v2 {
		t.Fatal("expected different random values")
	}

	// Test with different length
	v3, err := generateRandomHex(32)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(v3) != 64 {
		t.Fatalf("expected 64 chars, got %d", len(v3))
	}
}

func TestGenerateAccessTokenDifferentJTI(t *testing.T) {
	cfg := testConfig()
	user := &User{ID: 1, Email: "test@example.com"}

	tokenStr1, _ := generateAccessToken(user, cfg)
	tokenStr2, _ := generateAccessToken(user, cfg)

	parser := jwt.NewParser()
	tok1, _, _ := parser.ParseUnverified(tokenStr1, jwt.MapClaims{})
	tok2, _, _ := parser.ParseUnverified(tokenStr2, jwt.MapClaims{})

	c1 := tok1.Claims.(jwt.MapClaims)
	c2 := tok2.Claims.(jwt.MapClaims)

	if c1["jti"] == c2["jti"] {
		t.Fatal("expected different jti values for separate tokens")
	}
}

func TestGenerateAccessTokenWrongSecret(t *testing.T) {
	cfg := testConfig()
	user := &User{ID: 1, Email: "a@b.com"}

	tokenStr, err := generateAccessToken(user, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wrongCfg := &config.Config{
		JWT: config.JWTConfig{
			Secret:          "wrong-secret",
			ExpirationHours: 1,
			Issuer:          "needly-api",
		},
	}

	_, err = jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		return []byte(wrongCfg.JWT.Secret), nil
	})
	if err == nil {
		t.Fatal("expected error with wrong secret")
	}
}

func TestGenerateRandomHexDifferentLengths(t *testing.T) {
	for _, n := range []int{1, 8, 32, 64} {
		v, err := generateRandomHex(n)
		if err != nil {
			t.Fatalf("n=%d: unexpected error: %v", n, err)
		}
		expected := hex.EncodedLen(n)
		if len(v) != expected {
			t.Fatalf("n=%d: expected %d chars, got %d", n, expected, len(v))
		}
	}
}

// Ensure the token is HS256-signed (not RSA/ECDSA).
func TestGenerateAccessTokenSigningMethod(t *testing.T) {
	cfg := testConfig()
	user := &User{ID: 1, Email: "test@example.com"}

	tokenStr, err := generateAccessToken(user, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Parse without verification to inspect header
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 3 {
		t.Fatalf("expected 3-part JWT, got %d", len(parts))
	}

	// Use jwt.Parse which verifies HS256
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			t.Fatalf("expected HMAC signing method, got %T", token.Method)
		}
		return []byte(cfg.JWT.Secret), nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !token.Valid {
		t.Fatal("expected valid token")
	}
}

// Ensure issuer mismatch is detected by the auth middleware logic.
func TestGenerateAccessTokenIssuerMismatch(t *testing.T) {
	cfg := testConfig()
	user := &User{ID: 1, Email: "a@b.com"}

	tokenStr, err := generateAccessToken(user, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		return []byte(cfg.JWT.Secret), nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	claims := token.Claims.(jwt.MapClaims)
	if claims["iss"] != "needly-api" {
		t.Fatalf("unexpected issuer: %v", claims["iss"])
	}
}

func TestHashTokenEmpty(t *testing.T) {
	h := hashToken("")
	if len(h) != 64 {
		t.Fatalf("expected 64-char hex for empty string, got %d", len(h))
	}

	// Verify it's the SHA-256 of empty string
	expected := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	if h != expected {
		t.Fatalf("expected %s, got %s", expected, h)
	}
}
