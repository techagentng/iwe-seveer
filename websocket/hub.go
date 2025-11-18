package websocket

import (
	"log"
	"sync"

	"github.com/google/uuid"
)

// Hub maintains the set of active clients and broadcasts messages to them
type Hub struct {
	// Registered clients organized by user ID
	clients map[uuid.UUID]map[*Client]bool

	// Register requests from clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Broadcast message to specific user
	broadcast chan *Message

	// Mutex for thread-safe operations
	mu sync.RWMutex
}

// Message represents a message to be sent to a specific user
type Message struct {
	UserID  uuid.UUID
	Payload []byte
}

// NewHub creates a new Hub instance
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[uuid.UUID]map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *Message, 256),
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	log.Println("🔌 WebSocket Hub started")
	
	for {
		select {
		case client := <-h.register:
			h.registerClient(client)

		case client := <-h.unregister:
			h.unregisterClient(client)

		case message := <-h.broadcast:
			h.broadcastToUser(message)
		}
	}
}

// registerClient adds a new client to the hub
func (h *Hub) registerClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.clients[client.UserID]; !ok {
		h.clients[client.UserID] = make(map[*Client]bool)
	}
	h.clients[client.UserID][client] = true

	log.Printf("✅ WebSocket client connected: user=%s, total_clients=%d", 
		client.UserID, h.getTotalClients())
}

// unregisterClient removes a client from the hub
func (h *Hub) unregisterClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if clients, ok := h.clients[client.UserID]; ok {
		if _, ok := clients[client]; ok {
			delete(clients, client)
			close(client.send)

			// Remove user entry if no more clients
			if len(clients) == 0 {
				delete(h.clients, client.UserID)
			}

			log.Printf("❌ WebSocket client disconnected: user=%s, total_clients=%d", 
				client.UserID, h.getTotalClients())
		}
	}
}

// broadcastToUser sends a message to all connections of a specific user
func (h *Hub) broadcastToUser(message *Message) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if clients, ok := h.clients[message.UserID]; ok {
		for client := range clients {
			select {
			case client.send <- message.Payload:
				// Message sent successfully
			default:
				// Client's send channel is full, close it
				close(client.send)
				delete(clients, client)
				log.Printf("⚠️ Client send buffer full, disconnecting: user=%s", message.UserID)
			}
		}
	}
}

// BroadcastToUser queues a message to be sent to a specific user
func (h *Hub) BroadcastToUser(userID uuid.UUID, payload []byte) {
	h.broadcast <- &Message{
		UserID:  userID,
		Payload: payload,
	}
}

// GetConnectedUsers returns the number of users currently connected
func (h *Hub) GetConnectedUsers() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// GetUserConnectionCount returns the number of connections for a specific user
func (h *Hub) GetUserConnectionCount(userID uuid.UUID) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if clients, ok := h.clients[userID]; ok {
		return len(clients)
	}
	return 0
}

// IsUserConnected checks if a user has any active connections
func (h *Hub) IsUserConnected(userID uuid.UUID) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	clients, ok := h.clients[userID]
	return ok && len(clients) > 0
}

// getTotalClients returns total number of connected clients (not thread-safe, call with lock)
func (h *Hub) getTotalClients() int {
	total := 0
	for _, clients := range h.clients {
		total += len(clients)
	}
	return total
}

// DisconnectUser forcefully disconnects all connections for a user
func (h *Hub) DisconnectUser(userID uuid.UUID) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if clients, ok := h.clients[userID]; ok {
		for client := range clients {
			close(client.send)
			delete(clients, client)
		}
		delete(h.clients, userID)
		log.Printf("🔌 Disconnected all connections for user: %s", userID)
	}
}
