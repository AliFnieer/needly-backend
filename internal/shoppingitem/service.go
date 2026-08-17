package shoppingitem

import (
	"errors"
	"fmt"
	"strings"

	"github.com/AliFnieer/needly-backend/internal/category"
	"github.com/AliFnieer/needly-backend/internal/history"
	"gorm.io/gorm"
)

// Service handles shopping item business logic.
type Service struct {
	db      *gorm.DB
	history *history.Service
}

// CreateRequest is the payload for creating a shopping item.
type CreateRequest struct {
	Name        string  `json:"name" binding:"required,min=1,max=200"`
	Quantity    float64 `json:"quantity" binding:"omitempty,min=0.001,max=1000000"`
	Unit        string  `json:"unit" binding:"omitempty,min=1,max=50"`
	CategoryID  *uint   `json:"category_id" binding:"omitempty"`
	IsCompleted bool    `json:"is_completed"`
}

// UpdateRequest is the payload for updating a shopping item.
type UpdateRequest struct {
	Name        string   `json:"name" binding:"omitempty,min=1,max=200"`
	Quantity    *float64 `json:"quantity" binding:"omitempty,min=0.001,max=1000000"`
	Unit        string   `json:"unit" binding:"omitempty,min=1,max=50"`
	CategoryID  *uint    `json:"category_id" binding:"omitempty"`
	IsCompleted *bool    `json:"is_completed"`
}

// NewService creates a new shopping item service.
func NewService(db *gorm.DB, historySvc *history.Service) *Service {
	return &Service{
		db:      db,
		history: historySvc,
	}
}

// Create adds a new item to a shopping list.
func (s *Service) Create(listID, userID uint, req *CreateRequest) (*ShoppingItem, error) {
	quantity := req.Quantity
	if quantity <= 0 {
		quantity = 1
	}

	if err := s.validateCategoryID(req.CategoryID); err != nil {
		return nil, err
	}

	unit := strings.TrimSpace(req.Unit)

	var categoryID *uint
	if req.CategoryID != nil && *req.CategoryID != 0 {
		categoryID = req.CategoryID
	}

	item := ShoppingItem{
		ListID:      listID,
		CategoryID:  categoryID,
		Name:        req.Name,
		Quantity:    quantity,
		Unit:        unit,
		IsCompleted: req.IsCompleted,
		CreatedBy:   userID,
	}

	if err := s.db.Create(&item).Error; err != nil {
		return nil, fmt.Errorf("failed to create shopping item: %w", err)
	}

	// Record history immediately if the item is created as completed
	if item.IsCompleted && s.history != nil {
		if _, err := s.history.Record(item.ListID, item.ID, userID, item.Name, item.Quantity, item.Unit, item.CategoryID); err != nil {
			return nil, fmt.Errorf("failed to record shopping history: %w", err)
		}
	}

	return s.GetByID(item.ID)
}

// GetByID retrieves a shopping item by ID.
func (s *Service) GetByID(id uint) (*ShoppingItem, error) {
	var item ShoppingItem
	if err := s.db.Preload("Category").First(&item, id).Error; err != nil {
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
	if err := s.db.Preload("Category").Where("list_id = ?", listID).Order("is_completed ASC, created_at ASC").Find(&items).Error; err != nil {
		return nil, fmt.Errorf("failed to list shopping items: %w", err)
	}
	return items, nil
}

// Update updates a shopping item.
func (s *Service) Update(id, userID uint, req *UpdateRequest) (*ShoppingItem, error) {
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
	if req.Quantity != nil && *req.Quantity > 0 {
		item.Quantity = *req.Quantity
	}
	if req.Unit != "" {
		item.Unit = strings.TrimSpace(req.Unit)
	}
	if req.CategoryID != nil {
		if err := s.validateCategoryID(req.CategoryID); err != nil {
			return nil, err
		}
		// category_id of 0 clears the category assignment
		if *req.CategoryID == 0 {
			item.CategoryID = nil
		} else {
			item.CategoryID = req.CategoryID
		}
	}
	if req.IsCompleted != nil {
		// Record history when transitioning to completed
		if *req.IsCompleted && !item.IsCompleted && s.history != nil {
			if _, err := s.history.Record(item.ListID, item.ID, userID, item.Name, item.Quantity, item.Unit, item.CategoryID); err != nil {
				return nil, fmt.Errorf("failed to record shopping history: %w", err)
			}
		}
		item.IsCompleted = *req.IsCompleted
	}

	if err := s.db.Save(&item).Error; err != nil {
		return nil, fmt.Errorf("failed to update shopping item: %w", err)
	}

	return s.GetByID(item.ID)
}

// UpdateCompleted updates just the completion status of a shopping item.
func (s *Service) UpdateCompleted(id, userID uint, isCompleted bool) (*ShoppingItem, error) {
	var item ShoppingItem
	if err := s.db.First(&item, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("shopping item not found")
		}
		return nil, fmt.Errorf("failed to get shopping item: %w", err)
	}

	// Record history when transitioning to completed
	if isCompleted && !item.IsCompleted && s.history != nil {
		if _, err := s.history.Record(item.ListID, item.ID, userID, item.Name, item.Quantity, item.Unit, item.CategoryID); err != nil {
			return nil, fmt.Errorf("failed to record shopping history: %w", err)
		}
	}

	item.IsCompleted = isCompleted
	if err := s.db.Save(&item).Error; err != nil {
		return nil, fmt.Errorf("failed to update shopping item status: %w", err)
	}

	return s.GetByID(item.ID)
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

// ReAddFromHistory creates a new shopping item from a history entry, allowing one-tap re-add.
func (s *Service) ReAddFromHistory(historyID, userID uint) (*ShoppingItem, error) {
	entry, err := s.history.GetByID(historyID)
	if err != nil {
		return nil, err
	}

	item := ShoppingItem{
		ListID:      entry.ListID,
		CategoryID:  entry.CategoryID,
		Name:        entry.Name,
		Quantity:    entry.Quantity,
		Unit:        entry.Unit,
		IsCompleted: false,
		CreatedBy:   userID,
	}

	if err := s.db.Create(&item).Error; err != nil {
		return nil, fmt.Errorf("failed to re-add shopping item from history: %w", err)
	}

	return s.GetByID(item.ID)
}

// validateCategoryID checks that a given category ID exists, if provided.
func (s *Service) validateCategoryID(categoryID *uint) error {
	if categoryID == nil || *categoryID == 0 {
		return nil
	}
	var cat category.Category
	if err := s.db.First(&cat, *categoryID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("category not found")
		}
		return fmt.Errorf("failed to validate category: %w", err)
	}
	return nil
}