package shoppinglist

import (
	"time"

	"github.com/AliFnieer/needly-backend/internal/shoppingitem"
)

// ShoppingList represents a shopping list belonging to a household.
type ShoppingList struct {
	ID          uint                        `gorm:"primaryKey" json:"id"`
	HouseholdID uint                        `gorm:"not null;index" json:"household_id"`
	Name        string                      `gorm:"size:150;not null" json:"name"`
	CreatedBy   uint                        `gorm:"not null" json:"created_by"`
	Items       []shoppingitem.ShoppingItem `gorm:"foreignKey:ListID" json:"items,omitempty"`
	CreatedAt   time.Time                   `json:"created_at"`
	UpdatedAt   time.Time                   `json:"updated_at"`
}

// TableName overrides the default table name for GORM.
func (ShoppingList) TableName() string {
	return "shopping_lists"
}
