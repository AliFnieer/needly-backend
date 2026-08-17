package shoppinglist

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
)

// Service handles shopping list business logic.
type Service struct {
	db *gorm.DB
}

// CreateRequest is the payload for creating a shopping list.
type CreateRequest struct {
	Name string `json:"name" binding:"required,min=2,max=150"`
}

// UpdateRequest is the payload for updating a shopping list.
type UpdateRequest struct {
	Name string `json:"name" binding:"required,min=2,max=150"`
}

// NewService creates a new shopping list service.
func NewService(db *gorm.DB) *Service {
	return &Service{
		db: db,
	}
}

// Create creates a new shopping list for a household.
func (s *Service) Create(householdID, userID uint, req *CreateRequest) (*ShoppingList, error) {
	list := ShoppingList{
		HouseholdID: householdID,
		Name:        req.Name,
		CreatedBy:   userID,
	}

	if err := s.db.Create(&list).Error; err != nil {
		return nil, fmt.Errorf("failed to create shopping list: %w", err)
	}

	return &list, nil
}

// GetByID retrieves a shopping list by ID.
func (s *Service) GetByID(id uint) (*ShoppingList, error) {
	var list ShoppingList
	if err := s.db.Preload("Items.Category").First(&list, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("shopping list not found")
		}
		return nil, fmt.Errorf("failed to get shopping list: %w", err)
	}
	return &list, nil
}

// ListByHouseholdID retrieves all shopping lists for a household.
func (s *Service) ListByHouseholdID(householdID uint) ([]ShoppingList, error) {
	var lists []ShoppingList
	if err := s.db.Preload("Items.Category").Where("household_id = ?", householdID).Order("created_at DESC").Find(&lists).Error; err != nil {
		return nil, fmt.Errorf("failed to list shopping lists: %w", err)
	}
	return lists, nil
}

// Update updates a shopping list.
func (s *Service) Update(id uint, req *UpdateRequest) (*ShoppingList, error) {
	var list ShoppingList
	if err := s.db.First(&list, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("shopping list not found")
		}
		return nil, fmt.Errorf("failed to get shopping list: %w", err)
	}

	list.Name = req.Name
	if err := s.db.Save(&list).Error; err != nil {
		return nil, fmt.Errorf("failed to update shopping list: %w", err)
	}

	return &list, nil
}

// Delete removes a shopping list and all its items.
func (s *Service) Delete(id uint) error {
	var list ShoppingList
	if err := s.db.First(&list, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("shopping list not found")
		}
		return fmt.Errorf("failed to get shopping list: %w", err)
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		// Delete all items in the list
		if err := tx.Exec("DELETE FROM shopping_items WHERE list_id = ?", id).Error; err != nil {
			return err
		}
		// Delete the list
		if err := tx.Delete(&list).Error; err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to delete shopping list: %w", err)
	}

	return nil
}