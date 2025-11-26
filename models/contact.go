package models

import "time"

// Contact represents a contact form submission
type Contact struct {
	ID        string    `json:"id" gorm:"primaryKey"`
	FullName  string    `json:"full_name" gorm:"not null"`
	Email     string    `json:"email" gorm:"not null;uniqueIndex"`
	CreatedAt time.Time `json:"created_at"`
}

// TableName specifies the table name for the Contact model
func (Contact) TableName() string {
	return "contacts"
}
