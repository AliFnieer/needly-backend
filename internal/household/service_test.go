package household_test

import (
	"testing"
	"time"

	"github.com/AliFnieer/needly-backend/internal/history"
	"github.com/AliFnieer/needly-backend/internal/household"
	"github.com/AliFnieer/needly-backend/internal/shoppingitem"
	"github.com/AliFnieer/needly-backend/internal/shoppinglist"
	"github.com/AliFnieer/needly-backend/internal/testutil"
)

func TestCreate(t *testing.T) {
	db := testutil.SetupTestDB(t)
	user := testutil.SeedUser(t, db, "owner@test.com", "password123")
	svc := household.NewService(db, nil, nil)

	h, err := svc.Create(user.ID, &household.CreateRequest{Name: "My Household"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h.Name != "My Household" {
		t.Errorf("expected name 'My Household', got %q", h.Name)
	}
	if h.OwnerID != user.ID {
		t.Errorf("expected owner_id %d, got %d", user.ID, h.OwnerID)
	}

	member, err := svc.GetByID(h.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(member.Members) != 1 {
		t.Fatalf("expected 1 member (owner), got %d", len(member.Members))
	}
	if member.Members[0].UserID != user.ID {
		t.Errorf("expected member user_id %d, got %d", user.ID, member.Members[0].UserID)
	}
	if member.Members[0].Role != household.RoleOwner {
		t.Errorf("expected role 'owner', got %q", member.Members[0].Role)
	}
}

func TestGetByID(t *testing.T) {
	db := testutil.SetupTestDB(t)
	user := testutil.SeedUser(t, db, "get@test.com", "password123")
	svc := household.NewService(db, nil, nil)

	h, _ := svc.Create(user.ID, &household.CreateRequest{Name: "Get Test"})

	got, err := svc.GetByID(h.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != "Get Test" {
		t.Errorf("expected name 'Get Test', got %q", got.Name)
	}

	_, err = svc.GetByID(99999)
	if err == nil {
		t.Error("expected error for non-existent household")
	}
}

func TestListByUser(t *testing.T) {
	db := testutil.SetupTestDB(t)
	user1 := testutil.SeedUser(t, db, "user1@test.com", "password123")
	user2 := testutil.SeedUser(t, db, "user2@test.com", "password123")
	svc := household.NewService(db, nil, nil)

	h1, _ := svc.Create(user1.ID, &household.CreateRequest{Name: "User1 Household"})
	_, _ = svc.Create(user2.ID, &household.CreateRequest{Name: "User2 Household"})

	_, _ = svc.AddMember(h1.ID, user1.ID, &household.AddMemberRequest{UserID: user2.ID})

	lists, err := svc.ListByUser(user1.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lists) != 1 {
		t.Fatalf("expected 1 household for user1, got %d", len(lists))
	}
	if lists[0].Name != "User1 Household" {
		t.Errorf("expected 'User1 Household', got %q", lists[0].Name)
	}

	lists2, err := svc.ListByUser(user2.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lists2) != 2 {
		t.Fatalf("expected 2 households for user2, got %d", len(lists2))
	}
}

func TestUpdate(t *testing.T) {
	db := testutil.SetupTestDB(t)
	user := testutil.SeedUser(t, db, "update@test.com", "password123")
	svc := household.NewService(db, nil, nil)

	h, _ := svc.Create(user.ID, &household.CreateRequest{Name: "Old Name"})

	updated, err := svc.Update(h.ID, user.ID, &household.UpdateRequest{Name: "New Name"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Name != "New Name" {
		t.Errorf("expected name 'New Name', got %q", updated.Name)
	}

	fetched, _ := svc.GetByID(h.ID)
	if fetched.Name != "New Name" {
		t.Errorf("expected persisted name 'New Name', got %q", fetched.Name)
	}
}

func TestUpdateNonOwnerFails(t *testing.T) {
	db := testutil.SetupTestDB(t)
	owner := testutil.SeedUser(t, db, "owner@test.com", "password123")
	member := testutil.SeedUser(t, db, "member@test.com", "password123")
	svc := household.NewService(db, nil, nil)

	h, _ := svc.Create(owner.ID, &household.CreateRequest{Name: "Household"})
	_, _ = svc.AddMember(h.ID, owner.ID, &household.AddMemberRequest{UserID: member.ID})

	_, err := svc.Update(h.ID, member.ID, &household.UpdateRequest{Name: "Hacked"})
	if err == nil {
		t.Error("expected error when non-owner tries to update")
	}
}

func TestDelete(t *testing.T) {
	db := testutil.SetupTestDB(t)
	user := testutil.SeedUser(t, db, "del@test.com", "password123")
	svc := household.NewService(db, nil, nil)

	h, _ := svc.Create(user.ID, &household.CreateRequest{Name: "To Delete"})

	err := svc.Delete(h.ID, user.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = svc.GetByID(h.ID)
	if err == nil {
		t.Error("expected error when fetching deleted household")
	}
}

func TestDeleteNonOwnerFails(t *testing.T) {
	db := testutil.SetupTestDB(t)
	owner := testutil.SeedUser(t, db, "owner@test.com", "password123")
	member := testutil.SeedUser(t, db, "member@test.com", "password123")
	svc := household.NewService(db, nil, nil)

	h, _ := svc.Create(owner.ID, &household.CreateRequest{Name: "Household"})
	_, _ = svc.AddMember(h.ID, owner.ID, &household.AddMemberRequest{UserID: member.ID})

	err := svc.Delete(h.ID, member.ID)
	if err == nil {
		t.Error("expected error when non-owner tries to delete")
	}
}

func TestAddMember(t *testing.T) {
	db := testutil.SetupTestDB(t)
	owner := testutil.SeedUser(t, db, "owner@test.com", "password123")
	newMember := testutil.SeedUser(t, db, "new@test.com", "password123")
	svc := household.NewService(db, nil, nil)

	h, _ := svc.Create(owner.ID, &household.CreateRequest{Name: "Household"})

	m, err := svc.AddMember(h.ID, owner.ID, &household.AddMemberRequest{UserID: newMember.ID, Role: household.RoleMember})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Role != household.RoleMember {
		t.Errorf("expected role 'member', got %q", m.Role)
	}
	if m.UserID != newMember.ID {
		t.Errorf("expected user_id %d, got %d", newMember.ID, m.UserID)
	}

	hh, _ := svc.GetByID(h.ID)
	if len(hh.Members) != 2 {
		t.Errorf("expected 2 members, got %d", len(hh.Members))
	}
}

func TestAddMemberNonOwnerFails(t *testing.T) {
	db := testutil.SetupTestDB(t)
	owner := testutil.SeedUser(t, db, "owner@test.com", "password123")
	member := testutil.SeedUser(t, db, "member@test.com", "password123")
	third := testutil.SeedUser(t, db, "third@test.com", "password123")
	svc := household.NewService(db, nil, nil)

	h, _ := svc.Create(owner.ID, &household.CreateRequest{Name: "Household"})
	_, _ = svc.AddMember(h.ID, owner.ID, &household.AddMemberRequest{UserID: member.ID})

	_, err := svc.AddMember(h.ID, member.ID, &household.AddMemberRequest{UserID: third.ID})
	if err == nil {
		t.Error("expected error when non-owner tries to add member")
	}
}

func TestRemoveMember(t *testing.T) {
	db := testutil.SetupTestDB(t)
	owner := testutil.SeedUser(t, db, "owner@test.com", "password123")
	member := testutil.SeedUser(t, db, "member@test.com", "password123")
	svc := household.NewService(db, nil, nil)

	h, _ := svc.Create(owner.ID, &household.CreateRequest{Name: "Household"})
	_, _ = svc.AddMember(h.ID, owner.ID, &household.AddMemberRequest{UserID: member.ID})

	err := svc.RemoveMember(h.ID, owner.ID, member.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	hh, _ := svc.GetByID(h.ID)
	if len(hh.Members) != 1 {
		t.Errorf("expected 1 member after removal, got %d", len(hh.Members))
	}
}

func TestRemoveOwnerFails(t *testing.T) {
	db := testutil.SetupTestDB(t)
	owner := testutil.SeedUser(t, db, "owner@test.com", "password123")
	svc := household.NewService(db, nil, nil)

	h, _ := svc.Create(owner.ID, &household.CreateRequest{Name: "Household"})

	err := svc.RemoveMember(h.ID, owner.ID, owner.ID)
	if err == nil {
		t.Error("expected error when trying to remove owner")
	}
}

func TestRemoveNonMemberFails(t *testing.T) {
	db := testutil.SetupTestDB(t)
	owner := testutil.SeedUser(t, db, "owner@test.com", "password123")
	outsider := testutil.SeedUser(t, db, "outsider@test.com", "password123")
	svc := household.NewService(db, nil, nil)

	h, _ := svc.Create(owner.ID, &household.CreateRequest{Name: "Household"})

	err := svc.RemoveMember(h.ID, owner.ID, outsider.ID)
	if err == nil {
		t.Error("expected error when removing non-member")
	}
}

func TestRemoveMemberNonOwnerFails(t *testing.T) {
	db := testutil.SetupTestDB(t)
	owner := testutil.SeedUser(t, db, "owner@test.com", "password123")
	member := testutil.SeedUser(t, db, "member@test.com", "password123")
	third := testutil.SeedUser(t, db, "third@test.com", "password123")
	svc := household.NewService(db, nil, nil)

	h, _ := svc.Create(owner.ID, &household.CreateRequest{Name: "Household"})
	_, _ = svc.AddMember(h.ID, owner.ID, &household.AddMemberRequest{UserID: member.ID})
	_, _ = svc.AddMember(h.ID, owner.ID, &household.AddMemberRequest{UserID: third.ID})

	err := svc.RemoveMember(h.ID, member.ID, third.ID)
	if err == nil {
		t.Error("expected error when non-owner tries to remove member")
	}
}

func TestCheckMember(t *testing.T) {
	db := testutil.SetupTestDB(t)
	owner := testutil.SeedUser(t, db, "owner@test.com", "password123")
	member := testutil.SeedUser(t, db, "member@test.com", "password123")
	outsider := testutil.SeedUser(t, db, "outsider@test.com", "password123")
	svc := household.NewService(db, nil, nil)

	h, _ := svc.Create(owner.ID, &household.CreateRequest{Name: "Household"})
	_, _ = svc.AddMember(h.ID, owner.ID, &household.AddMemberRequest{UserID: member.ID})

	if !svc.CheckMember(h.ID, owner.ID) {
		t.Error("owner should be a member")
	}
	if !svc.CheckMember(h.ID, member.ID) {
		t.Error("member should be a member")
	}
	if svc.CheckMember(h.ID, outsider.ID) {
		t.Error("outsider should not be a member")
	}
}

func TestHouseholdIDForList(t *testing.T) {
	db := testutil.SetupTestDB(t)
	user := testutil.SeedUser(t, db, "owner@test.com", "password123")
	svc := household.NewService(db, nil, nil)

	h, _ := svc.Create(user.ID, &household.CreateRequest{Name: "Household"})

	list := shoppinglist.ShoppingList{
		HouseholdID: h.ID,
		Name:        "Test List",
		CreatedBy:   user.ID,
	}
	db.Create(&list)

	resolvedID, err := svc.HouseholdIDForList(list.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolvedID != h.ID {
		t.Errorf("expected household_id %d, got %d", h.ID, resolvedID)
	}
}

func TestHouseholdIDForItem(t *testing.T) {
	db := testutil.SetupTestDB(t)
	user := testutil.SeedUser(t, db, "owner@test.com", "password123")
	svc := household.NewService(db, nil, nil)

	h, _ := svc.Create(user.ID, &household.CreateRequest{Name: "Household"})

	list := shoppinglist.ShoppingList{
		HouseholdID: h.ID,
		Name:        "Test List",
		CreatedBy:   user.ID,
	}
	db.Create(&list)

	item := shoppingitem.ShoppingItem{
		ListID:    list.ID,
		Name:      "Milk",
		Quantity:  2,
		CreatedBy: user.ID,
	}
	db.Create(&item)

	resolvedID, err := svc.HouseholdIDForItem(item.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolvedID != h.ID {
		t.Errorf("expected household_id %d, got %d", h.ID, resolvedID)
	}
}

func TestHouseholdIDForHistory(t *testing.T) {
	db := testutil.SetupTestDB(t)
	user := testutil.SeedUser(t, db, "owner@test.com", "password123")
	svc := household.NewService(db, nil, nil)

	h, _ := svc.Create(user.ID, &household.CreateRequest{Name: "Household"})

	list := shoppinglist.ShoppingList{
		HouseholdID: h.ID,
		Name:        "Test List",
		CreatedBy:   user.ID,
	}
	db.Create(&list)

	hist := history.ShoppingHistory{
		ListID:      list.ID,
		Name:        "Milk",
		Quantity:    1,
		CompletedBy: user.ID,
		CompletedAt: time.Now(),
	}
	db.Create(&hist)

	resolvedID, err := svc.HouseholdIDForHistory(hist.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolvedID != h.ID {
		t.Errorf("expected household_id %d, got %d", h.ID, resolvedID)
	}
}
