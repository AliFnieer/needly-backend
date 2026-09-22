package category_test

import (
	"testing"

	"github.com/AliFnieer/needly-backend/internal/category"
	"github.com/AliFnieer/needly-backend/internal/testutil"
)

func newTestService(t *testing.T) *category.Service {
	t.Helper()
	db := testutil.SetupTestDB(t)
	return category.NewService(db, nil)
}

func TestCategory_Create(t *testing.T) {
	svc := newTestService(t)

	cat, err := svc.Create(1, &category.CreateRequest{Name: "Groceries"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cat.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if cat.Name != "Groceries" {
		t.Errorf("expected name Groceries, got %s", cat.Name)
	}
	if cat.HouseholdID != 1 {
		t.Errorf("expected household_id 1, got %d", cat.HouseholdID)
	}
	if cat.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
	if cat.UpdatedAt.IsZero() {
		t.Error("expected UpdatedAt to be set")
	}
}

func TestCategory_GetByID(t *testing.T) {
	svc := newTestService(t)

	cat, err := svc.Create(1, &category.CreateRequest{Name: "Dairy"})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	got, err := svc.GetByID(cat.ID, 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got.ID != cat.ID {
		t.Errorf("expected ID %d, got %d", cat.ID, got.ID)
	}
	if got.Name != "Dairy" {
		t.Errorf("expected name Dairy, got %s", got.Name)
	}
	if got.HouseholdID != 1 {
		t.Errorf("expected household_id 1, got %d", got.HouseholdID)
	}
}

func TestCategory_GetByID_NotFound(t *testing.T) {
	svc := newTestService(t)

	_, err := svc.GetByID(99999, 1)
	if err == nil {
		t.Fatal("expected error for nonexistent category")
	}
	if err.Error() != "category not found" {
		t.Errorf("expected 'category not found', got %v", err)
	}
}

func TestCategory_GetByID_WrongHousehold(t *testing.T) {
	svc := newTestService(t)

	cat, err := svc.Create(1, &category.CreateRequest{Name: "Produce"})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	_, err = svc.GetByID(cat.ID, 2)
	if err == nil {
		t.Fatal("expected error when querying with wrong household")
	}
	if err.Error() != "category not found" {
		t.Errorf("expected 'category not found', got %v", err)
	}
}

func TestCategory_List(t *testing.T) {
	svc := newTestService(t)

	_, _ = svc.Create(1, &category.CreateRequest{Name: "Beverages"})
	_, _ = svc.Create(1, &category.CreateRequest{Name: "Snacks"})
	_, _ = svc.Create(1, &category.CreateRequest{Name: "Apples"})
	_, _ = svc.Create(2, &category.CreateRequest{Name: "Household Items"})

	cats, err := svc.List(1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(cats) != 3 {
		t.Fatalf("expected 3 categories, got %d", len(cats))
	}
	if cats[0].Name != "Beverages" {
		t.Errorf("expected first category Beverages, got %s", cats[0].Name)
	}
	if cats[1].Name != "Snacks" {
		t.Errorf("expected second category Snacks, got %s", cats[1].Name)
	}
	if cats[2].Name != "Apples" {
		t.Errorf("expected third category Apples, got %s", cats[2].Name)
	}
}

func TestCategory_Reorder(t *testing.T) {
	svc := newTestService(t)

	cat1, _ := svc.Create(1, &category.CreateRequest{Name: "First"})
	cat2, _ := svc.Create(1, &category.CreateRequest{Name: "Second"})
	cat3, _ := svc.Create(1, &category.CreateRequest{Name: "Third"})

	err := svc.Reorder(1, []uint{cat3.ID, cat1.ID, cat2.ID})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	cats, err := svc.List(1)
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(cats) != 3 {
		t.Fatalf("expected 3 categories, got %d", len(cats))
	}
	got := []string{cats[0].Name, cats[1].Name, cats[2].Name}
	want := []string{"Third", "First", "Second"}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("index %d: expected %s, got %s", i, want[i], got[i])
		}
	}
}

func TestCategory_Reorder_RequiresCompleteSet(t *testing.T) {
	svc := newTestService(t)

	cat1, _ := svc.Create(1, &category.CreateRequest{Name: "Alpha"})
	_, _ = svc.Create(1, &category.CreateRequest{Name: "Beta"})

	err := svc.Reorder(1, []uint{cat1.ID})
	if err == nil {
		t.Fatal("expected error when omitting a category")
	}
	if err.Error() != "category_ids must contain every category in the household exactly once" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCategory_Reorder_Duplicate(t *testing.T) {
	svc := newTestService(t)

	cat, _ := svc.Create(1, &category.CreateRequest{Name: "Solo"})

	err := svc.Reorder(1, []uint{cat.ID, cat.ID})
	if err == nil {
		t.Fatal("expected error for duplicate ids")
	}
	if err.Error() != "duplicate category ids in category_ids" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCategory_Reorder_WrongHouseholdCategory(t *testing.T) {
	svc := newTestService(t)

	other, _ := svc.Create(1, &category.CreateRequest{Name: "House1"})
	ownA, _ := svc.Create(2, &category.CreateRequest{Name: "House2-A"})
	ownB, _ := svc.Create(2, &category.CreateRequest{Name: "House2-B"})

	err := svc.Reorder(2, []uint{ownB.ID, other.ID})
	if err == nil {
		t.Fatal("expected error when reordering another household's category")
	}
	if err.Error() != "category not found in household" {
		t.Errorf("unexpected error: %v", err)
	}

	// The failed reorder must not have mutated the existing order.
	cats, listErr := svc.List(2)
	if listErr != nil {
		t.Fatalf("list failed: %v", listErr)
	}
	if len(cats) != 2 || cats[0].ID != ownA.ID || cats[1].ID != ownB.ID {
		t.Errorf("household 2 order mutated after failed reorder: %+v", cats)
	}
}

func TestCategory_List_Empty(t *testing.T) {
	svc := newTestService(t)

	cats, err := svc.List(999)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(cats) != 0 {
		t.Errorf("expected 0 categories, got %d", len(cats))
	}
}

func TestCategory_Update(t *testing.T) {
	svc := newTestService(t)

	cat, err := svc.Create(1, &category.CreateRequest{Name: "Old Name"})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	updated, err := svc.Update(cat.ID, 1, &category.UpdateRequest{Name: "New Name"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updated.Name != "New Name" {
		t.Errorf("expected name New Name, got %s", updated.Name)
	}
	if updated.ID != cat.ID {
		t.Errorf("expected ID %d, got %d", cat.ID, updated.ID)
	}
}

func TestCategory_Update_NotFound(t *testing.T) {
	svc := newTestService(t)

	_, err := svc.Update(99999, 1, &category.UpdateRequest{Name: "X"})
	if err == nil {
		t.Fatal("expected error for nonexistent category")
	}
	if err.Error() != "category not found" {
		t.Errorf("expected 'category not found', got %v", err)
	}
}

func TestCategory_Update_WrongHousehold(t *testing.T) {
	svc := newTestService(t)

	cat, err := svc.Create(1, &category.CreateRequest{Name: "Protected"})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	_, err = svc.Update(cat.ID, 2, &category.UpdateRequest{Name: "Hacked"})
	if err == nil {
		t.Fatal("expected error when updating from wrong household")
	}
	if err.Error() != "category not found" {
		t.Errorf("expected 'category not found', got %v", err)
	}
}

func TestCategory_Delete(t *testing.T) {
	svc := newTestService(t)

	cat, err := svc.Create(1, &category.CreateRequest{Name: "To Delete"})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	err = svc.Delete(cat.ID, 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	_, err = svc.GetByID(cat.ID, 1)
	if err == nil {
		t.Fatal("expected error after deletion")
	}
}

func TestCategory_Delete_NotFound(t *testing.T) {
	svc := newTestService(t)

	err := svc.Delete(99999, 1)
	if err == nil {
		t.Fatal("expected error for nonexistent category")
	}
	if err.Error() != "category not found" {
		t.Errorf("expected 'category not found', got %v", err)
	}
}

func TestCategory_Delete_WrongHousehold(t *testing.T) {
	svc := newTestService(t)

	cat, err := svc.Create(1, &category.CreateRequest{Name: "House1Only"})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	err = svc.Delete(cat.ID, 2)
	if err == nil {
		t.Fatal("expected error when deleting from wrong household")
	}
	if err.Error() != "category not found" {
		t.Errorf("expected 'category not found', got %v", err)
	}

	got, err := svc.GetByID(cat.ID, 1)
	if err != nil {
		t.Fatalf("category should still exist: %v", err)
	}
	if got.Name != "House1Only" {
		t.Errorf("expected name House1Only, got %s", got.Name)
	}
}

func TestCategory_Create_DuplicateName_SameHousehold(t *testing.T) {
	svc := newTestService(t)

	_, err := svc.Create(1, &category.CreateRequest{Name: "Drinks"})
	if err != nil {
		t.Fatalf("first create failed: %v", err)
	}

	_, err = svc.Create(1, &category.CreateRequest{Name: "Drinks"})
	if err == nil {
		t.Fatal("expected error for duplicate name in same household")
	}
}

func TestCategory_Create_SameName_DifferentHouseholds(t *testing.T) {
	svc := newTestService(t)

	cat1, err := svc.Create(1, &category.CreateRequest{Name: "Fruit"})
	if err != nil {
		t.Fatalf("first create failed: %v", err)
	}

	cat2, err := svc.Create(2, &category.CreateRequest{Name: "Fruit"})
	if err != nil {
		t.Fatalf("second create failed: %v", err)
	}

	if cat1.HouseholdID == cat2.HouseholdID {
		t.Error("expected different household IDs")
	}
	if cat1.Name != cat2.Name {
		t.Error("expected same name across households")
	}

	got1, err := svc.GetByID(cat1.ID, 1)
	if err != nil {
		t.Fatalf("get household 1 category failed: %v", err)
	}
	got2, err := svc.GetByID(cat2.ID, 2)
	if err != nil {
		t.Fatalf("get household 2 category failed: %v", err)
	}
	if got1.ID == got2.ID {
		t.Error("expected different IDs")
	}
}

func TestCategory_Update_DuplicateName(t *testing.T) {
	svc := newTestService(t)

	_, _ = svc.Create(1, &category.CreateRequest{Name: "Existing"})
	cat2, err := svc.Create(1, &category.CreateRequest{Name: "Rename Me"})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	_, err = svc.Update(cat2.ID, 1, &category.UpdateRequest{Name: "Existing"})
	if err == nil {
		t.Fatal("expected error when updating to duplicate name in same household")
	}
}

func TestCategory_List_IsolationBetweenHouseholds(t *testing.T) {
	svc := newTestService(t)

	_, _ = svc.Create(1, &category.CreateRequest{Name: "Cat A"})
	_, _ = svc.Create(1, &category.CreateRequest{Name: "Cat B"})
	_, _ = svc.Create(2, &category.CreateRequest{Name: "Cat C"})

	h1, err := svc.List(1)
	if err != nil {
		t.Fatalf("list household 1 failed: %v", err)
	}
	if len(h1) != 2 {
		t.Errorf("expected 2 categories for household 1, got %d", len(h1))
	}

	h2, err := svc.List(2)
	if err != nil {
		t.Fatalf("list household 2 failed: %v", err)
	}
	if len(h2) != 1 {
		t.Errorf("expected 1 category for household 2, got %d", len(h2))
	}
	if h2[0].Name != "Cat C" {
		t.Errorf("expected Cat C, got %s", h2[0].Name)
	}
}

func TestCategory_Update_NoNameChange(t *testing.T) {
	svc := newTestService(t)

	cat, _ := svc.Create(1, &category.CreateRequest{Name: "Keep"})

	updated, err := svc.Update(cat.ID, 1, &category.UpdateRequest{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updated.Name != "Keep" {
		t.Errorf("expected name to remain Keep, got %s", updated.Name)
	}
}
