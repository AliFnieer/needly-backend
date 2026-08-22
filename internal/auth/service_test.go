package auth

import (
	"testing"
	"time"

	"github.com/AliFnieer/needly-backend/internal/config"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func testServiceConfig() *config.Config {
	return &config.Config{
		JWT: config.JWTConfig{
			Secret:               "test-secret-key-for-testing-only-32chars!",
			ExpirationHours:      1,
			RefreshTokenTTLHours: 720,
			Issuer:               "needly-api",
		},
	}
}

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger:                 logger.Default.LogMode(logger.Silent),
		SkipDefaultTransaction: true,
	})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	err = db.AutoMigrate(&User{}, &RefreshToken{})
	if err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	})
	return db
}

func newTestService(t *testing.T) *Service {
	t.Helper()
	db := setupTestDB(t)
	return NewService(db, testServiceConfig())
}

func testNow() time.Time {
	return time.Now()
}

func TestRegister_Success(t *testing.T) {
	svc := newTestService(t)

	resp, err := svc.Register(&RegisterRequest{
		FirstName: "Ali",
		LastName:  "Fnier",
		Email:     "ali@test.com",
		Password:  "securepass123",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.AccessToken == "" {
		t.Error("expected non-empty access token")
	}
	if resp.RefreshToken == "" {
		t.Error("expected non-empty refresh token")
	}
	if resp.TokenType != "Bearer" {
		t.Errorf("expected token type Bearer, got %s", resp.TokenType)
	}
	if resp.User.Email != "ali@test.com" {
		t.Errorf("expected email ali@test.com, got %s", resp.User.Email)
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	svc := newTestService(t)

	req := &RegisterRequest{
		FirstName: "Ali",
		LastName:  "Fnier",
		Email:     "ali@test.com",
		Password:  "securepass123",
	}

	if _, err := svc.Register(req); err != nil {
		t.Fatalf("first register failed: %v", err)
	}

	_, err := svc.Register(req)
	if err == nil {
		t.Fatal("expected duplicate email error")
	}
	if err.Error() != "email already registered" {
		t.Errorf("expected 'email already registered', got %v", err)
	}
}

func TestRegister_NormalizesEmail(t *testing.T) {
	svc := newTestService(t)

	resp, err := svc.Register(&RegisterRequest{
		FirstName: "Ali",
		LastName:  "Fnier",
		Email:     "  ALI@Test.com  ",
		Password:  "securepass123",
	})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}
	if resp.User.Email != "ali@test.com" {
		t.Fatalf("expected normalized email ali@test.com, got %q", resp.User.Email)
	}

	_, err = svc.Login(&LoginRequest{Email: "ALI@TEST.COM", Password: "securepass123"})
	if err != nil {
		t.Fatalf("login should succeed with normalized email: %v", err)
	}
}

func TestLogin_Success(t *testing.T) {
	svc := newTestService(t)

	_, err := svc.Register(&RegisterRequest{
		FirstName: "Ali",
		LastName:  "Fnier",
		Email:     "ali@test.com",
		Password:  "securepass123",
	})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	resp, err := svc.Login(&LoginRequest{
		Email:    "ali@test.com",
		Password: "securepass123",
	})
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	if resp.AccessToken == "" {
		t.Error("expected non-empty access token")
	}
	if resp.RefreshToken == "" {
		t.Error("expected non-empty refresh token")
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	svc := newTestService(t)

	_, _ = svc.Register(&RegisterRequest{
		FirstName: "Ali",
		LastName:  "Fnier",
		Email:     "ali@test.com",
		Password:  "securepass123",
	})

	_, err := svc.Login(&LoginRequest{
		Email:    "ali@test.com",
		Password: "wrongpassword",
	})
	if err == nil {
		t.Fatal("expected login error for wrong password")
	}
	if err.Error() != "invalid email or password" {
		t.Errorf("expected 'invalid email or password', got %v", err)
	}
}

func TestLogin_NonexistentUser(t *testing.T) {
	svc := newTestService(t)

	_, err := svc.Login(&LoginRequest{
		Email:    "nobody@test.com",
		Password: "whatever",
	})
	if err == nil {
		t.Fatal("expected login error for nonexistent user")
	}
	if err.Error() != "invalid email or password" {
		t.Errorf("expected 'invalid email or password', got %v", err)
	}
}

func TestRefresh_Success(t *testing.T) {
	svc := newTestService(t)

	reg, err := svc.Register(&RegisterRequest{
		FirstName: "Ali",
		LastName:  "Fnier",
		Email:     "ali@test.com",
		Password:  "securepass123",
	})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	resp, err := svc.Refresh(&RefreshRequest{
		RefreshToken: reg.RefreshToken,
	})
	if err != nil {
		t.Fatalf("refresh failed: %v", err)
	}
	if resp.AccessToken == "" {
		t.Error("expected new access token")
	}
	if resp.RefreshToken == "" {
		t.Error("expected new refresh token")
	}
	if resp.RefreshToken == reg.RefreshToken {
		t.Error("expected new refresh token (rotation), but got same token")
	}
}

func TestRefresh_InvalidToken(t *testing.T) {
	svc := newTestService(t)

	_, err := svc.Refresh(&RefreshRequest{
		RefreshToken: "nonexistent-token",
	})
	if err == nil {
		t.Fatal("expected error for invalid refresh token")
	}
	if err.Error() != "invalid refresh token" {
		t.Errorf("expected 'invalid refresh token', got %v", err)
	}
}

func TestRefresh_ReuseDetection(t *testing.T) {
	svc := newTestService(t)

	reg, err := svc.Register(&RegisterRequest{
		FirstName: "Ali",
		LastName:  "Fnier",
		Email:     "ali@test.com",
		Password:  "securepass123",
	})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	// First refresh — rotates the token
	_, err = svc.Refresh(&RefreshRequest{RefreshToken: reg.RefreshToken})
	if err != nil {
		t.Fatalf("first refresh failed: %v", err)
	}

	// Reuse the old token — should detect theft and revoke family
	_, err = svc.Refresh(&RefreshRequest{RefreshToken: reg.RefreshToken})
	if err == nil {
		t.Fatal("expected reuse detection error")
	}
}

func TestLogout_SpecificToken(t *testing.T) {
	svc := newTestService(t)

	reg, err := svc.Register(&RegisterRequest{
		FirstName: "Ali",
		LastName:  "Fnier",
		Email:     "ali@test.com",
		Password:  "securepass123",
	})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	err = svc.Logout(reg.User.ID, &LogoutRequest{
		RefreshToken: reg.RefreshToken,
	})
	if err != nil {
		t.Fatalf("logout failed: %v", err)
	}

	// Refresh with revoked token should fail
	_, err = svc.Refresh(&RefreshRequest{RefreshToken: reg.RefreshToken})
	if err == nil {
		t.Fatal("expected error after logout")
	}
}

func TestLogout_AllTokens(t *testing.T) {
	svc := newTestService(t)

	reg, err := svc.Register(&RegisterRequest{
		FirstName: "Ali",
		LastName:  "Fnier",
		Email:     "ali@test.com",
		Password:  "securepass123",
	})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	err = svc.Logout(reg.User.ID, nil)
	if err != nil {
		t.Fatalf("logout all failed: %v", err)
	}

	// All refresh tokens revoked — refresh should fail
	_, err = svc.Refresh(&RefreshRequest{RefreshToken: reg.RefreshToken})
	if err == nil {
		t.Fatal("expected error after logout all")
	}
}

func TestCleanupExpiredRefreshTokens(t *testing.T) {
	svc := newTestService(t)
	user := &User{FirstName: "Ali", LastName: "Fnier", Email: "ali@test.com", PasswordHash: "hash"}
	if err := svc.db.Create(user).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}

	expired := RefreshToken{UserID: user.ID, TokenHash: "expired-hash", FamilyID: "family-1", ExpiresAt: testNow().Add(-time.Hour)}
	active := RefreshToken{UserID: user.ID, TokenHash: "active-hash", FamilyID: "family-2", ExpiresAt: testNow().Add(time.Hour)}
	if err := svc.db.Create(&expired).Error; err != nil {
		t.Fatalf("create expired token failed: %v", err)
	}
	if err := svc.db.Create(&active).Error; err != nil {
		t.Fatalf("create active token failed: %v", err)
	}

	deleted, err := svc.CleanupExpiredRefreshTokens()
	if err != nil {
		t.Fatalf("cleanup failed: %v", err)
	}
	if deleted != 1 {
		t.Fatalf("expected 1 expired token deleted, got %d", deleted)
	}

	var remaining int64
	if err := svc.db.Model(&RefreshToken{}).Count(&remaining).Error; err != nil {
		t.Fatalf("count remaining failed: %v", err)
	}
	if remaining != 1 {
		t.Fatalf("expected 1 remaining token, got %d", remaining)
	}
}

func TestGetByID_Found(t *testing.T) {
	svc := newTestService(t)

	reg, err := svc.Register(&RegisterRequest{
		FirstName: "Ali",
		LastName:  "Fnier",
		Email:     "ali@test.com",
		Password:  "securepass123",
	})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	user, err := svc.GetByID(reg.User.ID)
	if err != nil {
		t.Fatalf("get by id failed: %v", err)
	}
	if user.Email != "ali@test.com" {
		t.Errorf("expected email ali@test.com, got %s", user.Email)
	}
}

func TestGetByID_NotFound(t *testing.T) {
	svc := newTestService(t)

	_, err := svc.GetByID(uint(99999))
	if err == nil {
		t.Fatal("expected error for nonexistent user")
	}
}
