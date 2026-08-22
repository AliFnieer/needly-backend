package shoppingitem_test

import (
	"context"
	"testing"
	"time"

	"github.com/AliFnieer/needly-backend/internal/category"
	"github.com/AliFnieer/needly-backend/internal/history"
	"github.com/AliFnieer/needly-backend/internal/shoppingitem"
	"github.com/AliFnieer/needly-backend/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func seedList(t *testing.T, db *gorm.DB, userID uint) uint {
	t.Helper()
	result := db.Exec(
		`INSERT INTO shopping_lists (household_id, name, created_by, created_at, updated_at) VALUES (1, 'Test List', ?, datetime('now'), datetime('now'))`,
		userID,
	)
	require.NoError(t, result.Error)
	var id uint
	db.Raw("SELECT id FROM shopping_lists WHERE name = 'Test List'").Scan(&id)
	require.NotZero(t, id)
	return id
}

func seedCategory(t *testing.T, db *gorm.DB, householdID uint) uint {
	t.Helper()
	cat := category.Category{
		HouseholdID: householdID,
		Name:        "Test Category",
	}
	require.NoError(t, db.Create(&cat).Error)
	return cat.ID
}

func TestCreate_NilCategory(t *testing.T) {
	db := testutil.SetupTestDB(t)
	user := testutil.SeedUser(t, db, "create_nil@example.com", "password123")
	listID := seedList(t, db, user.ID)

	svc := shoppingitem.NewService(db, nil, nil, nil)
	ctx := context.Background()

	req := &shoppingitem.CreateRequest{
		Name:     "Milk",
		Quantity: 2,
		Unit:     "liters",
	}

	item, err := svc.Create(ctx, listID, user.ID, req)
	require.NoError(t, err)
	assert.NotZero(t, item.ID)
	assert.Equal(t, listID, item.ListID)
	assert.Equal(t, "Milk", item.Name)
	assert.Equal(t, float64(2), item.Quantity)
	assert.Equal(t, "liters", item.Unit)
	assert.Nil(t, item.CategoryID)
	assert.False(t, item.IsCompleted)
	assert.Equal(t, user.ID, item.CreatedBy)
}

func TestCreate_WithValidCategory(t *testing.T) {
	db := testutil.SetupTestDB(t)
	user := testutil.SeedUser(t, db, "create_cat@example.com", "password123")
	listID := seedList(t, db, user.ID)
	catID := seedCategory(t, db, 1)

	svc := shoppingitem.NewService(db, nil, nil, nil)
	ctx := context.Background()

	req := &shoppingitem.CreateRequest{
		Name:       "Bread",
		Quantity:   1,
		Unit:       "loaf",
		CategoryID: &catID,
	}

	item, err := svc.Create(ctx, listID, user.ID, req)
	require.NoError(t, err)
	require.NotNil(t, item.CategoryID)
	assert.Equal(t, catID, *item.CategoryID)
}

func TestCreate_NonexistentCategory(t *testing.T) {
	db := testutil.SetupTestDB(t)
	user := testutil.SeedUser(t, db, "create_nocat@example.com", "password123")
	listID := seedList(t, db, user.ID)

	svc := shoppingitem.NewService(db, nil, nil, nil)
	ctx := context.Background()

	fakeID := uint(9999)
	req := &shoppingitem.CreateRequest{
		Name:       "Eggs",
		CategoryID: &fakeID,
	}

	_, err := svc.Create(ctx, listID, user.ID, req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "category not found")
}

func TestGetByID(t *testing.T) {
	db := testutil.SetupTestDB(t)
	user := testutil.SeedUser(t, db, "getbyid@example.com", "password123")
	listID := seedList(t, db, user.ID)

	svc := shoppingitem.NewService(db, nil, nil, nil)
	ctx := context.Background()

	createReq := &shoppingitem.CreateRequest{Name: "Butter", Quantity: 1}
	created, err := svc.Create(ctx, listID, user.ID, createReq)
	require.NoError(t, err)

	got, err := svc.GetByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, got.ID)
	assert.Equal(t, "Butter", got.Name)
}

func TestGetByID_NotFound(t *testing.T) {
	db := testutil.SetupTestDB(t)
	svc := shoppingitem.NewService(db, nil, nil, nil)
	ctx := context.Background()

	_, err := svc.GetByID(ctx, 9999)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestListByListID(t *testing.T) {
	db := testutil.SetupTestDB(t)
	user := testutil.SeedUser(t, db, "listbyid@example.com", "password123")
	listID := seedList(t, db, user.ID)

	svc := shoppingitem.NewService(db, nil, nil, nil)
	ctx := context.Background()

	svc.Create(ctx, listID, user.ID, &shoppingitem.CreateRequest{Name: "Item A"})
	svc.Create(ctx, listID, user.ID, &shoppingitem.CreateRequest{Name: "Item B"})

	items, err := svc.ListByListID(ctx, listID)
	require.NoError(t, err)
	assert.Len(t, items, 2)
}

func TestListByListID_Empty(t *testing.T) {
	db := testutil.SetupTestDB(t)
	svc := shoppingitem.NewService(db, nil, nil, nil)
	ctx := context.Background()

	items, err := svc.ListByListID(ctx, 9999)
	require.NoError(t, err)
	assert.Empty(t, items)
}

func TestUpdate(t *testing.T) {
	db := testutil.SetupTestDB(t)
	user := testutil.SeedUser(t, db, "update@example.com", "password123")
	listID := seedList(t, db, user.ID)

	svc := shoppingitem.NewService(db, nil, nil, nil)
	ctx := context.Background()

	created, err := svc.Create(ctx, listID, user.ID, &shoppingitem.CreateRequest{Name: "Old Name", Quantity: 1})
	require.NoError(t, err)

	newQty := float64(5)
	comp := true
	updateReq := &shoppingitem.UpdateRequest{
		Name:        "New Name",
		Quantity:    &newQty,
		Unit:        "kg",
		IsCompleted: &comp,
	}

	updated, err := svc.Update(ctx, created.ID, user.ID, updateReq)
	require.NoError(t, err)
	assert.Equal(t, "New Name", updated.Name)
	assert.Equal(t, float64(5), updated.Quantity)
	assert.Equal(t, "kg", updated.Unit)
	assert.True(t, updated.IsCompleted)
}

func TestUpdate_CategoryClear(t *testing.T) {
	db := testutil.SetupTestDB(t)
	user := testutil.SeedUser(t, db, "update_catclear@example.com", "password123")
	listID := seedList(t, db, user.ID)
	catID := seedCategory(t, db, 1)

	svc := shoppingitem.NewService(db, nil, nil, nil)
	ctx := context.Background()

	created, err := svc.Create(ctx, listID, user.ID, &shoppingitem.CreateRequest{Name: "Item", CategoryID: &catID})
	require.NoError(t, err)
	require.NotNil(t, created.CategoryID)

	clearID := uint(0)
	updated, err := svc.Update(ctx, created.ID, user.ID, &shoppingitem.UpdateRequest{CategoryID: &clearID})
	require.NoError(t, err)
	assert.Nil(t, updated.CategoryID)
}

func TestUpdate_NotFound(t *testing.T) {
	db := testutil.SetupTestDB(t)
	svc := shoppingitem.NewService(db, nil, nil, nil)
	ctx := context.Background()

	name := "nope"
	_, err := svc.Update(ctx, 9999, 1, &shoppingitem.UpdateRequest{Name: name})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestUpdateCompleted(t *testing.T) {
	db := testutil.SetupTestDB(t)
	user := testutil.SeedUser(t, db, "updatecomp@example.com", "password123")
	listID := seedList(t, db, user.ID)

	svc := shoppingitem.NewService(db, nil, nil, nil)
	ctx := context.Background()

	created, err := svc.Create(ctx, listID, user.ID, &shoppingitem.CreateRequest{Name: "Cheese"})
	require.NoError(t, err)
	assert.False(t, created.IsCompleted)

	updated, err := svc.UpdateCompleted(ctx, created.ID, user.ID, true)
	require.NoError(t, err)
	assert.True(t, updated.IsCompleted)

	updated2, err := svc.UpdateCompleted(ctx, created.ID, user.ID, false)
	require.NoError(t, err)
	assert.False(t, updated2.IsCompleted)
}

func TestUpdateCompleted_NilHistory_NoPanic(t *testing.T) {
	db := testutil.SetupTestDB(t)
	user := testutil.SeedUser(t, db, "nilhist@example.com", "password123")
	listID := seedList(t, db, user.ID)

	svc := shoppingitem.NewService(db, nil, nil, nil)
	ctx := context.Background()

	created, err := svc.Create(ctx, listID, user.ID, &shoppingitem.CreateRequest{Name: "Yogurt"})
	require.NoError(t, err)

	updated, err := svc.UpdateCompleted(ctx, created.ID, user.ID, true)
	require.NoError(t, err)
	assert.True(t, updated.IsCompleted)
}

func TestUpdateCompleted_NotFound(t *testing.T) {
	db := testutil.SetupTestDB(t)
	svc := shoppingitem.NewService(db, nil, nil, nil)
	ctx := context.Background()

	_, err := svc.UpdateCompleted(ctx, 9999, 1, true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestDelete(t *testing.T) {
	db := testutil.SetupTestDB(t)
	user := testutil.SeedUser(t, db, "delete@example.com", "password123")
	listID := seedList(t, db, user.ID)

	svc := shoppingitem.NewService(db, nil, nil, nil)
	ctx := context.Background()

	created, err := svc.Create(ctx, listID, user.ID, &shoppingitem.CreateRequest{Name: "Juice"})
	require.NoError(t, err)

	err = svc.Delete(ctx, created.ID)
	require.NoError(t, err)

	_, err = svc.GetByID(ctx, created.ID)
	require.Error(t, err)
}

func TestDelete_NotFound(t *testing.T) {
	db := testutil.SetupTestDB(t)
	svc := shoppingitem.NewService(db, nil, nil, nil)
	ctx := context.Background()

	err := svc.Delete(ctx, 9999)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestReAddFromHistory(t *testing.T) {
	db := testutil.SetupTestDB(t)
	user := testutil.SeedUser(t, db, "readd@example.com", "password123")
	listID := seedList(t, db, user.ID)

	svc := shoppingitem.NewService(db, nil, nil, nil)
	ctx := context.Background()

	result := db.Exec(
		`INSERT INTO shopping_history (list_id, name, quantity, unit, completed_by, completed_at, created_at, updated_at) VALUES (?, 'Chips', 3, 'bags', ?, datetime('now'), datetime('now'), datetime('now'))`,
		listID, user.ID,
	)
	require.NoError(t, result.Error)

	var historyID uint
	db.Raw("SELECT id FROM shopping_history WHERE name = 'Chips'").Scan(&historyID)
	require.NotZero(t, historyID)

	item, err := svc.ReAddFromHistory(ctx, historyID, user.ID)
	require.NoError(t, err)
	assert.Equal(t, "Chips", item.Name)
	assert.Equal(t, float64(3), item.Quantity)
	assert.Equal(t, listID, item.ListID)
	assert.False(t, item.IsCompleted)
	assert.Equal(t, user.ID, item.CreatedBy)
}

func TestReAddFromHistory_NotFound(t *testing.T) {
	db := testutil.SetupTestDB(t)
	svc := shoppingitem.NewService(db, nil, nil, nil)
	ctx := context.Background()

	_, err := svc.ReAddFromHistory(ctx, 9999, 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestCreate_WithRecurrenceRule(t *testing.T) {
	db := testutil.SetupTestDB(t)
	user := testutil.SeedUser(t, db, "create_recur@example.com", "password123")
	listID := seedList(t, db, user.ID)

	svc := shoppingitem.NewService(db, nil, nil, nil)
	ctx := context.Background()

	item, err := svc.Create(ctx, listID, user.ID, &shoppingitem.CreateRequest{
		Name:           "Milk",
		RecurrenceRule: "weekly",
	})
	require.NoError(t, err)
	assert.Equal(t, "weekly", item.RecurrenceRule)
	assert.Nil(t, item.NextDueAt)

	var stored shoppingitem.ShoppingItem
	require.NoError(t, db.First(&stored, item.ID).Error)
	assert.Equal(t, "weekly", stored.RecurrenceRule)
}

func TestCreate_InvalidRecurrenceRule(t *testing.T) {
	db := testutil.SetupTestDB(t)
	user := testutil.SeedUser(t, db, "create_badrecur@example.com", "password123")
	listID := seedList(t, db, user.ID)

	svc := shoppingitem.NewService(db, nil, nil, nil)
	ctx := context.Background()

	_, err := svc.Create(ctx, listID, user.ID, &shoppingitem.CreateRequest{
		Name:           "Milk",
		RecurrenceRule: "hourly",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid recurrence rule")
}

func TestUpdate_SetAndClearRecurrence(t *testing.T) {
	db := testutil.SetupTestDB(t)
	user := testutil.SeedUser(t, db, "update_recur@example.com", "password123")
	listID := seedList(t, db, user.ID)

	svc := shoppingitem.NewService(db, nil, nil, nil)
	ctx := context.Background()

	item, err := svc.Create(ctx, listID, user.ID, &shoppingitem.CreateRequest{Name: "Bread"})
	require.NoError(t, err)
	assert.Empty(t, item.RecurrenceRule)

	monthly := "monthly"
	updated, err := svc.Update(ctx, item.ID, user.ID, &shoppingitem.UpdateRequest{RecurrenceRule: &monthly})
	require.NoError(t, err)
	assert.Equal(t, "monthly", updated.RecurrenceRule)

	clear := ""
	updated2, err := svc.Update(ctx, item.ID, user.ID, &shoppingitem.UpdateRequest{RecurrenceRule: &clear})
	require.NoError(t, err)
	assert.Empty(t, updated2.RecurrenceRule)
	assert.Nil(t, updated2.NextDueAt)
}

func TestUpdate_InvalidRecurrenceRule(t *testing.T) {
	db := testutil.SetupTestDB(t)
	user := testutil.SeedUser(t, db, "update_badrecur@example.com", "password123")
	listID := seedList(t, db, user.ID)

	svc := shoppingitem.NewService(db, nil, nil, nil)
	ctx := context.Background()

	item, err := svc.Create(ctx, listID, user.ID, &shoppingitem.CreateRequest{Name: "Eggs"})
	require.NoError(t, err)

	bad := "fortnightly"
	_, err = svc.Update(ctx, item.ID, user.ID, &shoppingitem.UpdateRequest{RecurrenceRule: &bad})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid recurrence rule")
}

func TestUpdateCompleted_Recurring_SetsNextDueAt(t *testing.T) {
	db := testutil.SetupTestDB(t)
	user := testutil.SeedUser(t, db, "comp_recur@example.com", "password123")
	listID := seedList(t, db, user.ID)

	svc := shoppingitem.NewService(db, nil, history.NewService(db), nil)
	ctx := context.Background()

	item, err := svc.Create(ctx, listID, user.ID, &shoppingitem.CreateRequest{
		Name:           "Milk",
		RecurrenceRule: "weekly",
	})
	require.NoError(t, err)

	before := time.Now().UTC()
	completed, err := svc.UpdateCompleted(ctx, item.ID, user.ID, true)
	require.NoError(t, err)
	assert.True(t, completed.IsCompleted)
	require.NotNil(t, completed.NextDueAt)

	expected := before.AddDate(0, 0, 7)
	assert.WithinDuration(t, expected, *completed.NextDueAt, 5*time.Second)

	var historyCount int64
	require.NoError(t, db.Model(&history.ShoppingHistory{}).Where("name = ?", "Milk").Count(&historyCount).Error)
	assert.EqualValues(t, 1, historyCount)

	// Completing again must not duplicate the history entry
	_, err = svc.UpdateCompleted(ctx, item.ID, user.ID, true)
	require.NoError(t, err)
	require.NoError(t, db.Model(&history.ShoppingHistory{}).Where("name = ?", "Milk").Count(&historyCount).Error)
	assert.EqualValues(t, 1, historyCount)
}

func TestUpdateCompleted_Recurring_UncompleteClearsNextDueAt(t *testing.T) {
	db := testutil.SetupTestDB(t)
	user := testutil.SeedUser(t, db, "uncomp_recur@example.com", "password123")
	listID := seedList(t, db, user.ID)

	svc := shoppingitem.NewService(db, nil, nil, nil)
	ctx := context.Background()

	item, err := svc.Create(ctx, listID, user.ID, &shoppingitem.CreateRequest{
		Name:           "Milk",
		RecurrenceRule: "daily",
	})
	require.NoError(t, err)

	completed, err := svc.UpdateCompleted(ctx, item.ID, user.ID, true)
	require.NoError(t, err)
	require.NotNil(t, completed.NextDueAt)

	reopened, err := svc.UpdateCompleted(ctx, item.ID, user.ID, false)
	require.NoError(t, err)
	assert.False(t, reopened.IsCompleted)
	assert.Nil(t, reopened.NextDueAt)
}

func TestUpdateCompleted_NonRecurring_NextDueAtStaysNil(t *testing.T) {
	db := testutil.SetupTestDB(t)
	user := testutil.SeedUser(t, db, "comp_plain@example.com", "password123")
	listID := seedList(t, db, user.ID)

	svc := shoppingitem.NewService(db, nil, nil, nil)
	ctx := context.Background()

	item, err := svc.Create(ctx, listID, user.ID, &shoppingitem.CreateRequest{Name: "Chips"})
	require.NoError(t, err)

	completed, err := svc.UpdateCompleted(ctx, item.ID, user.ID, true)
	require.NoError(t, err)
	assert.True(t, completed.IsCompleted)
	assert.Empty(t, completed.RecurrenceRule)
	assert.Nil(t, completed.NextDueAt)
}

func TestUpdate_CompletingRecurring_SetsNextDueAt(t *testing.T) {
	db := testutil.SetupTestDB(t)
	user := testutil.SeedUser(t, db, "upd_comp_recur@example.com", "password123")
	listID := seedList(t, db, user.ID)

	svc := shoppingitem.NewService(db, nil, nil, nil)
	ctx := context.Background()

	item, err := svc.Create(ctx, listID, user.ID, &shoppingitem.CreateRequest{
		Name:           "Coffee",
		RecurrenceRule: "biweekly",
	})
	require.NoError(t, err)

	complete := true
	before := time.Now().UTC()
	updated, err := svc.Update(ctx, item.ID, user.ID, &shoppingitem.UpdateRequest{IsCompleted: &complete})
	require.NoError(t, err)
	assert.True(t, updated.IsCompleted)
	require.NotNil(t, updated.NextDueAt)
	assert.WithinDuration(t, before.AddDate(0, 0, 14), *updated.NextDueAt, 5*time.Second)
}

func TestListByListID_RollsOverDueItems(t *testing.T) {
	db := testutil.SetupTestDB(t)
	user := testutil.SeedUser(t, db, "rollover@example.com", "password123")
	listID := seedList(t, db, user.ID)

	svc := shoppingitem.NewService(db, nil, nil, nil)
	ctx := context.Background()

	dueItem, err := svc.Create(ctx, listID, user.ID, &shoppingitem.CreateRequest{
		Name:           "Milk",
		RecurrenceRule: "daily",
	})
	require.NoError(t, err)
	notDueItem, err := svc.Create(ctx, listID, user.ID, &shoppingitem.CreateRequest{
		Name:           "Rice",
		RecurrenceRule: "weekly",
	})
	require.NoError(t, err)

	// Complete both items
	_, err = svc.UpdateCompleted(ctx, dueItem.ID, user.ID, true)
	require.NoError(t, err)
	_, err = svc.UpdateCompleted(ctx, notDueItem.ID, user.ID, true)
	require.NoError(t, err)

	// Rewind one item's due date into the past
	past := time.Now().Add(-2 * time.Hour).UTC()
	require.NoError(t, db.Model(&shoppingitem.ShoppingItem{}).Where("id = ?", dueItem.ID).
		Update("next_due_at", past).Error)

	items, err := svc.ListByListID(ctx, listID)
	require.NoError(t, err)

	byName := make(map[string]shoppingitem.ShoppingItem, len(items))
	for _, it := range items {
		byName[it.Name] = it
	}

	milk := byName["Milk"]
	assert.False(t, milk.IsCompleted)
	assert.Nil(t, milk.NextDueAt)

	rice := byName["Rice"]
	assert.True(t, rice.IsCompleted)
	require.NotNil(t, rice.NextDueAt)

	// The rollover persists across subsequent fetches
	items2, err := svc.ListByListID(ctx, listID)
	require.NoError(t, err)
	for _, it := range items2 {
		if it.Name == "Milk" {
			assert.False(t, it.IsCompleted)
			assert.Nil(t, it.NextDueAt)
		}
	}
}

func TestListByListID_RolloverPersistsToDB(t *testing.T) {
	db := testutil.SetupTestDB(t)
	user := testutil.SeedUser(t, db, "rollover_db@example.com", "password123")
	listID := seedList(t, db, user.ID)

	svc := shoppingitem.NewService(db, nil, nil, nil)
	ctx := context.Background()

	item, err := svc.Create(ctx, listID, user.ID, &shoppingitem.CreateRequest{
		Name:           "Water",
		RecurrenceRule: "biweekly",
	})
	require.NoError(t, err)
	_, err = svc.UpdateCompleted(ctx, item.ID, user.ID, true)
	require.NoError(t, err)

	past := time.Now().Add(-24 * time.Hour).UTC()
	require.NoError(t, db.Model(&shoppingitem.ShoppingItem{}).Where("id = ?", item.ID).
		Update("next_due_at", past).Error)

	_, err = svc.ListByListID(ctx, listID)
	require.NoError(t, err)

	var stored shoppingitem.ShoppingItem
	require.NoError(t, db.First(&stored, item.ID).Error)
	assert.False(t, stored.IsCompleted)
	assert.Nil(t, stored.NextDueAt)
}