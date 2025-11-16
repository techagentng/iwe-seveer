package server

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/techagentng/iweapp/errors"
	"github.com/techagentng/iweapp/models"
	"github.com/techagentng/iweapp/server/response"
	"github.com/techagentng/iweapp/services/jwt"
)

// handleSignup handles user registration
func (s *Server) handleSignup() gin.HandlerFunc {
	return func(c *gin.Context) {
		var user models.User
		if err := c.ShouldBindJSON(&user); err != nil {
			response.JSON(c, "Invalid request data", http.StatusBadRequest, nil, errors.New(err.Error(), http.StatusBadRequest))
			return
		}

		// Get or assign the "User" role
		role, err := s.AuthService.GetRoleByName(models.RoleUser)
		if err != nil {
			log.Printf("handleSignup: error getting role: %v", err)
			response.JSON(c, "Failed to assign user role", http.StatusInternalServerError, nil, errors.ErrInternalServerError)
			return
		}
		user.RoleID = role.ID

		// Create the user
		createdUser, err := s.AuthService.SignupUser(&user)
		if err != nil {
			log.Printf("handleSignup: error creating user: %v", err)
			response.HandleErrors(c, err)
			return
		}

		// Generate tokens
		accessToken, refreshToken, err := jwt.GenerateTokenPair(
			createdUser.Email,
			s.Config.JWTSecret,
			createdUser.AdminStatus,
			createdUser.ID,
			role.Name,
		)
		if err != nil {
			log.Printf("handleSignup: error generating tokens: %v", err)
			response.JSON(c, "Signup successful but failed to generate tokens", http.StatusCreated, nil, errors.ErrInternalServerError)
			return
		}

		// Send welcome email (non-blocking)
		go func() {
			if s.Mail != nil {
				_, err := s.Mail.SendWelcomeMessage(createdUser.Email, "Welcome to Our Platform!")
				if err != nil {
					log.Printf("handleSignup: error sending welcome email: %v", err)
				}
			}
		}()

		// Return response
		response.JSON(c, "Signup successful", http.StatusCreated, models.LoginResponse{
			UserResponse: models.UserResponse{
				ID:        createdUser.ID,
				Fullname:  createdUser.Fullname,
				Username:  createdUser.Username,
				Telephone: createdUser.Telephone,
				Email:     createdUser.Email,
				RoleName:  role.Name,
			},
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
		}, nil)
	}
}

// handleLogin handles user login
func (s *Server) handleLogin() gin.HandlerFunc {
	return func(c *gin.Context) {
		var loginRequest models.LoginRequest
		if err := c.ShouldBindJSON(&loginRequest); err != nil {
			response.JSON(c, "Invalid request data", http.StatusBadRequest, nil, errors.New(err.Error(), http.StatusBadRequest))
			return
		}

		loginResponse, err := s.AuthService.LoginUser(&loginRequest)
		if err != nil {
			response.JSON(c, "", err.Status, nil, err)
			return
		}

		response.JSON(c, "Login successful", http.StatusOK, loginResponse, nil)
	}
}

// handleGoogleLogin handles Google OAuth login/signup
func (s *Server) handleGoogleLogin() gin.HandlerFunc {
	return func(c *gin.Context) {
		var loginRequest models.GoogleLoginRequest
		if err := c.ShouldBindJSON(&loginRequest); err != nil {
			response.JSON(c, "Invalid request data", http.StatusBadRequest, nil, errors.New(err.Error(), http.StatusBadRequest))
			return
		}

		loginResponse, err := s.AuthService.GoogleLoginUser(&loginRequest)
		if err != nil {
			response.JSON(c, "", err.Status, nil, err)
			return
		}

		response.JSON(c, "Login successful", http.StatusOK, loginResponse, nil)
	}
}

// handleLogout handles user logout by blacklisting the token
func (s *Server) handleLogout() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get access token from context (set by Authorize middleware)
		accessToken, exists := c.Get("access_token")
		if !exists {
			response.JSON(c, "Access token not found", http.StatusUnauthorized, nil, errors.New("Unauthorized", http.StatusUnauthorized))
			return
		}

		token, ok := accessToken.(string)
		if !ok {
			response.JSON(c, "Invalid token format", http.StatusInternalServerError, nil, errors.ErrInternalServerError)
			return
		}

		// Get user from context
		userInterface, exists := c.Get("user")
		if !exists {
			response.JSON(c, "User not found in context", http.StatusUnauthorized, nil, errors.New("Unauthorized", http.StatusUnauthorized))
			return
		}

		user, ok := userInterface.(*models.User)
		if !ok {
			response.JSON(c, "Invalid user data", http.StatusInternalServerError, nil, errors.ErrInternalServerError)
			return
		}

		// Add token to blacklist
		blacklistEntry := &models.Blacklist{
			Token: token,
			Email: user.Email,
		}
		if err := s.AuthRepository.AddToBlackList(blacklistEntry); err != nil {
			log.Printf("handleLogout: error blacklisting token: %v", err)
			response.JSON(c, "Logout failed", http.StatusInternalServerError, nil, errors.ErrInternalServerError)
			return
		}

		response.JSON(c, "Logout successful", http.StatusOK, nil, nil)
	}
}

