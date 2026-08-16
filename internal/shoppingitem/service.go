package shoppingitem

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
)

// Service handles shopping item business logic.
type Service struct {
	db *gorm.DB
}

// CreateRequest is the payload for creating a shopping item.
type CreateRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=200"`
	Quantity    int    `json:"quantity" binding:"omitempty,min=1,max=10000"`
	Unit        string `json:"unit" binding:"omitempty,max=50"`
	IsCompleted bool   `json:"is_completed"`
}

// UpdateRequest is the payload for updating a shopping item.
type UpdateRequest struct {
	Name        string `json:"name" binding:"omitempty,min=1,max=200"`
	Quantity    *int   `json:"quantity" binding:"omitempty,min=1,max=10000"`
	Unit        string `json:"unit" binding:"omitempty,max=50"`
	IsCompleted *bool  `json:"is_completed"`
}

// NewService creates a new shopping item service.
func NewService(db *gorm.DB) *Service {
	return &Service{
		db: db,
	}
}

// Create adds a new item to a shopping list.
func (s *Service) Create(listID, userID uint, req *CreateRequest) (*ShoppingItem, error) {
	quantity := req.Quantity
	if quantity == 0 {
		quantity = 1
	}

	item := ShoppingItem{
		ListID:      listID,
		Name:        req.Name,
		Quantity:    quantity,
		Unit:        req.Unit,
		IsCompleted: req.IsCompleted,
		CreatedBy:   userID,
	}

	if err := s.db.Create(&item).Error; err != nil {
		return nil, fmt.Errorf("failed to create shopping item: %w", err)
	}

	return &item, nil
}

// GetByID retrieves a shopping item by ID.
func (s *Service) GetByID(id uint) (*ShoppingItem, error) {
	var item ShoppingItem
	if err := s.db.First(&item, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("shopping item not found")
		}
		return nil, fmt.Errorf("failed to get shopping item: %w", err)
	}
	return &item, nil
}

// ListByListID retrieves all items in a shopping list.
func (s *Service) ListByListID(listID uint) ([]ShoppingItem, error) {
	var items []ShoppingItem
	if err := s.db.Where("list_id = ?", listID).Order("is_completed ASC, created_at ASC").Find(&items).Error; err != nil {
		return nil, fmt.Errorf("failed to list shopping items: %w", err)
	}
	return items, nil
}

// Update updates a shopping item.
func (s *Service) Update(id uint, req *UpdateRequest) (*ShoppingItem, error) {
	var item ShoppingItem
	if err := s.db.First(&item, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("shopping item not found")
		}
		return nil, fmt.Errorf("failed to get shopping item: %w", err)
	}

	if req.Name != "" {
		item.Name = req.Name
	}
	if req.Quantity != nil {
		item.Quantity = *req.Quantity
	}
	if req.Unit != "" {
		item.Unit = req.Unit
	}
	if req.IsCompleted != nil {
		item.IsCompleted = *req.IsCompleted
	}

	if err := s.db.Save(&item).Error; err != nil {
		return nil, fmt.Errorf("failed to update shopping item: %w", err)
	}

	return &item, nil
}

// UpdateCompleted updates just the completion status of a shopping item.
func (s *Service) UpdateCompleted(id uint, isCompleted bool) (*ShoppingItem, error) {
	var item ShoppingItem
	if err := s.db.First(&item, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("shopping item not found")
		}
		return nil, fmt.Errorf("failed to get shopping item: %w", err)
	}

	item.IsCompleted = isCompleted
	if err := s.db.Save(&item).Error; err != nil {
		return nil, fmt.Errorf("failed to update shopping item status: %w", err)
	}

	return &item, nil
}

// Delete removes a shopping item.
func (s *Service) Delete(id uint) error {
	result := s.db.Delete(&ShoppingItem{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete shopping item: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.New("shopping item not found")
	}
	return nil
}