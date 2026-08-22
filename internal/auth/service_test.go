package auth

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/AliFnieer/needly-backend/internal/config"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// fakeMailer captures outgoing emails for assertions.
type fakeMailer struct {
	sent []sentEmail
	err  error
}

type sentEmail struct {
	to      string
	subject string
	body    string
}

func (f *fakeMailer) Send(to, subject, body string) error {
	if f.err != nil {
		return f.err
	}
	f.sent = append(f.sent, sentEmail{to: to, subject: subject, body: body})
	return nil
}

func (f *fakeMailer) last() sentEmail { return f.sent[len(f.sent)-1] }

func testServiceConfig() *config.Config {
	return &config.Config{
		JWT: config.JWTConfig{
			Secret:               "test-secret-key-for-testing-only-32chars!",
			ExpirationHours:      1,
			RefreshTokenTTLHours: 720,
			Issuer:               "needly-api",
		},
		App: config.AppConfig{
			BaseURL: "http://localhost:3000",
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
	err = db.AutoMigrate(&User{}, &RefreshToken{}, &PasswordResetToken{}, &EmailVerificationToken{})
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

func newTestService(t *testing.T) (*Service, *fakeMailer) {
	t.Helper()
	db := setupTestDB(t)
	m := &fakeMailer{}
	return NewService(db, testServiceConfig(), m), m
}

func testNow() time.Time {
	return time.Now()
}

func TestRegister_Success(t *testing.T) {
	svc, _ := newTestService(t)

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
	svc, _ := newTestService(t)

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
	svc, _ := newTestService(t)

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
	svc, _ := newTestService(t)

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
	svc, _ := newTestService(t)

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
	svc, _ := newTestService(t)

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
	svc, _ := newTestService(t)

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
	svc, _ := newTestService(t)

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
	svc, _ := newTestService(t)

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
	svc, _ := newTestService(t)

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
	svc, _ := newTestService(t)

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
	svc, _ := newTestService(t)
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
	svc, _ := newTestService(t)

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
	svc, _ := newTestService(t)

	_, err := svc.GetByID(uint(99999))
	if err == nil {
		t.Fatal("expected error for nonexistent user")
	}
}

func tokenFromLink(body string) string {
	i := strings.Index(body, "token=")
	if i < 0 {
		return ""
	}
	return strings.TrimSpace(body[i+len("token="):])
}

func TestRegister_SendsVerificationEmail(t *testing.T) {
	svc, m := newTestService(t)

	resp, err := svc.Register(&RegisterRequest{
		FirstName: "Ali", LastName: "Fnier", Email: "verify@test.com", Password: "password123",
	})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}
	if len(m.sent) != 1 {
		t.Fatalf("expected 1 email after register, got %d", len(m.sent))
	}
	sent := m.last()
	if sent.to != "verify@test.com" {
		t.Errorf("expected email to verify@test.com, got %s", sent.to)
	}
	if token := tokenFromLink(sent.body); token == "" {
		t.Error("expected verification link with token in body")
	}
	if resp.User.EmailVerified {
		t.Error("new user should not be verified")
	}
}

func TestRequestPasswordReset_UnknownEmail_NoEnumeration(t *testing.T) {
	svc, m := newTestService(t)

	err := svc.RequestPasswordReset(&ForgotPasswordRequest{Email: "ghost@test.com"})
	if err != nil {
		t.Fatalf("expected nil error for unknown email, got %v", err)
	}
	if len(m.sent) != 0 {
		t.Errorf("expected no emails for unknown address, got %d", len(m.sent))
	}
}

func TestRequestPasswordReset_SendsLinkAndInvalidatesOldTokens(t *testing.T) {
	svc, m := newTestService(t)

	user := seedTestUser(t, svc, "reset@test.com")

	if _, err := svc.createPasswordResetToken(user.ID); err != nil {
		t.Fatalf("failed to seed old token: %v", err)
	}

	if err := svc.RequestPasswordReset(&ForgotPasswordRequest{Email: "RESET@Test.com"}); err != nil {
		t.Fatalf("request failed: %v", err)
	}

	var count int64
	svc.db.Model(&PasswordResetToken{}).Where("user_id = ?", user.ID).Count(&count)
	if count != 1 {
		t.Errorf("expected exactly 1 active reset token, got %d", count)
	}

	if len(m.sent) != 1 {
		t.Fatalf("expected 1 reset email, got %d", len(m.sent))
	}
	if token := tokenFromLink(m.last().body); token == "" {
		t.Error("expected reset link with token in body")
	}
}

func TestRequestPasswordReset_MailerFailure_StillSucceeds(t *testing.T) {
	svc, m := newTestService(t)
	m.err = errors.New("smtp down")

	user := seedTestUser(t, svc, "mailerfail@test.com")
	err := svc.RequestPasswordReset(&ForgotPasswordRequest{Email: user.Email})
	if err != nil {
		t.Fatalf("request should succeed even if mail delivery fails, got %v", err)
	}
}

func TestResetPassword_Success_RevokesAllSessions(t *testing.T) {
	svc, _ := newTestService(t)

	email := "fullflow@test.com"
	user := seedTestUser(t, svc, email)

	// Create a second session (login) so two families are active.
	firstLogin, err := svc.Login(&LoginRequest{Email: email, Password: "password123"})
	if err != nil {
		t.Fatalf("first login failed: %v", err)
	}
	if _, err := svc.Login(&LoginRequest{Email: email, Password: "password123"}); err != nil {
		t.Fatalf("second login failed: %v", err)
	}

	rawToken, err := svc.createPasswordResetToken(user.ID)
	if err != nil {
		t.Fatalf("failed to create reset token: %v", err)
	}

	if err := svc.ResetPassword(&ResetPasswordRequest{Token: rawToken, NewPassword: "newPassword456"}); err != nil {
		t.Fatalf("reset failed: %v", err)
	}

	// Old refresh tokens must be revoked.
	if _, err := svc.Refresh(&RefreshRequest{RefreshToken: firstLogin.RefreshToken}); err == nil {
		t.Error("expected refresh with pre-reset token to fail after password reset")
	}

	// New password works, old one does not.
	if _, err := svc.Login(&LoginRequest{Email: email, Password: "newPassword456"}); err != nil {
		t.Errorf("login with new password failed: %v", err)
	}
	if _, err := svc.Login(&LoginRequest{Email: email, Password: "password123"}); err == nil {
		t.Error("login with old password unexpectedly succeeded")
	}

	// Token cannot be reused.
	if err := svc.ResetPassword(&ResetPasswordRequest{Token: rawToken, NewPassword: "anotherPass789"}); err == nil {
		t.Error("expected reused reset token to be rejected")
	}
}

func TestResetPassword_ExpiredTokenRejected(t *testing.T) {
	svc, _ := newTestService(t)

	user := seedTestUser(t, svc, "expired@test.com")
	rawToken, err := svc.createPasswordResetToken(user.ID)
	if err != nil {
		t.Fatalf("failed to create token: %v", err)
	}
	if err := svc.db.Model(&PasswordResetToken{}).
		Where("token_hash = ?", hashToken(rawToken)).
		Update("expires_at", time.Now().Add(-time.Minute)).Error; err != nil {
		t.Fatalf("failed to expire token: %v", err)
	}

	if err := svc.ResetPassword(&ResetPasswordRequest{Token: rawToken, NewPassword: "newPassword456"}); err == nil {
		t.Error("expected expired token to be rejected")
	}
}

func TestVerifyEmail_Success(t *testing.T) {
	svc, m := newTestService(t)

	user := seedTestUser(t, svc, "verifyme@test.com")
	if _, err := svc.Register(&RegisterRequest{
		FirstName: "V", LastName: "U", Email: user.Email, Password: "password123",
	}); err == nil {
		t.Log("re-register of existing user is a conflict — using seeded user instead")
	}

	// Seed a fresh verification token via the service helper.
	rawToken, err := svc.createEmailVerificationToken(user.ID)
	if err != nil {
		t.Fatalf("failed to create token: %v", err)
	}
	_ = m

	if err := svc.VerifyEmail(rawToken); err != nil {
		t.Fatalf("verify failed: %v", err)
	}

	fresh, err := svc.GetByID(user.ID)
	if err != nil {
		t.Fatalf("get by id failed: %v", err)
	}
	if !fresh.EmailVerified {
		t.Error("expected user to be verified after consuming token")
	}

	if err := svc.VerifyEmail(rawToken); err == nil {
		t.Error("expected used verification token to be rejected")
	}
}

func TestVerifyEmail_BadAndEmptyTokens(t *testing.T) {
	svc, _ := newTestService(t)

	if err := svc.VerifyEmail(""); err == nil {
		t.Error("expected empty token to be rejected")
	}
	if err := svc.VerifyEmail("not-a-real-token"); err == nil {
		t.Error("expected unknown token to be rejected")
	}
}

func TestResendVerification_AlreadyVerifiedRejected(t *testing.T) {
	svc, _ := newTestService(t)

	user := seedTestUser(t, svc, "resend@test.com")
	now := time.Now()
	if err := svc.db.Model(&User{}).Where("id = ?", user.ID).
		Update("email_verified_at", now).Error; err != nil {
		t.Fatalf("failed to mark verified: %v", err)
	}

	if err := svc.ResendVerificationEmail(user.ID); err == nil {
		t.Error("expected resend for already-verified user to fail")
	}
}

func seedTestUser(t *testing.T, svc *Service, email string) *User {
	t.Helper()
	hashed, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	u := User{FirstName: "Test", LastName: "User", Email: strings.ToLower(email), PasswordHash: string(hashed)}
	if err := svc.db.Create(&u).Error; err != nil {
		t.Fatalf("failed to seed user: %v", err)
	}
	return &u
}
