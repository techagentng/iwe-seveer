package models

import (
	"time"

	"github.com/google/uuid"
)

// JobStatus represents the status of a processing job
type JobStatus string

const (
	JobStatusQueued     JobStatus = "queued"
	JobStatusProcessing JobStatus = "processing"
	JobStatusCompleted  JobStatus = "completed"
	JobStatusFailed     JobStatus = "failed"
)

// ProcessingJob represents a background job for file processing with AI
type ProcessingJob struct {
	ID          uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID      uuid.UUID  `json:"user_id" gorm:"type:uuid;not null;index"`
	FileID      uuid.UUID  `json:"file_id" gorm:"type:uuid;not null;index"`
	
	// File information
	FileName    string     `json:"file_name" gorm:"type:varchar(255)"`
	FileURL     string     `json:"file_url" gorm:"type:text"`
	FileType    FileType   `json:"file_type" gorm:"type:varchar(20)"`
	
	// User's question/prompt about the document
	Prompt      string     `json:"prompt" gorm:"type:text"`
	
	// Priority and scheduling
	Priority    int        `json:"priority" gorm:"default:0;index"` // Higher = more important
	ScheduledAt *time.Time `json:"scheduled_at,omitempty" gorm:"index"` // For delayed execution
	
	// Processing status and progress
	Status      JobStatus  `json:"status" gorm:"type:varchar(20);default:'queued';index"`
	Progress    int        `json:"progress" gorm:"default:0"` // 0-100
	
	// Processing results
	ExtractedText string   `json:"extracted_text,omitempty" gorm:"type:text"`
	AIResponse    string   `json:"ai_response,omitempty" gorm:"type:text"`
	
	// Error handling
	ErrorMsg      string   `json:"error_msg,omitempty" gorm:"type:text"`
	RetryCount    int      `json:"retry_count" gorm:"default:0"`
	
	// Timestamps
	CreatedAt   time.Time  `json:"created_at" gorm:"autoCreateTime"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	UpdatedAt   time.Time  `json:"updated_at" gorm:"autoUpdateTime"`
	
	// Relations
	User         User         `json:"user,omitempty" gorm:"foreignKey:UserID"`
	UploadedFile UploadedFile `json:"uploaded_file,omitempty" gorm:"foreignKey:FileID"`
}

// TableName specifies the table name for ProcessingJob
func (ProcessingJob) TableName() string {
	return "processing_jobs"
}

// BeforeCreate hook to generate UUID if not set
func (j *ProcessingJob) BeforeCreate() error {
	if j.ID == uuid.Nil {
		j.ID = uuid.New()
	}
	return nil
}

// IsCompleted checks if the job is in a terminal state
func (j *ProcessingJob) IsCompleted() bool {
	return j.Status == JobStatusCompleted || j.Status == JobStatusFailed
}

// CanRetry checks if the job can be retried
func (j *ProcessingJob) CanRetry() bool {
	return j.Status == JobStatusFailed && j.RetryCount < 3
}

// Duration returns the job processing duration
func (j *ProcessingJob) Duration() time.Duration {
	if j.StartedAt == nil {
		return 0
	}
	
	endTime := time.Now()
	if j.CompletedAt != nil {
		endTime = *j.CompletedAt
	}
	
	return endTime.Sub(*j.StartedAt)
}

// IsReadyToRun checks if a scheduled job is ready to execute
func (j *ProcessingJob) IsReadyToRun() bool {
	if j.ScheduledAt == nil {
		return true // Not scheduled, can run immediately
	}
	return time.Now().After(*j.ScheduledAt) || time.Now().Equal(*j.ScheduledAt)
}
