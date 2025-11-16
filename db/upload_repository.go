package db

import (
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/techagentng/iweapp/models"
	"gorm.io/gorm"
)

// UploadRepository defines the interface for file upload operations
type UploadRepository interface {
	CreateUploadedFile(file *models.UploadedFile) error
	UpdateUploadedFile(file *models.UploadedFile) error
	GetUploadedFileByID(id uuid.UUID) (*models.UploadedFile, error)
	GetUploadedFilesByUserID(userID uuid.UUID, limit, offset int) ([]models.UploadedFile, error)
	
	BatchCreateBankStatements(statements []models.BankStatement) error
	GetBankStatementsByFileID(fileID uuid.UUID) ([]models.BankStatement, error)
	
	CreateDocumentChunk(chunk *models.DocumentChunk) error
	BatchCreateDocumentChunks(chunks []models.DocumentChunk) error
	GetDocumentChunksByFileID(fileID uuid.UUID) ([]models.DocumentChunk, error)
}

// uploadRepository implements UploadRepository
type uploadRepository struct {
	db *gorm.DB
}

// NewUploadRepository creates a new UploadRepository instance
func NewUploadRepository(db *GormDB) UploadRepository {
	return &uploadRepository{
		db: db.DB,
	}
}

// CreateUploadedFile creates a new uploaded file record
func (r *uploadRepository) CreateUploadedFile(file *models.UploadedFile) error {
	if err := r.db.Create(file).Error; err != nil {
		log.Printf("CreateUploadedFile: error creating file: %v", err)
		return fmt.Errorf("failed to create uploaded file: %w", err)
	}
	return nil
}

// UpdateUploadedFile updates an existing uploaded file record
func (r *uploadRepository) UpdateUploadedFile(file *models.UploadedFile) error {
	if err := r.db.Save(file).Error; err != nil {
		log.Printf("UpdateUploadedFile: error updating file: %v", err)
		return fmt.Errorf("failed to update uploaded file: %w", err)
	}
	return nil
}

// GetUploadedFileByID retrieves an uploaded file by ID
func (r *uploadRepository) GetUploadedFileByID(id uuid.UUID) (*models.UploadedFile, error) {
	var file models.UploadedFile
	if err := r.db.Where("id = ?", id).First(&file).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("uploaded file not found")
		}
		log.Printf("GetUploadedFileByID: error retrieving file: %v", err)
		return nil, fmt.Errorf("failed to retrieve uploaded file: %w", err)
	}
	return &file, nil
}

// GetUploadedFilesByUserID retrieves all uploaded files for a user with pagination
func (r *uploadRepository) GetUploadedFilesByUserID(userID uuid.UUID, limit, offset int) ([]models.UploadedFile, error) {
	var files []models.UploadedFile
	query := r.db.Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset)
	
	if err := query.Find(&files).Error; err != nil {
		log.Printf("GetUploadedFilesByUserID: error retrieving files: %v", err)
		return nil, fmt.Errorf("failed to retrieve uploaded files: %w", err)
	}
	return files, nil
}

// BatchCreateBankStatements creates multiple bank statement records in batches
func (r *uploadRepository) BatchCreateBankStatements(statements []models.BankStatement) error {
	if len(statements) == 0 {
		return nil
	}

	// Use batch insert for better performance (batch size of 100)
	batchSize := 100
	for i := 0; i < len(statements); i += batchSize {
		end := i + batchSize
		if end > len(statements) {
			end = len(statements)
		}
		
		batch := statements[i:end]
		if err := r.db.Create(&batch).Error; err != nil {
			log.Printf("BatchCreateBankStatements: error creating batch: %v", err)
			return fmt.Errorf("failed to create bank statements batch: %w", err)
		}
	}
	
	log.Printf("Successfully inserted %d bank statements", len(statements))
	return nil
}

// GetBankStatementsByFileID retrieves all bank statements for a file
func (r *uploadRepository) GetBankStatementsByFileID(fileID uuid.UUID) ([]models.BankStatement, error) {
	var statements []models.BankStatement
	if err := r.db.Where("file_id = ?", fileID).
		Order("transaction_date DESC").
		Find(&statements).Error; err != nil {
		log.Printf("GetBankStatementsByFileID: error retrieving statements: %v", err)
		return nil, fmt.Errorf("failed to retrieve bank statements: %w", err)
	}
	return statements, nil
}

// CreateDocumentChunk creates a new document chunk record
func (r *uploadRepository) CreateDocumentChunk(chunk *models.DocumentChunk) error {
	if err := r.db.Create(chunk).Error; err != nil {
		log.Printf("CreateDocumentChunk: error creating chunk: %v", err)
		return fmt.Errorf("failed to create document chunk: %w", err)
	}
	return nil
}

// BatchCreateDocumentChunks creates multiple document chunk records in batches
func (r *uploadRepository) BatchCreateDocumentChunks(chunks []models.DocumentChunk) error {
	if len(chunks) == 0 {
		return nil
	}

	// Use batch insert for better performance (batch size of 50)
	batchSize := 50
	for i := 0; i < len(chunks); i += batchSize {
		end := i + batchSize
		if end > len(chunks) {
			end = len(chunks)
		}
		
		batch := chunks[i:end]
		if err := r.db.Create(&batch).Error; err != nil {
			log.Printf("BatchCreateDocumentChunks: error creating batch: %v", err)
			return fmt.Errorf("failed to create document chunks batch: %w", err)
		}
	}
	
	log.Printf("Successfully inserted %d document chunks", len(chunks))
	return nil
}

// GetDocumentChunksByFileID retrieves all document chunks for a file
func (r *uploadRepository) GetDocumentChunksByFileID(fileID uuid.UUID) ([]models.DocumentChunk, error) {
	var chunks []models.DocumentChunk
	if err := r.db.Where("file_id = ?", fileID).
		Order("chunk_index ASC").
		Find(&chunks).Error; err != nil {
		log.Printf("GetDocumentChunksByFileID: error retrieving chunks: %v", err)
		return nil, fmt.Errorf("failed to retrieve document chunks: %w", err)
	}
	return chunks, nil
}
