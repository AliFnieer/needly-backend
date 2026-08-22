package auth

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/AliFnieer/needly-backend/internal/apperr"
	"github.com/AliFnieer/needly-backend/internal/config"
	"github.com/AliFnieer/needly-backend/internal/mailer"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const (
	// passwordResetTokenTTL is how long a password reset token remains valid.
	passwordResetTokenTTL = 15 * time.Minute

	// emailVerificationTokenTTL is how long an email verification token remains valid.
	emailVerificationTokenTTL = 24 * time.Hour
)

// Service handles auth business logic.
type Service struct {
	db     *gorm.DB
	cfg    *config.Config
	mailer mailer.Mailer
}

// RegisterRequest is the payload for user registration.
type RegisterRequest struct {
	FirstName string `json:"first_name" binding:"required,min=2,max=100"`
	LastName  string `json:"last_name" binding:"required,min=2,max=100"`
	Email     string `json:"email" binding:"required,email"`
	Password  string `json:"password" binding:"required,min=8,max=72"`
}

// LoginRequest is the payload for user login.
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// RefreshRequest is the payload for token refresh.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// LogoutRequest is the payload for logout.
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// ForgotPasswordRequest is the payload for requesting a password reset.
type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// ResetPasswordRequest is the payload for completing a password reset.
type ResetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8,max=72"`
}

// AuthResponse is returned on successful registration/login.
type AuthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	User         User   `json:"user"`
}

// NewService creates a new auth service. A nil mailer falls back to the
// development log mailer.
func NewService(db *gorm.DB, cfg *config.Config, m mailer.Mailer) *Service {
	if m == nil {
		m = mailer.LogMailer{}
	}
	return &Service{
		db:     db,
		cfg:    cfg,
		mailer: m,
	}
}

// Register creates a new user account and returns token pair.
func (s *Service) Register(req *RegisterRequest) (*AuthResponse, error) {
	normalizedEmail := normalizeEmail(req.Email)

	// Check if email is already registered
	var existing User
	if err := s.db.Where("email = ?", normalizedEmail).First(&existing).Error; err == nil {
		return nil, apperr.Conflict("email already registered")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperr.Wrap(err, http.StatusInternalServerError, apperr.CodeInternal, "failed to check existing user")
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create the user
	user := User{
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		Email:        normalizedEmail,
		PasswordHash: string(hashedPassword),
	}

	if err := s.db.Create(&user).Error; err != nil {
		// Map unique constraint violations (concurrent duplicate email) to a conflict.
		if isUniqueViolation(err) {
			return nil, apperr.Conflict("email already registered")
		}
		return nil, apperr.Wrap(err, http.StatusInternalServerError, apperr.CodeInternal, "failed to create user")
	}

	// Send the verification email on a best-effort basis — registration must
	// not fail when mail delivery is unavailable.
	s.sendVerificationEmail(&user)

	// Issue token pair
	return s.issueTokenPair(&user)
}

// Login authenticates a user and returns token pair.
func (s *Service) Login(req *LoginRequest) (*AuthResponse, error) {
	normalizedEmail := normalizeEmail(req.Email)
	var user User
	if err := s.db.Where("email = ?", normalizedEmail).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperr.Unauthorized("invalid email or password")
		}
		return nil, apperr.Wrap(err, http.StatusInternalServerError, apperr.CodeInternal, "failed to find user")
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, apperr.Unauthorized("invalid email or password")
	}

	// Issue token pair
	return s.issueTokenPair(&user)
}

// Refresh validates a refresh token and returns a new token pair.
// If the refresh token has been revoked (potential theft), all tokens in its family are revoked.
func (s *Service) Refresh(req *RefreshRequest) (*AuthResponse, error) {
	tokenHash := hashToken(req.RefreshToken)

	var stored RefreshToken
	if err := s.db.Where("token_hash = ?", tokenHash).First(&stored).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("invalid refresh token")
		}
		return nil, fmt.Errorf("failed to look up refresh token: %w", err)
	}

	// Token has been revoked — revoke entire family (reuse detection)
	if stored.RevokedAt != nil {
		s.revokeFamily(stored.FamilyID)
		return nil, errors.New("refresh token has been revoked; all sessions in this family terminated")
	}

	// Token expired
	if time.Now().After(stored.ExpiresAt) {
		s.revokeFamily(stored.FamilyID)
		return nil, errors.New("refresh token has expired")
	}

	// Revoke the current refresh token (rotation)
	now := time.Now()
	if err := s.db.Model(&RefreshToken{}).Where("id = ?", stored.ID).Update("revoked_at", now).Error; err != nil {
		return nil, fmt.Errorf("failed to revoke old refresh token: %w", err)
	}

	// Look up the user
	var user User
	if err := s.db.First(&user, stored.UserID).Error; err != nil {
		return nil, errors.New("user not found")
	}

	// Issue a new token pair in the same family
	return s.issueTokenPairInFamily(&user, stored.FamilyID)
}

// Logout revokes a specific refresh token, or all tokens for the user if none specified.
func (s *Service) Logout(userID uint, req *LogoutRequest) error {
	if req != nil && req.RefreshToken != "" {
		tokenHash := hashToken(req.RefreshToken)
		result := s.db.Model(&RefreshToken{}).Where("user_id = ? AND token_hash = ?", userID, tokenHash).
			Update("revoked_at", time.Now())
		if result.Error != nil {
			return fmt.Errorf("failed to revoke refresh token: %w", result.Error)
		}
		return nil
	}
	// Revoke all tokens for the user
	return s.revokeAllForUser(userID)
}

// CleanupExpiredRefreshTokens removes expired refresh tokens from storage.
func (s *Service) CleanupExpiredRefreshTokens() (int64, error) {
	result := s.db.Where("expires_at < ?", time.Now()).Delete(&RefreshToken{})
	if result.Error != nil {
		return 0, fmt.Errorf("failed to cleanup expired refresh tokens: %w", result.Error)
	}
	return result.RowsAffected, nil
}

// RequestPasswordReset issues a single-use password reset token for the
// given email and delivers it via the mailer. The response is intentionally
// identical whether or not the email is registered, to prevent enumeration.
func (s *Service) RequestPasswordReset(req *ForgotPasswordRequest) error {
	normalizedEmail := normalizeEmail(req.Email)

	var user User
	if err := s.db.Where("email = ?", normalizedEmail).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			slog.Debug("password reset requested for unknown email", "email", normalizedEmail)
			return nil
		}
		return apperr.Wrap(err, http.StatusInternalServerError, apperr.CodeInternal, "failed to find user")
	}

	return s.sendPasswordResetEmail(&user)
}

// ResetPassword consumes a password reset token, updates the user's password,
// and revokes all active sessions (refresh tokens) so other devices must log
// in again with the new credentials.
func (s *Service) ResetPassword(req *ResetPasswordRequest) error {
	tokenHash := hashToken(req.Token)

	var token PasswordResetToken
	if err := s.db.Where("token_hash = ?", tokenHash).First(&token).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperr.BadRequest("invalid or expired reset token")
		}
		return apperr.Wrap(err, http.StatusInternalServerError, apperr.CodeInternal, "failed to look up reset token")
	}

	if time.Now().After(token.ExpiresAt) || token.UsedAt != nil {
		return apperr.BadRequest("invalid or expired reset token")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Atomically consume the token: only one request wins the race.
	now := time.Now()
	result := s.db.Model(&PasswordResetToken{}).
		Where("id = ? AND used_at IS NULL", token.ID).
		Update("used_at", now)
	if result.Error != nil {
		return apperr.Wrap(result.Error, http.StatusInternalServerError, apperr.CodeInternal, "failed to consume reset token")
	}
	if result.RowsAffected == 0 {
		return apperr.BadRequest("invalid or expired reset token")
	}

	if err := s.db.Model(&User{}).Where("id = ?", token.UserID).
		Update("password_hash", string(hashedPassword)).Error; err != nil {
		return apperr.Wrap(err, http.StatusInternalServerError, apperr.CodeInternal, "failed to update password")
	}

	// Force re-authentication on every device.
	if err := s.revokeAllForUser(token.UserID); err != nil {
		return apperr.Wrap(err, http.StatusInternalServerError, apperr.CodeInternal, "failed to revoke sessions")
	}
	return nil
}

// VerifyEmail marks a user's email as verified using a verification token.
func (s *Service) VerifyEmail(token string) error {
	if strings.TrimSpace(token) == "" {
		return apperr.BadRequest("verification token is required")
	}
	tokenHash := hashToken(token)

	var stored EmailVerificationToken
	if err := s.db.Where("token_hash = ?", tokenHash).First(&stored).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperr.BadRequest("invalid or expired verification token")
		}
		return apperr.Wrap(err, http.StatusInternalServerError, apperr.CodeInternal, "failed to look up verification token")
	}

	if time.Now().After(stored.ExpiresAt) || stored.UsedAt != nil {
		return apperr.BadRequest("invalid or expired verification token")
	}

	now := time.Now()
	result := s.db.Model(&EmailVerificationToken{}).
		Where("id = ? AND used_at IS NULL", stored.ID).
		Update("used_at", now)
	if result.Error != nil {
		return apperr.Wrap(result.Error, http.StatusInternalServerError, apperr.CodeInternal, "failed to consume verification token")
	}
	if result.RowsAffected == 0 {
		return apperr.BadRequest("invalid or expired verification token")
	}

	if err := s.db.Model(&User{}).Where("id = ? AND email_verified_at IS NULL", stored.UserID).
		Update("email_verified_at", now).Error; err != nil {
		return apperr.Wrap(err, http.StatusInternalServerError, apperr.CodeInternal, "failed to verify email")
	}
	return nil
}

// ResendVerificationEmail issues a fresh verification token and emails it to
// the authenticated user.
func (s *Service) ResendVerificationEmail(userID uint) error {
	var user User
	if err := s.db.First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperr.NotFound("user not found")
		}
		return apperr.Wrap(err, http.StatusInternalServerError, apperr.CodeInternal, "failed to find user")
	}
	if user.EmailVerifiedAt != nil {
		return apperr.BadRequest("email already verified")
	}
	s.sendVerificationEmail(&user)
	return nil
}

// sendVerificationEmail stores a fresh single-use token and emails the link.
func (s *Service) sendVerificationEmail(user *User) {
	rawToken, err := s.createEmailVerificationToken(user.ID)
	if err != nil {
		slog.Error("failed to store verification token", "user_id", user.ID, "error", err)
		return
	}

	link := fmt.Sprintf("%s/verify-email?token=%s", strings.TrimRight(s.cfg.App.BaseURL, "/"), rawToken)
	body := strings.Join([]string{
		"Hi " + user.FirstName + ",",
		"",
		"Welcome to Needly! Please confirm your email address:",
		link,
		"",
		"This link expires in 24 hours.",
	}, "\n")

	if err := s.mailer.Send(user.Email, "Verify your Needly account", body); err != nil {
		slog.Error("failed to send verification email", "user_id", user.ID, "error", err)
	}
}

// createEmailVerificationToken persists a fresh single-use verification token
// and returns its raw form.
func (s *Service) createEmailVerificationToken(userID uint) (string, error) {
	raw, err := generateRandomHex(32)
	if err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}
	token := EmailVerificationToken{
		UserID:    userID,
		TokenHash: hashToken(raw),
		ExpiresAt: time.Now().Add(emailVerificationTokenTTL),
	}
	if err := s.db.Create(&token).Error; err != nil {
		return "", fmt.Errorf("failed to store verification token: %w", err)
	}
	return raw, nil
}

// sendPasswordResetEmail stores a fresh single-use token and emails the link.
func (s *Service) sendPasswordResetEmail(user *User) error {
	// A new reset request invalidates any previous unconsumed tokens.
	if err := s.db.Where("user_id = ? AND used_at IS NULL", user.ID).
		Delete(&PasswordResetToken{}).Error; err != nil {
		return apperr.Wrap(err, http.StatusInternalServerError, apperr.CodeInternal, "failed to clear old reset tokens")
	}

	rawToken, err := s.createPasswordResetToken(user.ID)
	if err != nil {
		return apperr.Wrap(err, http.StatusInternalServerError, apperr.CodeInternal, "failed to create reset token")
	}

	link := fmt.Sprintf("%s/reset-password?token=%s", strings.TrimRight(s.cfg.App.BaseURL, "/"), rawToken)
	body := strings.Join([]string{
		"Hi " + user.FirstName + ",",
		"",
		"We received a request to reset your Needly password:",
		link,
		"",
		"This link expires in 15 minutes and can be used once.",
		"If you did not request this, you can safely ignore this email.",
	}, "\n")

	if err := s.mailer.Send(user.Email, "Reset your Needly password", body); err != nil {
		slog.Error("failed to send password reset email", "user_id", user.ID, "error", err)
	}
	return nil
}

// createPasswordResetToken persists a fresh single-use reset token and
// returns its raw form.
func (s *Service) createPasswordResetToken(userID uint) (string, error) {
	raw, err := generateRandomHex(32)
	if err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}
	token := PasswordResetToken{
		UserID:    userID,
		TokenHash: hashToken(raw),
		ExpiresAt: time.Now().Add(passwordResetTokenTTL),
	}
	if err := s.db.Create(&token).Error; err != nil {
		return "", fmt.Errorf("failed to store reset token: %w", err)
	}
	return raw, nil
}

// isUniqueViolation reports whether the error is a PostgreSQL unique
// constraint violation (SQLSTATE 23505).
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}

// GetByID retrieves a user by their ID.
func (s *Service) GetByID(id interface{}) (*User, error) {
	var user User
	if err := s.db.First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &user, nil
}

// issueTokenPair creates a new access + refresh token pair with a new family.
func (s *Service) issueTokenPair(user *User) (*AuthResponse, error) {
	familyID, err := generateRandomHex(16)
	if err != nil {
		return nil, fmt.Errorf("failed to generate family id: %w", err)
	}
	return s.issueTokenPairInFamily(user, familyID)
}

// issueTokenPairInFamily creates a new access + refresh token pair within an existing family.
func (s *Service) issueTokenPairInFamily(user *User, familyID string) (*AuthResponse, error) {
	accessToken, err := generateAccessToken(user, s.cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := generateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	ttl := time.Duration(s.cfg.JWT.RefreshTokenTTLHours) * time.Hour
	stored := RefreshToken{
		UserID:    user.ID,
		TokenHash: hashToken(refreshToken),
		FamilyID:  familyID,
		ExpiresAt: time.Now().Add(ttl),
	}

	if err := s.db.Create(&stored).Error; err != nil {
		return nil, fmt.Errorf("failed to store refresh token: %w", err)
	}

	return &AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    s.cfg.JWT.ExpirationHours * 3600,
		User:         *user,
	}, nil
}

// revokeFamily revokes all refresh tokens in a given family.
func (s *Service) revokeFamily(familyID string) {
	s.db.Model(&RefreshToken{}).
		Where("family_id = ? AND revoked_at IS NULL", familyID).
		Update("revoked_at", time.Now())
}

// revokeAllForUser revokes all non-revoked refresh tokens for a user.
func (s *Service) revokeAllForUser(userID uint) error {
	return s.db.Model(&RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", time.Now()).Error
}

func normalizeEmail(email string) string {
	return strings.TrimSpace(strings.ToLower(email))
}
