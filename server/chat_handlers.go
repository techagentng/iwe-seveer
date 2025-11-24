package server

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/techagentng/iweapp/models"
	errs "github.com/techagentng/iweapp/errors"
	"github.com/techagentng/iweapp/server/response"
)

// handleCreateConversation POST /api/v1/chat/conversations
// Body: { title?: string }
func (s *Server) handleCreateConversation() gin.HandlerFunc {
	return func(c *gin.Context) {
		uidVal, ok := c.Get("userID")
		if !ok {
			response.JSON(c, "Unauthorized", http.StatusUnauthorized, nil, errs.New("unauthorized", http.StatusUnauthorized))
			return
		}
		userID := uidVal.(uint)

		var body struct{ Title string `json:"title"` }
		if err := c.ShouldBindJSON(&body); err != nil {
			response.JSON(c, "Invalid request", http.StatusBadRequest, nil, errs.New(err.Error(), http.StatusBadRequest))
			return
		}

		conv := models.Conversation{UserID: userID, Title: body.Title}
		if err := s.DB.Create(&conv).Error; err != nil {
			response.JSON(c, "Failed to create conversation", http.StatusInternalServerError, nil, errs.ErrInternalServerError)
			return
		}
		response.JSON(c, "OK", http.StatusOK, gin.H{"conversationId": conv.ID}, nil)
	}
}

// handleListConversations GET /api/v1/chat/conversations
func (s *Server) handleListConversations() gin.HandlerFunc {
	return func(c *gin.Context) {
		uidVal, ok := c.Get("userID")
		if !ok {
			response.JSON(c, "Unauthorized", http.StatusUnauthorized, nil, errs.New("unauthorized", http.StatusUnauthorized))
			return
		}
		userID := uidVal.(uint)

		var convs []models.Conversation
		if err := s.DB.Where("user_id = ? AND archived = false", userID).Order("COALESCE(last_message_at, created_at) DESC").Find(&convs).Error; err != nil {
			response.JSON(c, "Failed to fetch conversations", http.StatusInternalServerError, nil, errs.ErrInternalServerError)
			return
		}
		response.JSON(c, "OK", http.StatusOK, gin.H{"conversations": convs}, nil)
	}
}

// handleGetMessages GET /api/v1/chat/conversations/:id/messages?limit=50&before=<messageID>
func (s *Server) handleGetMessages() gin.HandlerFunc {
	return func(c *gin.Context) {
		uidVal, ok := c.Get("userID")
		if !ok {
			response.JSON(c, "Unauthorized", http.StatusUnauthorized, nil, errs.New("unauthorized", http.StatusUnauthorized))
			return
		}
		userID := uidVal.(uint)

		convID, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil {
			response.JSON(c, "Invalid conversation id", http.StatusBadRequest, nil, errs.New("bad request", http.StatusBadRequest))
			return
		}

		var conv models.Conversation
		if err := s.DB.First(&conv, convID).Error; err != nil || conv.UserID != userID {
			response.JSON(c, "Conversation not found", http.StatusNotFound, nil, errs.New("not found", http.StatusNotFound))
			return
		}

		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
		if limit <= 0 || limit > 200 { limit = 50 }
		before := c.Query("before")

		q := s.DB.Where("conversation_id = ?", conv.ID)
		if before != "" {
			// before is numeric message primary key
			if bid, err := strconv.ParseUint(before, 10, 64); err == nil {
				q = q.Where("id < ?", bid)
			}
		}
		var msgs []models.Message
		if err := q.Order("id DESC").Limit(limit).Find(&msgs).Error; err != nil {
			response.JSON(c, "Failed to fetch messages", http.StatusInternalServerError, nil, errs.ErrInternalServerError)
			return
		}
		// return in ascending order for UI
		for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 { msgs[i], msgs[j] = msgs[j], msgs[i] }
		response.JSON(c, "OK", http.StatusOK, gin.H{"messages": msgs}, nil)
	}
}

// handlePostMessage POST /api/v1/chat/messages
// Body: { conversationId: number, role: 'user'|'assistant', content: string, messageId: string }
func (s *Server) handlePostMessage() gin.HandlerFunc {
	return func(c *gin.Context) {
		uidVal, ok := c.Get("userID")
		if !ok {
			response.JSON(c, "Unauthorized", http.StatusUnauthorized, nil, errs.New("unauthorized", http.StatusUnauthorized))
			return
		}
		userID := uidVal.(uint)

		var body struct {
			ConversationID uint   `json:"conversationId" binding:"required"`
			Role           string `json:"role" binding:"required"`
			Content        string `json:"content" binding:"required"`
			MessageKey     string `json:"messageId"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			response.JSON(c, "Invalid request", http.StatusBadRequest, nil, errs.New(err.Error(), http.StatusBadRequest))
			return
		}
		if body.Role != "user" && body.Role != "assistant" {
			response.JSON(c, "Invalid role", http.StatusBadRequest, nil, errs.New("bad request", http.StatusBadRequest))
			return
		}

		var conv models.Conversation
		if err := s.DB.First(&conv, body.ConversationID).Error; err != nil || conv.UserID != userID {
			response.JSON(c, "Conversation not found", http.StatusNotFound, nil, errs.New("not found", http.StatusNotFound))
			return
		}

		msg := models.Message{
			ConversationID: conv.ID,
			UserID:         conv.UserID,
			Role:           body.Role,
			Content:        body.Content,
			MessageKey:     body.MessageKey,
		}
		if err := s.DB.Create(&msg).Error; err != nil {
			response.JSON(c, "Failed to save message", http.StatusInternalServerError, nil, errs.ErrInternalServerError)
			return
		}
		// update last_message_at
		s.DB.Model(&conv).Update("last_message_at", time.Now())
		response.JSON(c, "OK", http.StatusOK, gin.H{"id": msg.ID}, nil)
	}
}
