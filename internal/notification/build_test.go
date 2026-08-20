package notification

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildNotification_AllFieldsPopulated(t *testing.T) {
	n := BuildNotification(
		NotificationTypeItemCreated,
		"New item",
		"Item was added",
		10, 20, 30, 40,
	)

	assert.NotNil(t, n)
	assert.Equal(t, NotificationTypeItemCreated, n.Type)
	assert.Equal(t, "New item", n.Title)
	assert.Equal(t, "Item was added", n.Body)
	assert.Equal(t, uint(10), n.HouseholdID)
	assert.Equal(t, uint(20), n.ListID)
	assert.Equal(t, uint(30), n.ItemID)
	assert.Equal(t, uint(40), n.ActorID)
	assert.False(t, n.CreatedAt.IsZero())
}
