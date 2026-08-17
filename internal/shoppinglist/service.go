package shoppinglist

import (
	"context"
	"errors"
	"fmt"

	"github.com/AliFnieer/needly-backend/internal/cache"
	"gorm.io/gorm"
)

const (
	// listCacheKeyPrefix is the prefix for individual shopping list cache keys.
	listCacheKeyPrefix = "shoppinglist:"
	// householdListsCacheKeyPrefix is the prefix for household lists cache keys.
	householdListsCacheKeyPrefix = "household:"
	// householdListsCacheKeySuffix is the suffix for household lists cache keys.
	householdListsCacheKeySuffix = ":lists"
)

// Service handles shopping list business logic.
type Service struct {
	db    *gorm.DB
	cache *cache.Cache
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
func NewService(db *gorm.DB, cache *cache.Cache) *Service {
	return &Service{
		db:    db,
		cache: cache,
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

	// Invalidate the household lists cache since a new list was added
	s.invalidateHouseholdLists(householdID)

	return &list, nil
}

// GetByID retrieves a shopping list by ID.
func (s *Service) GetByID(id uint) (*ShoppingList, error) {
	ctx := context.Background()
	cacheKey := listCacheKey(id)

	// Try cache first
	var list ShoppingList
	if s.cache != nil {
		hit, err := s.cache.Get(ctx, cacheKey, &list)
		if err != nil {
			// Log and fall through to DB on cache error
			fmt.Printf("cache get error for %s: %v\n", cacheKey, err)
		} else if hit {
			return &list, nil
		}
	}

	// Cache miss - load from DB
	if err := s.db.Preload("Items").First(&list, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("shopping list not found")
		}
		return nil, fmt.Errorf("failed to get shopping list: %w", err)
	}

	// Populate cache
	if s.cache != nil {
		if err := s.cache.Set(ctx, cacheKey, &list); err != nil {
			fmt.Printf("cache set error for %s: %v\n", cacheKey, err)
		}
	}

	return &list, nil
}

// ListByHouseholdID retrieves all shopping lists for a household.
func (s *Service) ListByHouseholdID(householdID uint) ([]ShoppingList, error) {
	ctx := context.Background()
	cacheKey := householdListsCacheKey(householdID)

	// Try cache first
	var lists []ShoppingList
	if s.cache != nil {
		hit, err := s.cache.Get(ctx, cacheKey, &lists)
		if err != nil {
			// Log and fall through to DB on cache error
			fmt.Printf("cache get error for %s: %v\n", cacheKey, err)
		} else if hit {
			return lists, nil
		}
	}

	// Cache miss - load from DB
	if err := s.db.Preload("Items").Where("household_id = ?", householdID).Order("created_at DESC").Find(&lists).Error; err != nil {
		return nil, fmt.Errorf("failed to list shopping lists: %w", err)
	}

	// Populate cache
	if s.cache != nil {
		if err := s.cache.Set(ctx, cacheKey, &lists); err != nil {
			fmt.Printf("cache set error for %s: %v\n", cacheKey, err)
		}
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

	// Invalidate caches for this list and its household
	s.invalidateList(list.ID, list.HouseholdID)

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

	// Invalidate caches for this list and its household
	s.invalidateList(list.ID, list.HouseholdID)

	return nil
}

// listCacheKey builds the cache key for a single shopping list.
func listCacheKey(id uint) string {
	return fmt.Sprintf("%s%d", listCacheKeyPrefix, id)
}

// householdListsCacheKey builds the cache key for a household's lists.
func householdListsCacheKey(householdID uint) string {
	return fmt.Sprintf("%s%d%s", householdListsCacheKeyPrefix, householdID, householdListsCacheKeySuffix)
}

// invalidateList removes cached data for a specific list and its household.
func (s *Service) invalidateList(listID, householdID uint) {
	if s.cache == nil {
		return
	}

	ctx := context.Background()
	keys := []string{
		listCacheKey(listID),
		householdListsCacheKey(householdID),
	}

	for _, key := range keys {
		if err := s.cache.Delete(ctx, key); err != nil {
			fmt.Printf("cache delete error for %s: %v\n", key, err)
		}
	}
}

// invalidateHouseholdLists removes cached data for a household's lists.
func (s *Service) invalidateHouseholdLists(householdID uint) {
	if s.cache == nil {
		return
	}

	ctx := context.Background()
	key := householdListsCacheKey(householdID)
	if err := s.cache.Delete(ctx, key); err != nil {
		fmt.Printf("cache delete error for %s: %v\n", key, err)
	}
}