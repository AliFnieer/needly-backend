package category

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
)

// Service handles category business logic.
type Service struct {
	db *gorm.DB
}

// CreateRequest is the payload for creating a category.
type CreateRequest struct {
	Name string `json:"name" binding:"required,min=1,max=100"`
}

// UpdateRequest is the payload for updating a category.
type UpdateRequest struct {
	Name string `json:"name" binding:"omitempty,min=1,max=100"`
}

// NewService creates a new category service.
func NewService(db *gorm.DB) *Service {
	return &Service{
		db: db,
	}
}

// Create adds a new category.
func (s *Service) Create(req *CreateRequest) (*Category, error) {
	category := Category{
		Name: req.Name,
	}

	if err := s.db.Create(&category).Error; err != nil {
		return nil, fmt.Errorf("failed to create category: %w", err)
	}

	return &category, nil
}

// GetByID retrieves a category by ID.
func (s *Service) GetByID(id uint) (*Category, error) {
	var category Category
	if err := s.db.First(&category, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("category not found")
		}
		return nil, fmt.Errorf("failed to get category: %w", err)
	}
	return &category, nil
}

// List retrieves all categories.
func (s *Service) List() ([]Category, error) {
	var categories []Category
	if err := s.db.Order("name ASC").Find(&categories).Error; err != nil {
		return nil, fmt.Errorf("failed to list categories: %w", err)
	}
	return categories, nil
}

// Update updates an existing category.
func (s *Service) Update(id uint, req *UpdateRequest) (*Category, error) {
	var category Category
	if err := s.db.First(&category, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("category not found")
		}
		return nil, fmt.Errorf("failed to get category: %w", err)
	}

	if req.Name != "" {
		category.Name = req.Name
	}

	if err := s.db.Save(&category).Error; err != nil {
		return nil, fmt.Errorf("failed to update category: %w", err)
	}

	return &category, nil
}

// Delete removes a category. Shopping items assigned to it will have category_id set to NULL.
func (s *Service) Delete(id uint) error {
	var category Category
	if err := s.db.First(&category, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("category not found")
		}
		return fmt.Errorf("failed to get category: %w", err)
	}

	if err := s.db.Delete(&category).Error; err != nil {
		return fmt.Errorf("failed to delete category: %w", err)
	}

	return nil
}