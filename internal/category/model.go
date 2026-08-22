package category

import (
	"time"
)

// Category represents a category scoped to a household.
type Category struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	HouseholdID uint      `gorm:"not null;index;uniqueIndex:idx_category_household_name" json:"household_id"`
	Name        string    `gorm:"size:100;not null;uniqueIndex:idx_category_household_name" json:"name"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TableName overrides the default table name for GORM.
func (Category) TableName() string {
	return "categories"
}
