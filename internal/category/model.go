package category

import (
	"time"
)

// Category represents a category that can be assigned to shopping items.
type Category struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:100;not null;uniqueIndex" json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName overrides the default table name for GORM.
func (Category) TableName() string {
	return "categories"
}