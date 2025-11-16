package models

import (
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// User represents a user of the application
type User struct {
	Model
	Fullname       string    `json:"fullname" binding:"required,min=2"`
	Username       string    `json:"username" binding:"required,min=2" gorm:"unique"`
	Telephone      string    `json:"telephone" gorm:"default:null"`
	Email          string    `json:"email" gorm:"unique;not null" binding:"required,email"`
	Password       string    `json:"password,omitempty" validate:"omitempty,min=6"`
	HashedPassword string    `json:"-"`
	IsSocial       bool      `json:"is_social" gorm:"default:false"`
	AdminStatus    bool      `json:"is_admin" gorm:"default:false"`
	RoleID         uuid.UUID `gorm:"type:uuid;not null" json:"role_id"`
	Role           Role      `gorm:"foreignKey:RoleID" json:"role"`
}

// VerifyPassword checks if the provided password matches the hashed password
func (u *User) VerifyPassword(password string) error {
	return bcrypt.CompareHashAndPassword([]byte(u.HashedPassword), []byte(password))
}

// UserResponse represents the response structure for user data
type UserResponse struct {
	ID        uint   `json:"id"`
	Fullname  string `json:"fullname"`
	Username  string `json:"username"`
	Telephone string `json:"telephone"`
	Email     string `json:"email"`
	RoleName  string `json:"role_name"`
}

// LoginRequest represents the login request structure
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// GoogleLoginRequest represents the Google OAuth login request
type GoogleLoginRequest struct {
	Email     string `json:"email" binding:"required,email"`
	Fullname  string `json:"fullname" binding:"omitempty"`
	Telephone string `json:"telephone" binding:"omitempty"`
}

// LoginResponse represents the response structure after successful login
type LoginResponse struct {
	UserResponse
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// OAuthState stores OAuth state tokens for security
type OAuthState struct {
	ID        string `gorm:"primaryKey" json:"id"`
	State     string `gorm:"not null" json:"state"`
	UserID    string `json:"user_id"`
	CreatedAt int64  `json:"created_at"`
}