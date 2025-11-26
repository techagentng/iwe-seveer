package server

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/techagentng/iweapp/db"
	"github.com/techagentng/iweapp/models"
)

// ContactRequest represents the JSON request body for contact form submission
type ContactRequest struct {
	FullName string `json:"full_name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
}

// handleContact handles the contact form submission
func (s *Server) handleContact() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Parse request body
		var req ContactRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Create contact repository
		contactRepo := db.NewContactRepository(s.DB)

		// Check if email already exists
		existingContact, err := contactRepo.GetContactByEmail(req.Email)
		if err == nil && existingContact != nil {
			c.JSON(http.StatusConflict, gin.H{
				"error": "Email already exists",
			})
			return
		}

		// Create new contact
		contact := &models.Contact{
			ID:        uuid.New().String(),
			FullName:  req.FullName,
			Email:     req.Email,
			CreatedAt: time.Now(),
		}

		// Save to database
		if err := contactRepo.CreateContact(contact); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to save contact information",
			})
			return
		}

		// Return success response
		c.JSON(http.StatusCreated, gin.H{
			"message": "Contact information saved successfully",
			"contact": contact,
		})
	}
}
