package server

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	errs "github.com/techagentng/iweapp/errors"
	"github.com/techagentng/iweapp/server/response"
	"github.com/techagentng/iweapp/services/jwt"
)

// Authorize middleware validates JWT tokens and sets user context
func (s *Server) Authorize() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract token from Authorization header
		authHeader := c.Request.Header.Get("Authorization")
		if len(authHeader) < 8 || authHeader[:7] != "Bearer " {
			respondAndAbort(c, "Missing or invalid authorization header", http.StatusUnauthorized, nil, errs.New("Unauthorized", http.StatusUnauthorized))
			return
		}
		accessToken := authHeader[7:]

		// Check if token is blacklisted
		if s.AuthRepository.IsTokenInBlacklist(accessToken) {
			respondAndAbort(c, "Token has been revoked", http.StatusUnauthorized, nil, errs.New("Unauthorized", http.StatusUnauthorized))
			return
		}

		// Validate token and extract claims
		accessClaims, err := jwt.ValidateAndGetClaims(accessToken, s.Config.JWTSecret)
		if err != nil {
			respondAndAbort(c, "Invalid token", http.StatusUnauthorized, nil, errs.New("Unauthorized", http.StatusUnauthorized))
			return
		}

		// Extract and convert userID from claims
		userIDValue, ok := accessClaims["id"]
		if !ok {
			respondAndAbort(c, "User ID not found in token", http.StatusBadRequest, nil, errs.New("Bad Request", http.StatusBadRequest))
			return
		}

		var userID uint
		switch v := userIDValue.(type) {
		case float64:
			userID = uint(v)
		case int:
			userID = uint(v)
		case string:
			parsedID, err := strconv.ParseUint(v, 10, 32)
			if err != nil {
				respondAndAbort(c, "Invalid User ID format", http.StatusBadRequest, nil, errs.New("Bad Request", http.StatusBadRequest))
				return
			}
			userID = uint(parsedID)
		default:
			respondAndAbort(c, "Invalid User ID type", http.StatusBadRequest, nil, errs.New("Bad Request", http.StatusBadRequest))
			return
		}

		// Fetch user from database
		user, err := s.AuthRepository.FindUserByID(userID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				respondAndAbort(c, "User not found", http.StatusUnauthorized, nil, errs.New("Unauthorized", http.StatusUnauthorized))
				return
			}
			respondAndAbort(c, "Failed to fetch user", http.StatusInternalServerError, nil, errs.ErrInternalServerError)
			return
		}

		// Set user data in context for downstream handlers
		c.Set("user", user)
		c.Set("userID", userID)
		c.Set("access_token", accessToken)
		
		// Extract role from claims
		if role, ok := accessClaims["role"].(string); ok {
			c.Set("user_role", role)
		}

		c.Next()
	}
}

// respondAndAbort is a helper function to send error response and abort the request
func respondAndAbort(c *gin.Context, message string, status int, data interface{}, e *errs.Error) {
	response.JSON(c, message, status, data, e)
	c.Abort()
}
