package websocket

import (
	"fmt"
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
		// Strict origin check for production
		origin := r.Header.Get("Origin")
		allowed := map[string]bool{
			"https://api.iweapps.com": true,
			"https://iweapps.com":      true,
			"https://www.iweapps.com":  true,
		}
		if allowed[origin] {
			return true
		}
		// Allow localhost for development convenience
		if origin == "http://localhost:3000" || origin == "http://localhost:3001" || origin == "http://localhost:5173" || origin == "http://127.0.0.1:3000" || origin == "http://127.0.0.1:3001" {
			return true
		}
		log.Printf("[WS] Blocked origin during upgrade: origin=%s remote=%s", origin, r.RemoteAddr)
		return false
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
			log.Printf("[WS] Missing user_id and no auth context: remote=%s ua=%s", c.Request.RemoteAddr, c.Request.UserAgent())
			c.JSON(http.StatusBadRequest, gin.H{"error": "user_id required"})
			return
		}

		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			log.Printf("[WS] Invalid user_id format: value=%s err=%v", userIDStr, err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id format"})
			return
		}

		// Upgrade HTTP connection to WebSocket
		log.Printf("[WS] Upgrade attempt (public): user=%s origin=%s remote=%s", userID, c.Request.Header.Get("Origin"), c.Request.RemoteAddr)
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Printf("[WS] Upgrade failed (public): user=%s err=%v", userID, err)
			return
		}

		// Create new client
		client := NewClient(userID, hub, conn)

		// Register client with hub
		hub.register <- client

		// Start client's message pumps
		client.Start()

		log.Printf("[WS] Connected (public): user=%s", userID)
	}
}

// HandleWebSocketAuth handles authenticated WebSocket connections
// This version extracts user ID from the JWT token
func HandleWebSocketAuth(hub *Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user ID from authenticated context (set by auth middleware)
		userIDValue, exists := c.Get("userID")
		if !exists {
			log.Printf("[WS] Missing user context on auth endpoint: remote=%s", c.Request.RemoteAddr)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		// Convert uint to UUID
		userID, ok := userIDValue.(uint)
		if !ok {
			log.Printf("[WS] Invalid userID type in context: value=%v", userIDValue)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user ID"})
			return
		}

		// Create deterministic UUID from user ID
		userUUID := uuid.NewSHA1(uuid.NameSpaceOID, []byte(string(rune(userID))))

		// Upgrade HTTP connection to WebSocket
		log.Printf("[WS] Upgrade attempt (auth): user=%d origin=%s remote=%s", userID, c.Request.Header.Get("Origin"), c.Request.RemoteAddr)
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Printf("[WS] Upgrade failed (auth): user=%d err=%v", userID, err)
			return
		}

		// Create new client
		client := NewClient(userUUID, hub, conn)

		// Register client with hub
		hub.register <- client

		// Start client's message pumps
		client.Start()

		log.Printf("[WS] Connected (auth): user=%s", userUUID)
	}
}

// HandleWebSocketAuto supports both authenticated and public connections on one endpoint
// Priority: use auth context if available; otherwise expect user_id query param
func HandleWebSocketAuto(hub *Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Try authenticated context first
		if v, ok := c.Get("userID"); ok {
			if uid, ok := v.(uint); ok {
				userUUID := uuid.NewSHA1(uuid.NameSpaceOID, []byte(string(rune(uid))))
				log.Printf("[WS] Upgrade attempt (auto-auth): user=%d origin=%s remote=%s", uid, c.Request.Header.Get("Origin"), c.Request.RemoteAddr)
				conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
				if err != nil {
					log.Printf("[WS] Upgrade failed (auto-auth): user=%d err=%v", uid, err)
					return
				}
				client := NewClient(userUUID, hub, conn)
				hub.register <- client
				client.Start()
				log.Printf("[WS] Connected (auto-auth): user=%s", userUUID)
				return
			}
			log.Printf("[WS] Invalid userID type in context on auto endpoint: value=%v", v)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user ID"})
			return
		}

		// Fallback to public connection with query param
		userIDStr := c.Query("user_id")
		if userIDStr == "" {
			log.Printf("[WS] user_id missing on auto endpoint and no auth context: remote=%s ua=%s", c.Request.RemoteAddr, c.Request.UserAgent())
			c.JSON(http.StatusBadRequest, gin.H{"error": "user_id query param required or provide Authorization header"})
			return
		}

		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			log.Printf("[WS] Invalid user_id format on auto endpoint: value=%s err=%v", userIDStr, err)
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid user_id format: %v", err)})
			return
		}

		log.Printf("[WS] Upgrade attempt (auto-public): user=%s origin=%s remote=%s", userID, c.Request.Header.Get("Origin"), c.Request.RemoteAddr)
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Printf("[WS] Upgrade failed (auto-public): user=%s err=%v", userID, err)
			return
		}
		client := NewClient(userID, hub, conn)
		hub.register <- client
		client.Start()
		log.Printf("[WS] Connected (auto-public): user=%s", userID)
	}
}
