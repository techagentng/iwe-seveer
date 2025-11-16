package models

import (
	"github.com/google/uuid"
)

// Role constants
const (
	RoleAdmin = "Admin"
	RoleUser  = "User"
)

// Role represents a user role in the system
type Role struct {
	ID   uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	Name string    `gorm:"unique;not null" json:"name"`
}
