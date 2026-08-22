package notification

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/AliFnieer/needly-backend/internal/config"
	"github.com/AliFnieer/needly-backend/internal/websocket"
	"github.com/redis/go-redis/v9"
)

const (
	// notificationHistoryKeyPrefix is the Redis key prefix for recent notification history per household.
	notificationHistoryKeyPrefix = "notifications:"

	// notificationHistoryKeySuffix is the suffixed used to build household history keys.
	notificationHistoryKeySuffix = ":history"
)

// Service delivers push notifications to household members via the WebSocket
// hub and persists a short notification history in Redis.
type Service struct {
	hub   *websocket.Hub
	redis *redis.Client
	cfg   *config.NotificationConfig
}

// NewService creates a push notification service.
func NewService(hub *websocket.Hub, redisClient *redis.Client, cfg *config.NotificationConfig) *Service {
	return &Service{
		hub:   hub,
		redis: redisClient,
		cfg:   cfg,
	}
}

// NotifyHousehold delivers a notification to all members of a household.
// It broadcasts the notification over the WebSocket hub (which distributes
// it to all API instances via Redis Pub/Sub) and stores a recent history
// of per-household notifications in Redis.
func (s *Service) NotifyHousehold(ctx context.Context, n *Notification) error {
	if s.cfg == nil || !s.cfg.Enabled {
		return nil
	}

	// Deliver in real time over the WebSocket hub.
	if s.cfg.WebSocketEnabled && s.hub != nil {
		if err := s.broadcastToHousehold(n); err != nil {
			slog.Error("notification broadcast failed", "error", err)
		}
	}

	// Keep a bounded history of notifications per household in Redis.
	if s.redis != nil {
		s.storeHistory(ctx, n)
	}

	return nil
}

// NotifyClient delivers a notification to a specific connected client.
func (s *Service) NotifyClient(ctx context.Context, clientID string, n *Notification) error {
	if s.cfg == nil || !s.cfg.Enabled {
		return nil
	}

	if s.cfg.WebSocketEnabled && s.hub != nil {
		data, err := marshalNotification(n)
		if err != nil {
			return err
		}
		s.hub.BroadcastToClient(clientID, data)
	}

	return nil
}

// HistoryByHousehold returns recently delivered notifications for a household.
func (s *Service) HistoryByHousehold(ctx context.Context, householdID uint) ([]*Notification, error) {
	key := householdHistoryKey(householdID)
	data, err := s.redis.LRange(ctx, key, 0, -1).Result()
	if err != nil {
		if err == redis.Nil {
			return []*Notification{}, nil
		}
		return nil, err
	}

	items := make([]*Notification, 0, len(data))
	for _, raw := range data {
		var n Notification
		if err := json.Unmarshal([]byte(raw), &n); err != nil {
			slog.Error("notification history unmarshal failed", "error", err)
			continue
		}
		items = append(items, &n)
	}
	return items, nil
}

// broadcastToHousehold marshals and dispatches the notification to the hub.
func (s *Service) broadcastToHousehold(n *Notification) error {
	data, err := marshalNotification(n)
	if err != nil {
		return err
	}
	s.hub.BroadcastToHousehold(n.HouseholdID, data)
	return nil
}

// storeHistory appends a notification to per-household history, keeping the
// list bounded by the configured history limit. It uses a Redis list with
// LPUSH + LTRIM so concurrent writes are atomic and never lose events.
func (s *Service) storeHistory(ctx context.Context, n *Notification) {
	key := householdHistoryKey(n.HouseholdID)

	limit := 50
	if s.cfg != nil && s.cfg.HistoryLimit > 0 {
		limit = s.cfg.HistoryLimit
	}

	data, err := json.Marshal(n)
	if err != nil {
		slog.Error("notification history marshal failed", "household_id", n.HouseholdID, "error", err)
		return
	}

	// Atomically prepend the new notification and trim to the limit.
	pipe := s.redis.TxPipeline()
	pipe.LPush(ctx, key, data)
	pipe.LTrim(ctx, key, 0, int64(limit-1))
	pipe.Expire(ctx, key, 24*time.Hour)
	if _, err := pipe.Exec(ctx); err != nil {
		slog.Error("notification history store failed", "household_id", n.HouseholdID, "error", err)
	}
}

// marshalNotification serializes a notification for the WebSocket payload.
func marshalNotification(n *Notification) ([]byte, error) {
	// Build the real-time envelope. The WebSocket clients expect the
	// notification type as a top-level "$type" so they can route events.
	envelope := struct {
		Type        string `json:"type"`
		Title       string `json:"title"`
		Body        string `json:"body"`
		HouseholdID uint   `json:"household_id"`
		ListID      uint   `json:"list_id,omitempty"`
		ItemID      uint   `json:"item_id,omitempty"`
		ActorID     uint   `json:"actor_id,omitempty"`
		CreatedAt   string `json:"created_at"`
	}{
		Type:        string(n.Type),
		Title:       n.Title,
		Body:        n.Body,
		HouseholdID: n.HouseholdID,
		ListID:      n.ListID,
		ItemID:      n.ItemID,
		ActorID:     n.ActorID,
		CreatedAt:   n.CreatedAt.UTC().Format(time.RFC3339),
	}

	return json.Marshal(envelope)
}

// householdHistoryKey builds the Redis key for a household's notification history.
func householdHistoryKey(householdID uint) string {
	return notificationHistoryKeyPrefix + uintToString(householdID) + notificationHistoryKeySuffix
}

// uintToString converts a uint to a string for Redis key construction.
func uintToString(v uint) string {
	if v == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	return string(buf[i:])
}
