package websocket

import (
	"log"
	"sync"
)

// Client represents a connected WebSocket client.
type Client struct {
	ID         string
	HouseholdID uint
	Send       chan []byte
}

// Hub manages active WebSocket clients and broadcasts messages.
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
}

// broadcastMessage is an internal message type for household broadcasts.
type broadcastMessage struct {
	HouseholdID uint
	Message     []byte
}

// NewHub creates a new WebSocket hub.
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[string]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan broadcastMessage),
	}
}

// Run starts the hub's event loop. It should be called as a goroutine.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client.ID] = client
			h.mu.Unlock()
			log.Printf("websocket client connected: %s", client.ID)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.ID]; ok {
				delete(h.clients, client.ID)
				close(client.Send)
				log.Printf("websocket client disconnected: %s", client.ID)
			}
			h.mu.Unlock()

		case msg := <-h.broadcast:
			h.mu.RLock()
			for _, client := range h.clients {
				if client.HouseholdID == msg.HouseholdID {
					select {
					case client.Send <- msg.Message:
					default:
						// Slow client, skip to prevent blocking
					}
				}
			}
			h.mu.RUnlock()
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

// BroadcastToHousehold sends a message to all clients in a household.
func (h *Hub) BroadcastToHousehold(householdID uint, message []byte) {
	h.broadcast <- broadcastMessage{
		HouseholdID: householdID,
		Message:     message,
	}
}

// BroadcastToClient sends a message to a specific client.
func (h *Hub) BroadcastToClient(clientID string, message []byte) {
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

// GetClientCount returns the number of active clients.
func (h *Hub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}