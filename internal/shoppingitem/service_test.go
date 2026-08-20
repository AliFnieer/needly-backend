package shoppingitem_test

import (
	"testing"

	"github.com/AliFnieer/needly-backend/internal/category"
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

	req := &shoppingitem.CreateRequest{
		Name:     "Milk",
		Quantity: 2,
		Unit:     "liters",
	}

	item, err := svc.Create(listID, user.ID, req)
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

	req := &shoppingitem.CreateRequest{
		Name:       "Bread",
		Quantity:   1,
		Unit:       "loaf",
		CategoryID: &catID,
	}

	item, err := svc.Create(listID, user.ID, req)
	require.NoError(t, err)
	require.NotNil(t, item.CategoryID)
	assert.Equal(t, catID, *item.CategoryID)
}

func TestCreate_NonexistentCategory(t *testing.T) {
	db := testutil.SetupTestDB(t)
	user := testutil.SeedUser(t, db, "create_nocat@example.com", "password123")
	listID := seedList(t, db, user.ID)

	svc := shoppingitem.NewService(db, nil, nil, nil)

	fakeID := uint(9999)
	req := &shoppingitem.CreateRequest{
		Name:       "Eggs",
		CategoryID: &fakeID,
	}

	_, err := svc.Create(listID, user.ID, req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "category not found")
}

func TestGetByID(t *testing.T) {
	db := testutil.SetupTestDB(t)
	user := testutil.SeedUser(t, db, "getbyid@example.com", "password123")
	listID := seedList(t, db, user.ID)

	svc := shoppingitem.NewService(db, nil, nil, nil)

	createReq := &shoppingitem.CreateRequest{Name: "Butter", Quantity: 1}
	created, err := svc.Create(listID, user.ID, createReq)
	require.NoError(t, err)

	got, err := svc.GetByID(created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, got.ID)
	assert.Equal(t, "Butter", got.Name)
}

func TestGetByID_NotFound(t *testing.T) {
	db := testutil.SetupTestDB(t)
	svc := shoppingitem.NewService(db, nil, nil, nil)

	_, err := svc.GetByID(9999)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestListByListID(t *testing.T) {
	db := testutil.SetupTestDB(t)
	user := testutil.SeedUser(t, db, "listbyid@example.com", "password123")
	listID := seedList(t, db, user.ID)

	svc := shoppingitem.NewService(db, nil, nil, nil)

	svc.Create(listID, user.ID, &shoppingitem.CreateRequest{Name: "Item A"})
	svc.Create(listID, user.ID, &shoppingitem.CreateRequest{Name: "Item B"})

	items, err := svc.ListByListID(listID)
	require.NoError(t, err)
	assert.Len(t, items, 2)
}

func TestListByListID_Empty(t *testing.T) {
	db := testutil.SetupTestDB(t)
	svc := shoppingitem.NewService(db, nil, nil, nil)

	items, err := svc.ListByListID(9999)
	require.NoError(t, err)
	assert.Empty(t, items)
}

func TestUpdate(t *testing.T) {
	db := testutil.SetupTestDB(t)
	user := testutil.SeedUser(t, db, "update@example.com", "password123")
	listID := seedList(t, db, user.ID)

	svc := shoppingitem.NewService(db, nil, nil, nil)

	created, err := svc.Create(listID, user.ID, &shoppingitem.CreateRequest{Name: "Old Name", Quantity: 1})
	require.NoError(t, err)

	newQty := float64(5)
	comp := true
	updateReq := &shoppingitem.UpdateRequest{
		Name:        "New Name",
		Quantity:    &newQty,
		Unit:        "kg",
		IsCompleted: &comp,
	}

	updated, err := svc.Update(created.ID, user.ID, updateReq)
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

	created, err := svc.Create(listID, user.ID, &shoppingitem.CreateRequest{Name: "Item", CategoryID: &catID})
	require.NoError(t, err)
	require.NotNil(t, created.CategoryID)

	clearID := uint(0)
	updated, err := svc.Update(created.ID, user.ID, &shoppingitem.UpdateRequest{CategoryID: &clearID})
	require.NoError(t, err)
	assert.Nil(t, updated.CategoryID)
}

func TestUpdate_NotFound(t *testing.T) {
	db := testutil.SetupTestDB(t)
	svc := shoppingitem.NewService(db, nil, nil, nil)

	name := "nope"
	_, err := svc.Update(9999, 1, &shoppingitem.UpdateRequest{Name: name})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestUpdateCompleted(t *testing.T) {
	db := testutil.SetupTestDB(t)
	user := testutil.SeedUser(t, db, "updatecomp@example.com", "password123")
	listID := seedList(t, db, user.ID)

	svc := shoppingitem.NewService(db, nil, nil, nil)

	created, err := svc.Create(listID, user.ID, &shoppingitem.CreateRequest{Name: "Cheese"})
	require.NoError(t, err)
	assert.False(t, created.IsCompleted)

	updated, err := svc.UpdateCompleted(created.ID, user.ID, true)
	require.NoError(t, err)
	assert.True(t, updated.IsCompleted)

	updated2, err := svc.UpdateCompleted(created.ID, user.ID, false)
	require.NoError(t, err)
	assert.False(t, updated2.IsCompleted)
}

func TestUpdateCompleted_NilHistory_NoPanic(t *testing.T) {
	db := testutil.SetupTestDB(t)
	user := testutil.SeedUser(t, db, "nilhist@example.com", "password123")
	listID := seedList(t, db, user.ID)

	svc := shoppingitem.NewService(db, nil, nil, nil)

	created, err := svc.Create(listID, user.ID, &shoppingitem.CreateRequest{Name: "Yogurt"})
	require.NoError(t, err)

	updated, err := svc.UpdateCompleted(created.ID, user.ID, true)
	require.NoError(t, err)
	assert.True(t, updated.IsCompleted)
}

func TestUpdateCompleted_NotFound(t *testing.T) {
	db := testutil.SetupTestDB(t)
	svc := shoppingitem.NewService(db, nil, nil, nil)

	_, err := svc.UpdateCompleted(9999, 1, true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestDelete(t *testing.T) {
	db := testutil.SetupTestDB(t)
	user := testutil.SeedUser(t, db, "delete@example.com", "password123")
	listID := seedList(t, db, user.ID)

	svc := shoppingitem.NewService(db, nil, nil, nil)

	created, err := svc.Create(listID, user.ID, &shoppingitem.CreateRequest{Name: "Juice"})
	require.NoError(t, err)

	err = svc.Delete(created.ID)
	require.NoError(t, err)

	_, err = svc.GetByID(created.ID)
	require.Error(t, err)
}

func TestDelete_NotFound(t *testing.T) {
	db := testutil.SetupTestDB(t)
	svc := shoppingitem.NewService(db, nil, nil, nil)

	err := svc.Delete(9999)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestReAddFromHistory(t *testing.T) {
	db := testutil.SetupTestDB(t)
	user := testutil.SeedUser(t, db, "readd@example.com", "password123")
	listID := seedList(t, db, user.ID)

	svc := shoppingitem.NewService(db, nil, nil, nil)

	result := db.Exec(
		`INSERT INTO shopping_history (list_id, name, quantity, unit, completed_by, completed_at, created_at, updated_at) VALUES (?, 'Chips', 3, 'bags', ?, datetime('now'), datetime('now'), datetime('now'))`,
		listID, user.ID,
	)
	require.NoError(t, result.Error)

	var historyID uint
	db.Raw("SELECT id FROM shopping_history WHERE name = 'Chips'").Scan(&historyID)
	require.NotZero(t, historyID)

	item, err := svc.ReAddFromHistory(historyID, user.ID)
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

	_, err := svc.ReAddFromHistory(9999, 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}
