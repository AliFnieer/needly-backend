package shoppingitem

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/AliFnieer/needly-backend/internal/cache"
	"github.com/AliFnieer/needly-backend/internal/category"
	"github.com/AliFnieer/needly-backend/internal/history"
	"github.com/AliFnieer/needly-backend/internal/notification"
	"gorm.io/gorm"
)

const (
	// itemCacheKeyPrefix is the prefix for individual shopping item cache keys.
	itemCacheKeyPrefix = "shoppingitem:"
	// listItemsCacheKeyPrefix is the prefix for list items cache keys.
	listItemsCacheKeyPrefix = "list:"
	// listItemsCacheKeySuffix is the suffix for list items cache keys.
	listItemsCacheKeySuffix = ":items"
)

// Service handles shopping item business logic.
type Service struct {
	db           *gorm.DB
	cache        *cache.Cache
	history      *history.Service
	notification *notification.Service
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
func NewService(db *gorm.DB, cache *cache.Cache, historySvc *history.Service, notificationSvc *notification.Service) *Service {
	return &Service{
		db:           db,
		cache:        cache,
		history:      historySvc,
		notification: notificationSvc,
	}
}

func (s *Service) validateCategoryID(categoryID *uint, householdID uint) error {
	if categoryID == nil || *categoryID == 0 {
		return nil
	}

	var cat category.Category
	if err := s.db.Where("id = ? AND household_id = ?", *categoryID, householdID).First(&cat).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("category not found")
		}
		return fmt.Errorf("failed to validate category: %w", err)
	}

	return nil
}

// Create adds a new item to a shopping list.
func (s *Service) Create(ctx context.Context, listID, userID uint, req *CreateRequest) (*ShoppingItem, error) {
	quantity := req.Quantity
	if quantity <= 0 {
		quantity = 1
	}

	// Resolve the household that owns the list so we can validate the
	// category belongs to the same household (data isolation).
	householdID := s.householdIDForList(listID)
	if err := s.validateCategoryID(req.CategoryID, householdID); err != nil {
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

	// Invalidate the list items cache since a new item was added
	s.invalidateListItems(listID)

	// Notify household members about the new item
	s.notify(ctx, notification.NotificationTypeItemCreated,
		"New shopping item",
		fmt.Sprintf("Item %q was added to the list", item.Name),
		householdID, listID, item.ID, userID)

	return &item, nil
}

// GetByID retrieves a shopping item by ID.
func (s *Service) GetByID(ctx context.Context, id uint) (*ShoppingItem, error) {
	cacheKey := itemCacheKey(id)

	// Try cache first
	var item ShoppingItem
	if s.cache != nil {
		hit, err := s.cache.Get(ctx, cacheKey, &item)
		if err != nil {
			// Log and fall through to DB on cache error
			slog.Error("cache get error", "key", cacheKey, "error", err)
		} else if hit {
			return &item, nil
		}
	}

	// Cache miss - load from DB
	if err := s.db.First(&item, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("shopping item not found")
		}
		return nil, fmt.Errorf("failed to get shopping item: %w", err)
	}

	// Populate cache
	if s.cache != nil {
		if err := s.cache.Set(ctx, cacheKey, &item); err != nil {
			slog.Error("cache set error", "key", cacheKey, "error", err)
		}
	}

	return &item, nil
}

// ListByListID retrieves all items in a shopping list.
func (s *Service) ListByListID(ctx context.Context, listID uint) ([]ShoppingItem, error) {
	cacheKey := listItemsCacheKey(listID)

	// Try cache first
	var items []ShoppingItem
	if s.cache != nil {
		hit, err := s.cache.Get(ctx, cacheKey, &items)
		if err != nil {
			// Log and fall through to DB on cache error
			slog.Error("cache get error", "key", cacheKey, "error", err)
		} else if hit {
			return items, nil
		}
	}

	// Cache miss - load from DB
	if err := s.db.Where("list_id = ?", listID).Order("is_completed ASC, created_at ASC").Find(&items).Error; err != nil {
		return nil, fmt.Errorf("failed to list shopping items: %w", err)
	}

	// Populate cache
	if s.cache != nil {
		if err := s.cache.Set(ctx, cacheKey, &items); err != nil {
			slog.Error("cache set error", "key", cacheKey, "error", err)
		}
	}

	return items, nil
}

// Update updates a shopping item.
func (s *Service) Update(ctx context.Context, id, userID uint, req *UpdateRequest) (*ShoppingItem, error) {
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
		// Validate the category belongs to the same household as the item's list.
		householdID := s.householdIDForList(item.ListID)
		if err := s.validateCategoryID(req.CategoryID, householdID); err != nil {
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

	// Invalidate caches for this item and its list
	s.invalidateItem(item.ID, item.ListID)

	// Notify household members about the updated item
	householdID := s.householdIDForList(item.ListID)
	s.notify(ctx, notification.NotificationTypeItemUpdated,
		"Shopping item updated",
		fmt.Sprintf("Item %q was updated", item.Name),
		householdID, item.ListID, item.ID, userID)

	return &item, nil
}

// UpdateCompleted updates just the completion status of a shopping item.
func (s *Service) UpdateCompleted(ctx context.Context, id, userID uint, isCompleted bool) (*ShoppingItem, error) {
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

	// Invalidate caches for this item and its list
	s.invalidateItem(item.ID, item.ListID)

	// Notify household members about the completion status change
	nt := notification.NotificationTypeItemUpdated
	title := "Shopping item updated"
	if isCompleted {
		nt = notification.NotificationTypeItemCompleted
		title = "Shopping item completed"
	}

	householdID := s.householdIDForList(item.ListID)
	s.notify(ctx, nt,
		title,
		fmt.Sprintf("Item %q was %s", item.Name, map[bool]string{true: "completed", false: "re-opened"}[isCompleted]),
		householdID, item.ListID, item.ID, userID)

	return &item, nil
}

// Delete removes a shopping item.
func (s *Service) Delete(ctx context.Context, id uint) error {
	var item ShoppingItem
	if err := s.db.First(&item, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("shopping item not found")
		}
		return fmt.Errorf("failed to get shopping item: %w", err)
	}

	result := s.db.Delete(&ShoppingItem{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete shopping item: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.New("shopping item not found")
	}

	// Invalidate caches for this item and its list
	s.invalidateItem(item.ID, item.ListID)

	// Notify household members about the deleted item
	householdID := s.householdIDForList(item.ListID)
	s.notify(ctx, notification.NotificationTypeItemDeleted,
		"Shopping item deleted",
		fmt.Sprintf("Item %q was deleted", item.Name),
		householdID, item.ListID, item.ID, 0)

	return nil
}

// ReAddFromHistory recreates a shopping item from a completed history entry.
func (s *Service) ReAddFromHistory(ctx context.Context, historyID, userID uint) (*ShoppingItem, error) {
	var entry history.ShoppingHistory
	if err := s.db.First(&entry, historyID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("shopping history entry not found")
		}
		return nil, fmt.Errorf("failed to get shopping history entry: %w", err)
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
		return nil, fmt.Errorf("failed to recreate shopping item from history: %w", err)
	}

	s.invalidateListItems(item.ListID)

	// Notify household members about the re-added item
	householdID := s.householdIDForList(item.ListID)
	s.notify(ctx, notification.NotificationTypeItemReAdded,
		"Shopping item re-added",
		fmt.Sprintf("Item %q was re-added to the list", item.Name),
		householdID, item.ListID, item.ID, userID)

	return &item, nil
}

// notify delivers a notification to all household members.
func (s *Service) notify(ctx context.Context, nt notification.NotificationType, title, body string, householdID, listID, itemID, actorID uint) {
	if s.notification == nil {
		return
	}

	if err := s.notification.NotifyHousehold(ctx, notification.BuildNotification(nt, title, body, householdID, listID, itemID, actorID)); err != nil {
		slog.Error("shopping item notification error", "error", err)
	}
}

// householdIDForList resolves the household ID for a given shopping list.
func (s *Service) householdIDForList(listID uint) uint {
	var list struct {
		HouseholdID uint
	}
	if err := s.db.Table("shopping_lists").Select("household_id").Where("id = ?", listID).Scan(&list).Error; err != nil {
		slog.Error("failed to resolve household for list", "list_id", listID, "error", err)
		return 0
	}
	return list.HouseholdID
}

// itemCacheKey builds the cache key for a single shopping item.
func itemCacheKey(id uint) string {
	return fmt.Sprintf("%s%d", itemCacheKeyPrefix, id)
}

// listItemsCacheKey builds the cache key for a list's items.
func listItemsCacheKey(listID uint) string {
	return fmt.Sprintf("%s%d%s", listItemsCacheKeyPrefix, listID, listItemsCacheKeySuffix)
}

// invalidateItem removes cached data for a specific item and its list.
func (s *Service) invalidateItem(itemID, listID uint) {
	if s.cache == nil {
		return
	}

	ctx := context.Background()
	keys := []string{
		itemCacheKey(itemID),
		listItemsCacheKey(listID),
	}

	for _, key := range keys {
		if err := s.cache.Delete(ctx, key); err != nil {
			slog.Error("cache delete error", "key", key, "error", err)
		}
	}
}

// invalidateListItems removes cached data for a list's items.
func (s *Service) invalidateListItems(listID uint) {
	if s.cache == nil {
		return
	}

	ctx := context.Background()
	key := listItemsCacheKey(listID)
	if err := s.cache.Delete(ctx, key); err != nil {
		slog.Error("cache delete error", "key", key, "error", err)
	}
}