package category

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/AliFnieer/needly-backend/internal/cache"
	"gorm.io/gorm"
)

const categoryCacheKeyPrefix = "category:"
const categoryListCacheKeyPrefix = "categories:household:"

// Service handles category business logic scoped to households.
type Service struct {
	db    *gorm.DB
	cache *cache.Cache
}

// CreateRequest is the payload for creating a category.
type CreateRequest struct {
	Name string `json:"name" binding:"required,min=1,max=100"`
}

// UpdateRequest is the payload for updating a category.
type UpdateRequest struct {
	Name string `json:"name" binding:"omitempty,min=1,max=100"`
}

// ReorderRequest is the payload for reordering a household's categories.
// CategoryIDs must contain every category in the household exactly once, in
// the desired display order (following sort_order ascending).
type ReorderRequest struct {
	CategoryIDs []uint `json:"category_ids" binding:"required,min=1"`
}

// NewService creates a new category service.
func NewService(db *gorm.DB, cacheClient *cache.Cache) *Service {
	return &Service{
		db:    db,
		cache: cacheClient,
	}
}

// Create adds a new category within a household.
func (s *Service) Create(householdID uint, req *CreateRequest) (*Category, error) {
	var maxOrder uint
	s.db.Model(&Category{}).
		Where("household_id = ?", householdID).
		Select("COALESCE(MAX(sort_order), 0)").
		Scan(&maxOrder)

	category := Category{
		HouseholdID: householdID,
		Name:        req.Name,
		SortOrder:   maxOrder + 1,
	}

	if err := s.db.Create(&category).Error; err != nil {
		return nil, fmt.Errorf("failed to create category: %w", err)
	}

	s.invalidateCache(householdID)

	return &category, nil
}

// GetByID retrieves a category by ID, scoped to household.
func (s *Service) GetByID(id, householdID uint) (*Category, error) {
	cacheKey := fmt.Sprintf("%s%d:%d", categoryCacheKeyPrefix, householdID, id)
	if s.cache != nil {
		var cached Category
		if hit, err := s.cache.Get(context.Background(), cacheKey, &cached); hit && err == nil {
			return &cached, nil
		}
	}

	var category Category
	if err := s.db.Where("id = ? AND household_id = ?", id, householdID).First(&category).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("category not found")
		}
		return nil, fmt.Errorf("failed to get category: %w", err)
	}

	if s.cache != nil {
		if err := s.cache.Set(context.Background(), cacheKey, &category); err != nil {
			slog.Warn("failed to cache category", "error", err)
		}
	}

	return &category, nil
}

// List retrieves all categories for a household.
func (s *Service) List(householdID uint) ([]Category, error) {
	cacheKey := fmt.Sprintf("%s%d", categoryListCacheKeyPrefix, householdID)
	if s.cache != nil {
		var cached []Category
		if hit, err := s.cache.Get(context.Background(), cacheKey, &cached); hit && err == nil {
			return cached, nil
		}
	}

	var categories []Category
	if err := s.db.Where("household_id = ?", householdID).Order("sort_order ASC, id ASC").Find(&categories).Error; err != nil {
		return nil, fmt.Errorf("failed to list categories: %w", err)
	}

	if s.cache != nil {
		if err := s.cache.Set(context.Background(), cacheKey, &categories); err != nil {
			slog.Warn("failed to cache categories", "error", err)
		}
	}

	return categories, nil
}

// Update updates an existing category within a household.
func (s *Service) Update(id, householdID uint, req *UpdateRequest) (*Category, error) {
	var category Category
	if err := s.db.Where("id = ? AND household_id = ?", id, householdID).First(&category).Error; err != nil {
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

	s.invalidateCache(householdID)

	return &category, nil
}

// Delete removes a category within a household.
func (s *Service) Delete(id, householdID uint) error {
	var category Category
	if err := s.db.Where("id = ? AND household_id = ?", id, householdID).First(&category).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("category not found")
		}
		return fmt.Errorf("failed to get category: %w", err)
	}

	if err := s.db.Delete(&category).Error; err != nil {
		return fmt.Errorf("failed to delete category: %w", err)
	}

	s.invalidateCache(householdID)

	return nil
}

// Reorder persists a new display order for a household's categories.
func (s *Service) Reorder(householdID uint, ids []uint) error {
	seen := make(map[uint]struct{}, len(ids))
	for _, id := range ids {
		if _, dup := seen[id]; dup {
			return errors.New("duplicate category ids in category_ids")
		}
		seen[id] = struct{}{}
	}

	var total int64
	if err := s.db.Model(&Category{}).Where("household_id = ?", householdID).Count(&total).Error; err != nil {
		return fmt.Errorf("failed to count categories: %w", err)
	}
	if len(ids) != int(total) {
		return errors.New("category_ids must contain every category in the household exactly once")
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		for i, id := range ids {
			res := tx.Model(&Category{}).Where("id = ? AND household_id = ?", id, householdID).
				Update("sort_order", i+1)
			if res.Error != nil {
				return fmt.Errorf("failed to reorder categories: %w", res.Error)
			}
			if res.RowsAffected == 0 {
				return errors.New("category not found in household")
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	s.invalidateCache(householdID)
	return nil
}

func (s *Service) invalidateCache(householdID uint) {
	if s.cache == nil {
		return
	}
	ctx := context.Background()
	pattern := fmt.Sprintf("%s%d:*", categoryCacheKeyPrefix, householdID)
	if err := s.cache.DeleteByPattern(ctx, pattern); err != nil {
		slog.Warn("failed to invalidate category cache", "error", err)
	}
	listKey := fmt.Sprintf("%s%d", categoryListCacheKeyPrefix, householdID)
	if err := s.cache.Delete(ctx, listKey); err != nil {
		slog.Warn("failed to invalidate category list cache", "error", err)
	}
}
