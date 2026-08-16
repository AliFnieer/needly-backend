package auth

import (
	"time"
)

// User represents a registered user in the system.
type User struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	FirstName    string         `gorm:"size:100;not null" json:"first_name"`
	LastName     string         `gorm:"size:100;not null" json:"last_name"`
	Email        string         `gorm:"size:255;uniqueIndex;not null" json:"email"`
	PasswordHash string         `gorm:"size:255;not null" json:"-"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

// TableName overrides the default table name for GORM.
func (User) TableName() string {
	return "users"
}