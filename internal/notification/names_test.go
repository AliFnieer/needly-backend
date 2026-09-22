package notification

import (
	"encoding/json"
	"testing"
)

func TestNotification_NameFieldsRoundTripHistory(t *testing.T) {
	n := BuildNotification(NotificationTypeItemCreated, "New shopping item", "added", 1, 2, 3, 4)
	n.WithNames(NameContext{ItemName: "Milk", ListName: "Groceries", HouseholdName: "Family"})

	data, err := json.Marshal(n)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var out Notification
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if out.ItemName != "Milk" || out.ListName != "Groceries" || out.HouseholdName != "Family" {
		t.Errorf("name fields lost after round trip: got %+v", out)
	}
}

func TestNotification_MarshalNotificationIncludesNameFields(t *testing.T) {
	n := BuildNotification(NotificationTypeListDeleted, "Shopping list deleted", "deleted", 1, 2, 0, 0)
	n.WithNames(NameContext{ListName: "Weekly"})

	data, err := marshalNotification(n)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var envelope map[string]interface{}
	if err := json.Unmarshal(data, &envelope); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if envelope["list_name"] != "Weekly" {
		t.Errorf("expected list_name in envelope, got %v", envelope["list_name"])
	}
}