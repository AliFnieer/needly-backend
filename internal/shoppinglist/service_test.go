package shoppinglist_test

import (
	"context"
	"testing"

	"github.com/AliFnieer/needly-backend/internal/household"
	"github.com/AliFnieer/needly-backend/internal/shoppinglist"
	"github.com/AliFnieer/needly-backend/internal/testutil"
)

func TestCreate(t *testing.T) {
	db := testutil.SetupTestDB(t)
	user := testutil.SeedUser(t, db, "owner@test.com", "password123")
	hhSvc := household.NewService(db, nil, nil)
	svc := shoppinglist.NewService(db, nil, nil)
	ctx := context.Background()

	hh, err := hhSvc.Create(user.ID, &household.CreateRequest{Name: "Household"})
	if err != nil {
		t.Fatalf("failed to create household: %v", err)
	}

	list, err := svc.Create(ctx, hh.ID, user.ID, &shoppinglist.CreateRequest{Name: "Groceries"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if list.Name != "Groceries" {
		t.Errorf("expected name 'Groceries', got %q", list.Name)
	}
	if list.HouseholdID != hh.ID {
		t.Errorf("expected household_id %d, got %d", hh.ID, list.HouseholdID)
	}
	if list.CreatedBy != user.ID {
		t.Errorf("expected created_by %d, got %d", user.ID, list.CreatedBy)
	}
}

func TestGetByID(t *testing.T) {
	db := testutil.SetupTestDB(t)
	user := testutil.SeedUser(t, db, "owner@test.com", "password123")
	hhSvc := household.NewService(db, nil, nil)
	svc := shoppinglist.NewService(db, nil, nil)
	ctx := context.Background()

	hh, _ := hhSvc.Create(user.ID, &household.CreateRequest{Name: "Household"})
	list, _ := svc.Create(ctx, hh.ID, user.ID, &shoppinglist.CreateRequest{Name: "My List"})

	got, err := svc.GetByID(ctx, list.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != "My List" {
		t.Errorf("expected name 'My List', got %q", got.Name)
	}

	_, err = svc.GetByID(ctx, 99999)
	if err == nil {
		t.Error("expected error for non-existent list")
	}
}

func TestListByHouseholdID(t *testing.T) {
	db := testutil.SetupTestDB(t)
	user := testutil.SeedUser(t, db, "owner@test.com", "password123")
	hhSvc := household.NewService(db, nil, nil)
	svc := shoppinglist.NewService(db, nil, nil)
	ctx := context.Background()

	hh, _ := hhSvc.Create(user.ID, &household.CreateRequest{Name: "Household"})

	svc.Create(ctx, hh.ID, user.ID, &shoppinglist.CreateRequest{Name: "First"})
	svc.Create(ctx, hh.ID, user.ID, &shoppinglist.CreateRequest{Name: "Second"})
	svc.Create(ctx, hh.ID, user.ID, &shoppinglist.CreateRequest{Name: "Third"})

	lists, err := svc.ListByHouseholdID(ctx, hh.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lists) != 3 {
		t.Fatalf("expected 3 lists, got %d", len(lists))
	}
	if lists[0].Name != "Third" {
		t.Errorf("expected DESC ordering, first item should be 'Third', got %q", lists[0].Name)
	}
	if lists[2].Name != "First" {
		t.Errorf("expected DESC ordering, last item should be 'First', got %q", lists[2].Name)
	}
}

func TestUpdate(t *testing.T) {
	db := testutil.SetupTestDB(t)
	user := testutil.SeedUser(t, db, "owner@test.com", "password123")
	hhSvc := household.NewService(db, nil, nil)
	svc := shoppinglist.NewService(db, nil, nil)
	ctx := context.Background()

	hh, _ := hhSvc.Create(user.ID, &household.CreateRequest{Name: "Household"})
	list, _ := svc.Create(ctx, hh.ID, user.ID, &shoppinglist.CreateRequest{Name: "Old Name"})

	updated, err := svc.Update(ctx, list.ID, &shoppinglist.UpdateRequest{Name: "New Name"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Name != "New Name" {
		t.Errorf("expected name 'New Name', got %q", updated.Name)
	}

	fetched, _ := svc.GetByID(ctx, list.ID)
	if fetched.Name != "New Name" {
		t.Errorf("expected persisted name 'New Name', got %q", fetched.Name)
	}
}

func TestUpdateNotFound(t *testing.T) {
	db := testutil.SetupTestDB(t)
	svc := shoppinglist.NewService(db, nil, nil)
	ctx := context.Background()

	_, err := svc.Update(ctx, 99999, &shoppinglist.UpdateRequest{Name: "Nope"})
	if err == nil {
		t.Error("expected error for non-existent list")
	}
}

func TestDelete(t *testing.T) {
	db := testutil.SetupTestDB(t)
	user := testutil.SeedUser(t, db, "owner@test.com", "password123")
	hhSvc := household.NewService(db, nil, nil)
	svc := shoppinglist.NewService(db, nil, nil)
	ctx := context.Background()

	hh, _ := hhSvc.Create(user.ID, &household.CreateRequest{Name: "Household"})
	list, _ := svc.Create(ctx, hh.ID, user.ID, &shoppinglist.CreateRequest{Name: "To Delete"})

	err := svc.Delete(ctx, list.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = svc.GetByID(ctx, list.ID)
	if err == nil {
		t.Error("expected error when fetching deleted list")
	}
}

func TestDeleteNotFound(t *testing.T) {
	db := testutil.SetupTestDB(t)
	svc := shoppinglist.NewService(db, nil, nil)
	ctx := context.Background()

	err := svc.Delete(ctx, 99999)
	if err == nil {
		t.Error("expected error for non-existent list")
	}
}

func TestListByHouseholdIDEmpty(t *testing.T) {
	db := testutil.SetupTestDB(t)
	user := testutil.SeedUser(t, db, "owner@test.com", "password123")
	hhSvc := household.NewService(db, nil, nil)
	svc := shoppinglist.NewService(db, nil, nil)
	ctx := context.Background()

	hh, _ := hhSvc.Create(user.ID, &household.CreateRequest{Name: "Household"})

	lists, err := svc.ListByHouseholdID(ctx, hh.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lists) != 0 {
		t.Errorf("expected 0 lists, got %d", len(lists))
	}
}