package websocket

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// TODO: In production, validate origin properly
		// For now, allow all origins for development
		return true
	},
}

// HandleWebSocket handles WebSocket upgrade requests
func HandleWebSocket(hub *Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user ID from query parameter or authenticated context
		userIDStr := c.Query("user_id")
		if userIDStr == "" {
			// Try to get from authenticated context
			if userID, exists := c.Get("userID"); exists {
				// Convert uint to UUID
				if uid, ok := userID.(uint); ok {
					userIDStr = uuid.NewSHA1(uuid.NameSpaceOID, []byte(string(rune(uid)))).String()
				}
			}
		}

		if userIDStr == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "user_id required"})
			return
		}

		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id format"})
			return
		}

		// Upgrade HTTP connection to WebSocket
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Printf("WebSocket upgrade error: %v", err)
			return
		}

		// Create new client
		client := NewClient(userID, hub, conn)

		// Register client with hub
		hub.register <- client

		// Start client's message pumps
		client.Start()

		log.Printf("🔌 WebSocket connection established for user: %s", userID)
	}
}

// HandleWebSocketAuth handles authenticated WebSocket connections
// This version extracts user ID from the JWT token
func HandleWebSocketAuth(hub *Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user ID from authenticated context (set by auth middleware)
		userIDValue, exists := c.Get("userID")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		// Convert uint to UUID
		userID, ok := userIDValue.(uint)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user ID"})
			return
		}

		// Create deterministic UUID from user ID
		userUUID := uuid.NewSHA1(uuid.NameSpaceOID, []byte(string(rune(userID))))

		// Upgrade HTTP connection to WebSocket
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Printf("WebSocket upgrade error: %v", err)
			return
		}

		// Create new client
		client := NewClient(userUUID, hub, conn)

		// Register client with hub
		hub.register <- client

		// Start client's message pumps
		client.Start()

		log.Printf("🔌 Authenticated WebSocket connection for user: %s (ID: %d)", userUUID, userID)
	}
}
