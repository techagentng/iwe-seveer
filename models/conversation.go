package models

import "time"

type Conversation struct {
	Model
	UserID        uint      `json:"user_id" gorm:"index;not null"`
	Title         string    `json:"title" gorm:"type:varchar(255)"`
	Archived      bool      `json:"archived" gorm:"default:false;index"`
	LastMessageAt time.Time `json:"last_message_at" gorm:"index"`
}
