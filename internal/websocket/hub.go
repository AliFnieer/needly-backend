package websocket

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"sync"

	"github.com/redis/go-redis/v9"
)

// Client represents a connected WebSocket client.
type Client struct {
	ID          string
	HouseholdID uint
	UserID      uint
	Send        chan []byte
}

// Hub manages active WebSocket clients and broadcasts messages.
// It uses Redis Pub/Sub to distribute events across multiple API instances.
type Hub struct {
	mu sync.RWMutex

	// clients maps client ID to Client.
	clients map[string]*Client

	// register receives new client connections.
	register chan *Client

	// unregister receives disconnected clients.
	unregister chan *Client

	// broadcast receives messages to send to all clients in a household.
	broadcast chan broadcastMessage

	// pubsub distributes events across API instances via Redis.
	pubsub *PubSub

	// instanceID uniquely identifies this API instance so it can ignore
	// messages it published itself.
	instanceID string

	// ctx is the hub's lifecycle context.
	ctx context.Context

	// cancel cancels the hub's lifecycle context.
	cancel context.CancelFunc
}

// broadcastMessage is an internal message type for household broadcasts.
type broadcastMessage struct {
	HouseholdID uint
	Message     []byte
}

// NewHub creates a new WebSocket hub with Redis Pub/Sub distribution.
// If redisClient is nil, the hub operates in local-only mode.
func NewHub(redisClient *redis.Client) *Hub {
	ctx, cancel := context.WithCancel(context.Background())

	hub := &Hub{
		clients:    make(map[string]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan broadcastMessage),
		instanceID: newInstanceID(),
		ctx:        ctx,
		cancel:     cancel,
	}

	if redisClient != nil {
		hub.pubsub = NewPubSub(redisClient)
	}

	return hub
}

// Run starts the hub's event loop. It should be called as a goroutine.
func (h *Hub) Run() {
	// Subscribe to cross-instance events if Redis Pub/Sub is enabled.
	if h.pubsub != nil {
		h.pubsub.Subscribe(h.ctx, h.handlePubSubMessage)
	}

	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client.ID] = client
			h.mu.Unlock()
			slog.Info("websocket client connected", "id", client.ID)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.ID]; ok {
				delete(h.clients, client.ID)
				close(client.Send)
				slog.Info("websocket client disconnected", "id", client.ID)
			}
			h.mu.Unlock()

		case msg := <-h.broadcast:
			// Deliver locally and publish to Redis for other instances.
			h.deliverToHousehold(msg.HouseholdID, msg.Message)
			if h.pubsub != nil {
				if err := h.pubsub.PublishHousehold(h.ctx, h.instanceID, msg.HouseholdID, msg.Message); err != nil {
					slog.Error("websocket pubsub publish household failed", "error", err)
				}
			}

		case <-h.ctx.Done():
			slog.Info("websocket hub shutting down")
			return
		}
	}
}

// handlePubSubMessage delivers a message received from another API instance
// to locally connected clients.
func (h *Hub) handlePubSubMessage(msg PubSubMessage) {
	// Ignore messages this instance published itself to avoid double delivery.
	if msg.Origin == h.instanceID {
		return
	}

	switch msg.Type {
	case MessageTypeHousehold:
		h.deliverToHousehold(msg.HouseholdID, msg.Message)
	case MessageTypeClient:
		h.deliverToClient(msg.ClientID, msg.Message)
	default:
		slog.Warn("websocket pubsub unknown message type", "type", msg.Type)
	}
}

// deliverToHousehold sends a message to all locally connected clients
// in the given household.
func (h *Hub) deliverToHousehold(householdID uint, message []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, client := range h.clients {
		if client.HouseholdID == householdID {
			select {
			case client.Send <- message:
			default:
				// Slow client, skip to prevent blocking
			}
		}
	}
}

// deliverToClient sends a message to a specific locally connected client.
func (h *Hub) deliverToClient(clientID string, message []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if client, ok := h.clients[clientID]; ok {
		select {
		case client.Send <- message:
		default:
			// Slow client, skip to prevent blocking
		}
	}
}

// Register adds a client to the hub.
func (h *Hub) Register(client *Client) {
	h.register <- client
}

// Unregister removes a client from the hub.
func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

// BroadcastToHousehold sends a message to all clients in a household
// across all API instances via Redis Pub/Sub.
func (h *Hub) BroadcastToHousehold(householdID uint, message []byte) {
	h.broadcast <- broadcastMessage{
		HouseholdID: householdID,
		Message:     message,
	}
}

// BroadcastToClient sends a message to a specific client across all
// API instances via Redis Pub/Sub.
func (h *Hub) BroadcastToClient(clientID string, message []byte) {
	// Deliver locally and publish to Redis for other instances.
	h.deliverToClient(clientID, message)
	if h.pubsub != nil {
		if err := h.pubsub.PublishClient(h.ctx, h.instanceID, clientID, message); err != nil {
			slog.Error("websocket pubsub publish client failed", "error", err)
		}
	}
}

// GetClientCount returns the number of active clients.
func (h *Hub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// Shutdown gracefully stops the hub and its Redis subscription.
func (h *Hub) Shutdown() {
	h.cancel()
}

// newInstanceID generates a random hex string identifying this API instance.
func newInstanceID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		// Fallback to a timestamp-based ID if crypto/rand fails.
		return "instance"
	}
	return hex.EncodeToString(b)
}
