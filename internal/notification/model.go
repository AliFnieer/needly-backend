package notification

import "time"

// NotificationType identifies the kind of change that triggered a notification.
type NotificationType string

const (
	// NotificationTypeHouseholdCreated is sent when a household is created.
	NotificationTypeHouseholdCreated NotificationType = "household.created"
	// NotificationTypeHouseholdUpdated is sent when a household is updated.
	NotificationTypeHouseholdUpdated NotificationType = "household.updated"
	// NotificationTypeHouseholdDeleted is sent when a household is deleted.
	NotificationTypeHouseholdDeleted NotificationType = "household.deleted"
	// NotificationTypeMemberAdded is sent when a member is added to a household.
	NotificationTypeMemberAdded NotificationType = "household.member_added"
	// NotificationTypeMemberRemoved is sent when a member is removed from a household.
	NotificationTypeMemberRemoved NotificationType = "household.member_removed"

	// NotificationTypeListCreated is sent when a shopping list is created.
	NotificationTypeListCreated NotificationType = "list.created"
	// NotificationTypeListUpdated is sent when a shopping list is updated.
	NotificationTypeListUpdated NotificationType = "list.updated"
	// NotificationTypeListDeleted is sent when a shopping list is deleted.
	NotificationTypeListDeleted NotificationType = "list.deleted"

	// NotificationTypeItemCreated is sent when a shopping item is created.
	NotificationTypeItemCreated NotificationType = "item.created"
	// NotificationTypeItemUpdated is sent when a shopping item is updated.
	NotificationTypeItemUpdated NotificationType = "item.updated"
	// NotificationTypeItemCompleted is sent when a shopping item is marked completed.
	NotificationTypeItemCompleted NotificationType = "item.completed"
	// NotificationTypeItemDeleted is sent when a shopping item is deleted.
	NotificationTypeItemDeleted NotificationType = "item.deleted"
	// NotificationTypeItemReAdded is sent when an item is re-added from history.
	NotificationTypeItemReAdded NotificationType = "item.radded"
	// NotificationTypeItemRecurred is sent when a recurring item becomes due again.
	NotificationTypeItemRecurred NotificationType = "item.recurred"
)

// Notification is a push notification broadcast to household members.
type Notification struct {
	Type        NotificationType `json:"type"`
	Title       string           `json:"title"`
	Body        string           `json:"body"`
	HouseholdID uint             `json:"household_id"`
	ListID      uint             `json:"list_id,omitempty"`
	ItemID      uint             `json:"item_id,omitempty"`
	ActorID     uint             `json:"actor_id,omitempty"`
	CreatedAt   time.Time        `json:"created_at"`
}

// BuildNotification creates a notification payload.
func BuildNotification(nt NotificationType, title, body string, householdID, listID, itemID, actorID uint) *Notification {
	return &Notification{
		Type:        nt,
		Title:       title,
		Body:        body,
		HouseholdID: householdID,
		ListID:      listID,
		ItemID:      itemID,
		ActorID:     actorID,
		CreatedAt:   time.Now().UTC(),
	}
}
