package models

type Message struct {
	Model
	ConversationID uint      `json:"conversation_id" gorm:"index;not null"`
	UserID         uint      `json:"user_id" gorm:"index;not null"`
	Role           string    `json:"role" gorm:"type:varchar(20);not null"` // user | assistant
	Content        string    `json:"content" gorm:"type:text;not null"`
	MessageKey     string    `json:"message_id" gorm:"type:varchar(100);index"` // frontend messageId
}
