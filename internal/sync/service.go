package sync

import (
	"context"
	"fmt"
	"time"

	"github.com/AliFnieer/needly-backend/internal/shoppingitem"
	"github.com/AliFnieer/needly-backend/internal/shoppinglist"
	"gorm.io/gorm"
)

// Service builds synchronization snapshots for households so clients can
// catch up after being offline.
type Service struct {
	db *gorm.DB
}

// NewService creates a new sync service.
func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

// Snapshot is a point-in-time view of every list and item in a household.
// Clients reconcile by diffing the snapshot against local state: entities
// present locally but absent from the snapshot were deleted remotely.
type Snapshot struct {
	HouseholdID uint                        `json:"household_id"`
	ServerTime  time.Time                   `json:"server_time"`
	Lists       []shoppinglist.ShoppingList `json:"lists"`
	Items       []shoppingitem.ShoppingItem `json:"items"`
}

// Snapshot returns the household's lists and items, optionally filtered to
// entities changed after the given timestamp (delta mode).
//
// Delta filtering happens in Go rather than SQL: stored timestamps may mix
// timezone representations (e.g. "+02:00" vs "Z"), which makes raw string
// comparisons unreliable on SQLite. Comparing parsed time values here is
// correct regardless of storage format.
func (s *Service) Snapshot(ctx context.Context, householdID uint, since *time.Time) (*Snapshot, error) {
	listQuery := s.db.WithContext(ctx).Where("household_id = ?", householdID)

	var lists []shoppinglist.ShoppingList
	if err := listQuery.Order("id ASC").Find(&lists).Error; err != nil {
		return nil, fmt.Errorf("failed to load household lists: %w", err)
	}

	itemQuery := s.db.WithContext(ctx).
		Where("list_id IN (?)",
			s.db.Model(&shoppinglist.ShoppingList{}).
				Select("id").
				Where("household_id = ?", householdID))

	var items []shoppingitem.ShoppingItem
	if err := itemQuery.Order("id ASC").Find(&items).Error; err != nil {
		return nil, fmt.Errorf("failed to load household items: %w", err)
	}

	if since != nil {
		lists = filterChanged(lists, *since, func(l shoppinglist.ShoppingList) time.Time { return l.UpdatedAt })
		items = filterChanged(items, *since, func(i shoppingitem.ShoppingItem) time.Time { return i.UpdatedAt })
	}

	return &Snapshot{
		HouseholdID: householdID,
		ServerTime:  time.Now().UTC(),
		Lists:       lists,
		Items:       items,
	}, nil
}

// filterChanged keeps only entities whose update timestamp is after since.
func filterChanged[T any](entities []T, since time.Time, updatedAt func(T) time.Time) []T {
	changed := entities[:0]
	for _, e := range entities {
		if updatedAt(e).After(since) {
			changed = append(changed, e)
		}
	}
	return changed
}
