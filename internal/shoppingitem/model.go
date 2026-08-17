package shoppingitem

import (
	"time"

	"github.com/AliFnieer/needly-backend/internal/category"
)

// ShoppingItem represents an individual item within a shopping list.
type ShoppingItem struct {
	ID          uint              `gorm:"primaryKey" json:"id"`
	ListID      uint              `gorm:"not null;index" json:"list_id"`
	CategoryID  *uint             `gorm:"index" json:"category_id"`
	Name        string            `gorm:"size:200;not null" json:"name"`
	Quantity    float64           `gorm:"not null;default:1" json:"quantity"`
	Unit        string            `gorm:"size:50" json:"unit"`
	IsCompleted bool              `gorm:"not null;default:false" json:"is_completed"`
	CreatedBy   uint              `gorm:"not null" json:"created_by"`
	Category    *category.Category `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// TableName overrides the default table name for GORM.
func (ShoppingItem) TableName() string {
	return "shopping_items"
}