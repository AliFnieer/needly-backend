package websocket

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// wsEventsChannel is the Redis Pub/Sub channel used to distribute
	// WebSocket events across all API instances.
	wsEventsChannel = "ws:events"

	// pubSubReconnectDelay is the base delay between subscription reconnects.
	pubSubReconnectDelay = 2 * time.Second
)

// Message types published over Redis Pub/Sub.
const (
	MessageTypeHousehold = "household"
	MessageTypeClient    = "client"
)

// PubSubMessage is the envelope published to Redis for cross-instance
// WebSocket event distribution.
type PubSubMessage struct {
	Type        string `json:"type"`
	Origin      string `json:"origin"`
	HouseholdID uint   `json:"household_id,omitempty"`
	ClientID    string `json:"client_id,omitempty"`
	Message     []byte `json:"message"`
}

// PubSub distributes WebSocket events across multiple API instances
// using Redis Pub/Sub.
type PubSub struct {
	client *redis.Client
}

// NewPubSub creates a new Redis-backed Pub/Sub distributor.
func NewPubSub(client *redis.Client) *PubSub {
	return &PubSub{client: client}
}

// PublishHousehold publishes a household broadcast to all API instances.
func (p *PubSub) PublishHousehold(ctx context.Context, origin string, householdID uint, message []byte) error {
	return p.publish(ctx, PubSubMessage{
		Type:        MessageTypeHousehold,
		Origin:      origin,
		HouseholdID: householdID,
		Message:     message,
	})
}

// PublishClient publishes a client-targeted message to all API instances.
func (p *PubSub) PublishClient(ctx context.Context, origin, clientID string, message []byte) error {
	return p.publish(ctx, PubSubMessage{
		Type:     MessageTypeClient,
		Origin:   origin,
		ClientID: clientID,
		Message:  message,
	})
}

func (p *PubSub) publish(ctx context.Context, msg PubSubMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return p.client.Publish(ctx, wsEventsChannel, data).Err()
}

// Subscribe subscribes to the WebSocket events channel and invokes handler
// for every received message. It blocks until ctx is cancelled and
// automatically reconnects on transient errors.
func (p *PubSub) Subscribe(ctx context.Context, handler func(PubSubMessage)) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			sub := p.client.Subscribe(ctx, wsEventsChannel)
			ch := sub.Channel()

			slog.Info("websocket pubsub subscribed", "channel", wsEventsChannel)

			for {
				select {
				case <-ctx.Done():
					_ = sub.Close()
					return
				case msg, ok := <-ch:
					if !ok {
						// Channel closed, reconnect
						goto reconnect
					}
					var envelope PubSubMessage
					if err := json.Unmarshal([]byte(msg.Payload), &envelope); err != nil {
						slog.Error("websocket pubsub unmarshal failed", "error", err)
						continue
					}
					handler(envelope)
				}
			}

		reconnect:
			_ = sub.Close()
			slog.Warn("websocket pubsub subscription lost, reconnecting", "delay", pubSubReconnectDelay)
			select {
			case <-ctx.Done():
				return
			case <-time.After(pubSubReconnectDelay):
			}
		}
	}()
}