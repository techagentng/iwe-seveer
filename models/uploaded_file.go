package models

import (
	"time"

	"github.com/google/uuid"
)

// FileStatus represents the processing status of an uploaded file
type FileStatus string

const (
	FileStatusPending    FileStatus = "pending"
	FileStatusProcessing FileStatus = "processing"
	FileStatusCompleted  FileStatus = "completed"
	FileStatusFailed     FileStatus = "failed"
)

// FileType represents the type of uploaded file
type FileType string

const (
	FileTypeCSV   FileType = "csv"
	FileTypePDF   FileType = "pdf"
	FileTypeImage FileType = "image"
)

// UploadedFile represents a file uploaded by a user
type UploadedFile struct {
	ID          uuid.UUID  `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	UserID      uuid.UUID  `gorm:"type:uuid;not null;index" json:"user_id"`
	FileName    string     `gorm:"type:varchar(255);not null" json:"file_name"`
	FileType    FileType   `gorm:"type:varchar(50);not null" json:"file_type"`
	FileURL     string     `gorm:"type:text" json:"file_url"`
	FileSize    int64      `gorm:"not null" json:"file_size"`
	Status      FileStatus `gorm:"type:varchar(50);default:'pending'" json:"status"`
	ErrorMsg    string     `gorm:"type:text" json:"error_msg,omitempty"`
	CreatedAt   time.Time  `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
	ProcessedAt *time.Time `json:"processed_at,omitempty"`
	
	// Relationships
	BankStatements  []BankStatement  `gorm:"foreignKey:FileID" json:"bank_statements,omitempty"`
	DocumentChunks  []DocumentChunk  `gorm:"foreignKey:FileID" json:"document_chunks,omitempty"`
}

// TableName specifies the table name for UploadedFile
func (UploadedFile) TableName() string {
	return "uploaded_files"
}
