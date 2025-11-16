package models

import (
	"time"

	"github.com/google/uuid"
)

// DocumentChunk represents a chunk of text extracted from a document
type DocumentChunk struct {
	ID         uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	FileID     uuid.UUID `gorm:"type:uuid;not null;index" json:"file_id"`
	ChunkIndex int       `gorm:"not null" json:"chunk_index"`
	Content    string    `gorm:"type:text;not null" json:"content"`
	CreatedAt  time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
	
	// Relationship
	UploadedFile UploadedFile `gorm:"foreignKey:FileID" json:"-"`
}

// TableName specifies the table name for DocumentChunk
func (DocumentChunk) TableName() string {
	return "document_chunks"
}
