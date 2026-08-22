package household

import (
	"time"
)

// Role defines the permission level of a household member.
type Role string

const (
	RoleOwner  Role = "owner"
	RoleMember Role = "member"
)

// Household represents a shared household group.
type Household struct {
	ID        uint              `gorm:"primaryKey" json:"id"`
	Name      string            `gorm:"size:100;not null" json:"name"`
	OwnerID   uint              `gorm:"not null" json:"owner_id"`
	Members   []HouseholdMember `gorm:"foreignKey:HouseholdID" json:"members,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// TableName overrides the default table name for GORM.
func (Household) TableName() string {
	return "households"
}

// HouseholdMember links a user to a household with a role.
type HouseholdMember struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	HouseholdID uint      `gorm:"not null;index" json:"household_id"`
	UserID      uint      `gorm:"not null;index" json:"user_id"`
	Role        Role      `gorm:"size:20;not null;default:member" json:"role"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TableName overrides the default table name for GORM.
func (HouseholdMember) TableName() string {
	return "household_members"
}
