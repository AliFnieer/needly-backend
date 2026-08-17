package history

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// Service handles shopping history business logic.
type Service struct {
	db *gorm.DB
}

// NewService creates a new shopping history service.
func NewService(db *gorm.DB) *Service {
	return &Service{
		db: db,
	}
}

// Record creates a history entry from a completed shopping item.
func (s *Service) Record(listID, itemID, completedBy uint, name string, quantity float64, unit string, categoryID *uint) (*ShoppingHistory, error) {
	entry := ShoppingHistory{
		ListID:      listID,
		ItemID:      &itemID,
		Name:        name,
		Quantity:    quantity,
		Unit:        unit,
		CategoryID:  categoryID,
		CompletedBy: completedBy,
		CompletedAt: time.Now(),
	}

	if err := s.db.Create(&entry).Error; err != nil {
		return nil, fmt.Errorf("failed to record shopping history: %w", err)
	}

	return &entry, nil
}

// ListByListID retrieves history entries for a shopping list.
func (s *Service) ListByListID(listID uint) ([]ShoppingHistory, error) {
	var entries []ShoppingHistory
	if err := s.db.Preload("Category").Where("list_id = ?", listID).Order("completed_at DESC").Find(&entries).Error; err != nil {
		return nil, fmt.Errorf("failed to list shopping history: %w", err)
	}
	return entries, nil
}

// ListByHouseholdID retrieves history entries across all lists in a household.
func (s *Service) ListByHouseholdID(householdID uint) ([]ShoppingHistory, error) {
	var entries []ShoppingHistory
	if err := s.db.Preload("Category").
		Joins("JOIN shopping_lists ON shopping_lists.id = shopping_history.list_id").
		Where("shopping_lists.household_id = ?", householdID).
		Order("shopping_history.completed_at DESC").
		Find(&entries).Error; err != nil {
		return nil, fmt.Errorf("failed to list household shopping history: %w", err)
	}
	return entries, nil
}

// GetByID retrieves a single history entry by ID.
func (s *Service) GetByID(id uint) (*ShoppingHistory, error) {
	var entry ShoppingHistory
	if err := s.db.Preload("Category").First(&entry, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("shopping history entry not found")
		}
		return nil, fmt.Errorf("failed to get shopping history entry: %w", err)
	}
	return &entry, nil
}

// Delete removes a history entry.
func (s *Service) Delete(id uint) error {
	result := s.db.Delete(&ShoppingHistory{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete shopping history entry: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.New("shopping history entry not found")
	}
	return nil
}