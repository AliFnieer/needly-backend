package sync_test

import (
	"context"
	"testing"
	"time"

	"github.com/AliFnieer/needly-backend/internal/sync"
	"github.com/AliFnieer/needly-backend/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func seedSyncList(t *testing.T, db *gorm.DB, userID uint, updatedAt time.Time) uint {
	t.Helper()
	result := db.Exec(
		`INSERT INTO shopping_lists (household_id, name, created_by, created_at, updated_at) VALUES (1, 'Sync List', ?, ?, ?)`,
		userID, updatedAt.UTC(), updatedAt.UTC(),
	)
	require.NoError(t, result.Error)
	var id uint
	db.Raw("SELECT id FROM shopping_lists WHERE name = 'Sync List'").Scan(&id)
	require.NotZero(t, id)
	return id
}

func seedSyncItem(t *testing.T, db *gorm.DB, listID, userID uint, name string, updatedAt time.Time) uint {
	t.Helper()
	result := db.Exec(
		`INSERT INTO shopping_items (list_id, name, quantity, is_completed, recurrence_rule, created_by, created_at, updated_at) VALUES (?, ?, 1, 0, '', ?, ?, ?)`,
		listID, name, userID, updatedAt.UTC(), updatedAt.UTC(),
	)
	require.NoError(t, result.Error)
	var id uint
	db.Raw("SELECT id FROM shopping_items WHERE name = ?", name).Scan(&id)
	require.NotZero(t, id)
	return id
}

func TestSnapshot_ReturnsAllListsAndItems(t *testing.T) {
	db := testutil.SetupTestDB(t)
	user := testutil.SeedUser(t, db, "sync_all@example.com", "password123")
	base := time.Now().Add(-2 * time.Hour).UTC()

	listID := seedSyncList(t, db, user.ID, base)
	seedSyncItem(t, db, listID, user.ID, "Milk", base)

	svc := sync.NewService(db)
	snapshot, err := svc.Snapshot(context.Background(), 1, nil)
	require.NoError(t, err)
	assert.Equal(t, uint(1), snapshot.HouseholdID)
	assert.Len(t, snapshot.Lists, 1)
	assert.Len(t, snapshot.Items, 1)
	assert.False(t, snapshot.ServerTime.IsZero())
}

func TestSnapshot_SinceFiltersUnchangedEntities(t *testing.T) {
	db := testutil.SetupTestDB(t)
	user := testutil.SeedUser(t, db, "sync_delta@example.com", "password123")
	base := time.Now().Add(-4 * time.Hour).UTC()

	listID := seedSyncList(t, db, user.ID, base)
	seedSyncItem(t, db, listID, user.ID, "Milk", base)
	seedSyncItem(t, db, listID, user.ID, "Bread", base.Add(2*time.Hour))

	svc := sync.NewService(db)

	// Delta since an hour after the base: only the newer item is returned,
	// and the unchanged list is excluded.
	cutoff := base.Add(1 * time.Hour)
	snapshot, err := svc.Snapshot(context.Background(), 1, &cutoff)
	require.NoError(t, err)
	assert.Empty(t, snapshot.Lists)
	require.Len(t, snapshot.Items, 1)
	assert.Equal(t, "Bread", snapshot.Items[0].Name)

	// A cutoff before everything returns the full household
	early := base.Add(-time.Hour)
	full, err := svc.Snapshot(context.Background(), 1, &early)
	require.NoError(t, err)
	assert.Len(t, full.Lists, 1)
	assert.Len(t, full.Items, 2)
}

func TestSnapshot_EmptyHousehold(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.SeedUser(t, db, "sync_empty@example.com", "password123")

	svc := sync.NewService(db)
	snapshot, err := svc.Snapshot(context.Background(), 42, nil)
	require.NoError(t, err)
	assert.Empty(t, snapshot.Lists)
	assert.Empty(t, snapshot.Items)
}
