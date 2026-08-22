package history

import (
	"time"

	"github.com/AliFnieer/needly-backend/internal/category"
)

// ShoppingHistory records a completed shopping item snapshot.
type ShoppingHistory struct {
	ID          uint               `gorm:"primaryKey" json:"id"`
	ListID      uint               `gorm:"not null;index" json:"list_id"`
	ItemID      *uint              `gorm:"index" json:"item_id"`
	Name        string             `gorm:"size:200;not null" json:"name"`
	Quantity    float64            `gorm:"not null;default:1" json:"quantity"`
	Unit        string             `gorm:"size:50" json:"unit"`
	CategoryID  *uint              `gorm:"index" json:"category_id"`
	Category    *category.Category `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
	CompletedBy uint               `gorm:"not null" json:"completed_by"`
	CompletedAt time.Time          `gorm:"not null;index" json:"completed_at"`
	CreatedAt   time.Time          `json:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"`
}

// TableName overrides the default table name for GORM.
func (ShoppingHistory) TableName() string {
	return "shopping_history"
}
