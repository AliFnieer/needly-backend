package history_test

import (
	"testing"

	"github.com/AliFnieer/needly-backend/internal/category"
	"github.com/AliFnieer/needly-backend/internal/history"
	"github.com/AliFnieer/needly-backend/internal/shoppinglist"
	"github.com/AliFnieer/needly-backend/internal/testutil"
)

func newTestService(t *testing.T) (*history.Service, *category.Service) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	return history.NewService(db), category.NewService(db, nil)
}

func TestHistory_Record(t *testing.T) {
	svc, _ := newTestService(t)

	entry, err := svc.Record(1, 10, 5, "Milk", 2, "liters", nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if entry.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if entry.ListID != 1 {
		t.Errorf("expected list_id 1, got %d", entry.ListID)
	}
	if entry.ItemID == nil || *entry.ItemID != 10 {
		t.Error("expected item_id 10")
	}
	if entry.Name != "Milk" {
		t.Errorf("expected name Milk, got %s", entry.Name)
	}
	if entry.Quantity != 2 {
		t.Errorf("expected quantity 2, got %f", entry.Quantity)
	}
	if entry.Unit != "liters" {
		t.Errorf("expected unit liters, got %s", entry.Unit)
	}
	if entry.CompletedBy != 5 {
		t.Errorf("expected completed_by 5, got %d", entry.CompletedBy)
	}
	if entry.CompletedAt.IsZero() {
		t.Error("expected CompletedAt to be set")
	}
}

func TestHistory_Record_WithCategory(t *testing.T) {
	svc, catSvc := newTestService(t)

	cat, _ := catSvc.Create(1, &category.CreateRequest{Name: "Dairy"})
	catID := &cat.ID

	entry, err := svc.Record(1, 10, 5, "Cheese", 1, "kg", catID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if entry.CategoryID == nil || *entry.CategoryID != cat.ID {
		t.Error("expected category_id to match")
	}
}

func TestHistory_GetByID(t *testing.T) {
	svc, _ := newTestService(t)

	created, err := svc.Record(1, 10, 5, "Bread", 3, "pcs", nil)
	if err != nil {
		t.Fatalf("record failed: %v", err)
	}

	got, err := svc.GetByID(created.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got.Name != "Bread" {
		t.Errorf("expected name Bread, got %s", got.Name)
	}
	if got.Quantity != 3 {
		t.Errorf("expected quantity 3, got %f", got.Quantity)
	}
}

func TestHistory_GetByID_NotFound(t *testing.T) {
	svc, _ := newTestService(t)

	_, err := svc.GetByID(99999)
	if err == nil {
		t.Fatal("expected error for nonexistent entry")
	}
	if err.Error() != "shopping history entry not found" {
		t.Errorf("expected 'shopping history entry not found', got %v", err)
	}
}

func TestHistory_ListByListID(t *testing.T) {
	svc, _ := newTestService(t)

	_, _ = svc.Record(1, 10, 5, "Milk", 2, "liters", nil)
	_, _ = svc.Record(1, 11, 5, "Bread", 1, "pcs", nil)
	_, _ = svc.Record(2, 12, 5, "Eggs", 12, "pcs", nil)

	entries, err := svc.ListByListID(1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
}

func TestHistory_ListByListID_Empty(t *testing.T) {
	svc, _ := newTestService(t)

	entries, err := svc.ListByListID(999)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestHistory_Delete(t *testing.T) {
	svc, _ := newTestService(t)

	entry, _ := svc.Record(1, 10, 5, "Butter", 1, "pcs", nil)

	err := svc.Delete(entry.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	_, err = svc.GetByID(entry.ID)
	if err == nil {
		t.Fatal("expected error after deletion")
	}
}

func TestHistory_Delete_NotFound(t *testing.T) {
	svc, _ := newTestService(t)

	err := svc.Delete(99999)
	if err == nil {
		t.Fatal("expected error for nonexistent entry")
	}
	if err.Error() != "shopping history entry not found" {
		t.Errorf("expected 'shopping history entry not found', got %v", err)
	}
}

func TestHistory_ListByHouseholdID(t *testing.T) {
	db := testutil.SetupTestDB(t)
	historySvc := history.NewService(db)
	catSvc := category.NewService(db, nil)

	sl1 := &shoppinglist.ShoppingList{HouseholdID: 1, Name: "List A", CreatedBy: 1}
	sl2 := &shoppinglist.ShoppingList{HouseholdID: 1, Name: "List B", CreatedBy: 1}
	if err := db.Create(sl1).Error; err != nil {
		t.Fatalf("seed list A failed: %v", err)
	}
	if err := db.Create(sl2).Error; err != nil {
		t.Fatalf("seed list B failed: %v", err)
	}

	sl3 := &shoppinglist.ShoppingList{HouseholdID: 2, Name: "List C", CreatedBy: 2}
	if err := db.Create(sl3).Error; err != nil {
		t.Fatalf("seed list C failed: %v", err)
	}

	cat, _ := catSvc.Create(1, &category.CreateRequest{Name: "Food"})
	catID := &cat.ID

	_, _ = historySvc.Record(sl1.ID, 10, 1, "Milk", 2, "liters", catID)
	_, _ = historySvc.Record(sl2.ID, 11, 1, "Bread", 1, "pcs", nil)
	_, _ = historySvc.Record(sl3.ID, 12, 2, "Soda", 6, "pcs", nil)

	entries, err := historySvc.ListByHouseholdID(1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries for household 1, got %d", len(entries))
	}

	entries2, err := historySvc.ListByHouseholdID(2)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(entries2) != 1 {
		t.Fatalf("expected 1 entry for household 2, got %d", len(entries2))
	}
	if entries2[0].Name != "Soda" {
		t.Errorf("expected Soda, got %s", entries2[0].Name)
	}
}

func TestHistory_ListByHouseholdID_Empty(t *testing.T) {
	svc, _ := newTestService(t)

	entries, err := svc.ListByHouseholdID(999)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestHistory_Record_WithCategoryPreload(t *testing.T) {
	svc, catSvc := newTestService(t)

	cat, _ := catSvc.Create(1, &category.CreateRequest{Name: "Snacks"})
	entry, _ := svc.Record(1, 10, 5, "Chips", 1, "bag", &cat.ID)

	got, err := svc.GetByID(entry.ID)
	if err != nil {
		t.Fatalf("get by id failed: %v", err)
	}
	if got.Category == nil {
		t.Fatal("expected Category to be preloaded")
	}
	if got.Category.Name != "Snacks" {
		t.Errorf("expected category name Snacks, got %s", got.Category.Name)
	}
}

func TestHistory_ListByListID_Ordering(t *testing.T) {
	svc, _ := newTestService(t)

	entry1, _ := svc.Record(1, 10, 5, "First", 1, "pcs", nil)
	entry2, _ := svc.Record(1, 11, 5, "Second", 2, "pcs", nil)

	entries, err := svc.ListByListID(1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].ID != entry2.ID {
		t.Errorf("expected second entry first (DESC), got ID %d", entries[0].ID)
	}
	if entries[1].ID != entry1.ID {
		t.Errorf("expected first entry second, got ID %d", entries[1].ID)
	}
}
