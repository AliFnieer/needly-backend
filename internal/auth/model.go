package auth

import (
	"time"

	"gorm.io/gorm"
)

// User represents a registered user in the system.
type User struct {
	ID              uint       `gorm:"primaryKey" json:"id"`
	FirstName       string     `gorm:"size:100;not null" json:"first_name"`
	LastName        string     `gorm:"size:100;not null" json:"last_name"`
	Email           string     `gorm:"size:255;uniqueIndex;not null" json:"email"`
	PasswordHash    string     `gorm:"size:255;not null" json:"-"`
	EmailVerifiedAt *time.Time `gorm:"column:email_verified_at" json:"-"`
	EmailVerified   bool       `gorm:"-" json:"email_verified"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// TableName overrides the default table name for GORM.
func (User) TableName() string {
	return "users"
}

// AfterFind hydrates computed fields after loading from the database.
func (u *User) AfterFind(tx *gorm.DB) error {
	u.EmailVerified = u.EmailVerifiedAt != nil
	return nil
}

// PasswordResetToken stores the hash of a single-use password reset token.
type PasswordResetToken struct {
	ID        uint       `gorm:"primaryKey" json:"-"`
	UserID    uint       `gorm:"not null;index" json:"-"`
	TokenHash string     `gorm:"size:255;not null;uniqueIndex" json:"-"`
	ExpiresAt time.Time  `gorm:"not null" json:"-"`
	UsedAt    *time.Time `json:"-"`
	CreatedAt time.Time  `json:"-"`
}

// TableName overrides the default table name for GORM.
func (PasswordResetToken) TableName() string {
	return "password_reset_tokens"
}

// EmailVerificationToken stores the hash of a single-use email verification token.
type EmailVerificationToken struct {
	ID        uint       `gorm:"primaryKey" json:"-"`
	UserID    uint       `gorm:"not null;index" json:"-"`
	TokenHash string     `gorm:"size:255;not null;uniqueIndex" json:"-"`
	ExpiresAt time.Time  `gorm:"not null" json:"-"`
	UsedAt    *time.Time `json:"-"`
	CreatedAt time.Time  `json:"-"`
}

// TableName overrides the default table name for GORM.
func (EmailVerificationToken) TableName() string {
	return "email_verification_tokens"
}

// RefreshToken represents a stored refresh token for session management.
type RefreshToken struct {
	ID        uint       `gorm:"primaryKey" json:"-"`
	UserID    uint       `gorm:"not null;index" json:"-"`
	TokenHash string     `gorm:"size:255;not null" json:"-"`
	FamilyID  string     `gorm:"size:64;not null;index" json:"-"`
	ExpiresAt time.Time  `gorm:"not null" json:"-"`
	RevokedAt *time.Time `json:"-"`
	CreatedAt time.Time  `json:"-"`
}

// TableName overrides the default table name for GORM.
func (RefreshToken) TableName() string {
	return "refresh_tokens"
}
