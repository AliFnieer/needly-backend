package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/AliFnieer/needly-backend/internal/config"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// Service handles auth business logic.
type Service struct {
	db  *gorm.DB
	cfg *config.Config
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

// AuthResponse is returned on successful registration/login.
type AuthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	User         User   `json:"user"`
}

// NewService creates a new auth service.
func NewService(db *gorm.DB, cfg *config.Config) *Service {
	return &Service{
		db:  db,
		cfg: cfg,
	}
}

// Register creates a new user account and returns token pair.
func (s *Service) Register(req *RegisterRequest) (*AuthResponse, error) {
	// Check if email is already registered
	var existing User
	if err := s.db.Where("email = ?", req.Email).First(&existing).Error; err == nil {
		return nil, errors.New("email already registered")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
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
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
	}

	if err := s.db.Create(&user).Error; err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Issue token pair
	return s.issueTokenPair(&user)
}

// Login authenticates a user and returns token pair.
func (s *Service) Login(req *LoginRequest) (*AuthResponse, error) {
	var user User
	if err := s.db.Where("email = ?", req.Email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("invalid email or password")
		}
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, errors.New("invalid email or password")
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
