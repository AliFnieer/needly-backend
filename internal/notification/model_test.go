package notification

import (
	"encoding/json"
	"testing"
	"time"
)

// TestBuildNotification verifies BuildNotification creates a complete payload.
func TestBuildNotification(t *testing.T) {
	n := BuildNotification(NotificationTypeListCreated, "New list", "A list was created", 7, 3, 0, 42)

	if n.Type != NotificationTypeListCreated {
		t.Fatalf("expected type %q, got %q", NotificationTypeListCreated, n.Type)
	}
	if n.Title != "New list" {
		t.Fatalf("expected title %q, got %q", "New list", n.Title)
	}
	if n.HouseholdID != 7 {
		t.Fatalf("expected household id 7, got %d", n.HouseholdID)
	}
	if n.ListID != 3 {
		t.Fatalf("expected list id 3, got %d", n.ListID)
	}
	if n.ItemID != 0 {
		t.Fatalf("expected item id 0, got %d", n.ItemID)
	}
	if n.ActorID != 42 {
		t.Fatalf("expected actor id 42, got %d", n.ActorID)
	}
	if n.CreatedAt.IsZero() {
		t.Fatal("expected a non-zero CreatedAt")
	}
}

// TestMarshalNotification verifies the WebSocket envelope marshaling.
func TestMarshalNotification(t *testing.T) {
	n := &Notification{
		Type:        NotificationTypeItemCompleted,
		Title:       "Item completed",
		Body:        "Milk was completed",
		HouseholdID: 5,
		ListID:      2,
		ItemID:      9,
		ActorID:     1,
		CreatedAt:   time.Date(2026, 8, 19, 9, 0, 0, 0, time.UTC),
	}

	data, err := marshalNotification(n)
	if err != nil {
		t.Fatalf("failed to marshal notification: %v", err)
	}

	var envelope map[string]interface{}
	if err := json.Unmarshal(data, &envelope); err != nil {
		t.Fatalf("failed to unmarshal notification envelope: %v", err)
	}

	if envelope["type"] != "item.completed" {
		t.Fatalf("expected type item.completed, got %v", envelope["type"])
	}
	if envelope["household_id"].(float64) != 5 {
		t.Fatalf("expected household_id 5, got %v", envelope["household_id"])
	}
	if envelope["title"] != "Item completed" {
		t.Fatalf("expected title %q, got %v", "Item completed", envelope["title"])
	}
}
