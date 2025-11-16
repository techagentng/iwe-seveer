package server

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/techagentng/iweapp/errors"
	"github.com/techagentng/iweapp/models"
	"github.com/techagentng/iweapp/processors"
	"github.com/techagentng/iweapp/server/response"
	"github.com/techagentng/iweapp/storage"
)

const (
	MaxFileSize = 50 << 20 // 50 MB
)

// handleFileUpload handles file upload requests
func (s *Server) handleFileUpload() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user ID from context (set by auth middleware)
		userIDValue, exists := c.Get("userID")
		if !exists {
			response.JSON(c, "Unauthorized", http.StatusUnauthorized, nil, errors.New("User not authenticated", http.StatusUnauthorized))
			return
		}

		// Convert uint to UUID (generate deterministic UUID from user ID)
		userID, ok := userIDValue.(uint)
		if !ok {
			response.JSON(c, "Invalid user ID", http.StatusInternalServerError, nil, errors.ErrInternalServerError)
			return
		}

		// Create a UUID from the user ID (you can use a namespace UUID or generate one)
		// For now, we'll create a deterministic UUID based on user ID
		userUUID := uuid.NewSHA1(uuid.NameSpaceOID, []byte(fmt.Sprintf("user-%d", userID)))

		// Parse multipart form
		if err := c.Request.ParseMultipartForm(MaxFileSize); err != nil {
			response.JSON(c, "File too large or invalid form data", http.StatusBadRequest, nil, errors.New(err.Error(), http.StatusBadRequest))
			return
		}

		// Get file from form
		file, header, err := c.Request.FormFile("file")
		if err != nil {
			response.JSON(c, "File is required", http.StatusBadRequest, nil, errors.New("No file provided", http.StatusBadRequest))
			return
		}
		defer file.Close()

		// Get file type from form
		fileTypeStr := c.PostForm("type")
		if fileTypeStr == "" {
			response.JSON(c, "File type is required", http.StatusBadRequest, nil, errors.New("type field is required (csv, pdf, or image)", http.StatusBadRequest))
			return
		}

		// Validate file type
		fileType := models.FileType(strings.ToLower(fileTypeStr))
		if !isValidFileType(fileType) {
			response.JSON(c, "Invalid file type", http.StatusBadRequest, nil, errors.New("type must be csv, pdf, or image", http.StatusBadRequest))
			return
		}

		// Validate file extension matches type
		if err := validateFileExtension(header.Filename, fileType); err != nil {
			response.JSON(c, "File extension mismatch", http.StatusBadRequest, nil, errors.New(err.Error(), http.StatusBadRequest))
			return
		}

		// Validate file size
		if header.Size > MaxFileSize {
			response.JSON(c, "File too large", http.StatusBadRequest, nil, errors.New("Maximum file size is 50MB", http.StatusBadRequest))
			return
		}

		log.Printf("Received file upload: %s (type: %s, size: %d bytes)", header.Filename, fileType, header.Size)

		// Create uploaded file record
		uploadedFile := &models.UploadedFile{
			UserID:   userUUID,
			FileName: header.Filename,
			FileType: fileType,
			FileSize: header.Size,
			Status:   models.FileStatusPending,
		}

		if err := s.UploadRepository.CreateUploadedFile(uploadedFile); err != nil {
			log.Printf("handleFileUpload: error creating file record: %v", err)
			response.JSON(c, "Failed to create file record", http.StatusInternalServerError, nil, errors.ErrInternalServerError)
			return
		}

		// Handle file based on type
		switch fileType {
		case models.FileTypeCSV:
			// For CSV, we process immediately but asynchronously
			go s.processCSVFile(uploadedFile.ID, file, header)
			
		case models.FileTypePDF, models.FileTypeImage:
			// For PDF/Image, upload to S3 first, then process asynchronously
			go s.processMediaFile(uploadedFile.ID, file, header, userUUID, fileType)
		}

		// Return success response immediately
		response.JSON(c, "File uploaded successfully and is being processed", http.StatusAccepted, gin.H{
			"file_id":   uploadedFile.ID,
			"file_name": uploadedFile.FileName,
			"file_type": uploadedFile.FileType,
			"status":    uploadedFile.Status,
		}, nil)
	}
}

// processCSVFile processes a CSV file asynchronously
func (s *Server) processCSVFile(fileID uuid.UUID, file io.Reader, header interface{}) {
	log.Printf("Starting async CSV processing for file: %s", fileID)

	// Read file content into buffer (since file might be closed)
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, file); err != nil {
		log.Printf("processCSVFile: error reading file: %v", err)
		s.markFileAsFailed(fileID, fmt.Sprintf("Failed to read file: %v", err))
		return
	}

	// Process CSV
	csvProcessor := processors.NewCSVProcessor(s.UploadRepository)
	if err := csvProcessor.ProcessBankStatementCSV(fileID, &buf); err != nil {
		log.Printf("processCSVFile: error processing CSV: %v", err)
		return
	}

	log.Printf("CSV processing completed successfully for file: %s", fileID)
}

// processMediaFile uploads media to S3 and processes it asynchronously
func (s *Server) processMediaFile(fileID uuid.UUID, file io.Reader, header interface{}, userID uuid.UUID, fileType models.FileType) {
	log.Printf("Starting async media processing for file: %s", fileID)

	// Read file content into buffer
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, file); err != nil {
		log.Printf("processMediaFile: error reading file: %v", err)
		s.markFileAsFailed(fileID, fmt.Sprintf("Failed to read file: %v", err))
		return
	}

	// Initialize S3 storage
	s3Storage, err := storage.NewS3Storage(s.Config)
	if err != nil {
		log.Printf("processMediaFile: error initializing S3: %v", err)
		s.markFileAsFailed(fileID, fmt.Sprintf("Failed to initialize storage: %v", err))
		return
	}

	// Upload file to S3
	// Convert buffer to io.Reader and create a mock multipart.FileHeader
	fileReader := bytes.NewReader(buf.Bytes())
	
	// Get filename from header if available
	var filename string
	if h, ok := header.(*multipart.FileHeader); ok {
		filename = h.Filename
	} else {
		// Generate filename based on file type
		ext := ".bin"
		if fileType == models.FileTypePDF {
			ext = ".pdf"
		} else if fileType == models.FileTypeImage {
			ext = ".jpg"
		}
		filename = fmt.Sprintf("file-%s%s", fileID.String(), ext)
	}
	
	// Create a temporary multipart.FileHeader for S3 upload
	tempHeader := &multipart.FileHeader{
		Filename: filename,
		Size:     int64(buf.Len()),
	}
	
	// Upload to S3
	fileURL, err := s3Storage.UploadFile(fileReader, tempHeader, userID)
	if err != nil {
		log.Printf("processMediaFile: error uploading to S3: %v", err)
		s.markFileAsFailed(fileID, fmt.Sprintf("Failed to upload to S3: %v", err))
		return
	}

	log.Printf("File uploaded to S3: %s", fileURL)

	// Update file record with URL
	uploadedFile, err := s.UploadRepository.GetUploadedFileByID(fileID)
	if err != nil {
		log.Printf("processMediaFile: error getting file record: %v", err)
		return
	}

	uploadedFile.FileURL = fileURL
	if err := s.UploadRepository.UpdateUploadedFile(uploadedFile); err != nil {
		log.Printf("processMediaFile: error updating file URL: %v", err)
	}

	// Process media with OCR
	mediaProcessor := processors.NewMediaProcessor(s.UploadRepository)
	if err := mediaProcessor.ProcessMediaFile(fileID, fileURL, fileType); err != nil {
		log.Printf("processMediaFile: error processing media: %v", err)
		return
	}

	log.Printf("Media processing completed successfully for file: %s", fileID)
}

// markFileAsFailed updates a file's status to failed
func (s *Server) markFileAsFailed(fileID uuid.UUID, errorMsg string) {
	file, err := s.UploadRepository.GetUploadedFileByID(fileID)
	if err != nil {
		log.Printf("markFileAsFailed: error getting file: %v", err)
		return
	}

	file.Status = models.FileStatusFailed
	file.ErrorMsg = errorMsg
	if err := s.UploadRepository.UpdateUploadedFile(file); err != nil {
		log.Printf("markFileAsFailed: error updating file: %v", err)
	}
}

// handleGetUploadStatus returns the status of an uploaded file
func (s *Server) handleGetUploadStatus() gin.HandlerFunc {
	return func(c *gin.Context) {
		fileIDStr := c.Param("id")
		fileID, err := uuid.Parse(fileIDStr)
		if err != nil {
			response.JSON(c, "Invalid file ID", http.StatusBadRequest, nil, errors.New("Invalid UUID format", http.StatusBadRequest))
			return
		}

		file, err := s.UploadRepository.GetUploadedFileByID(fileID)
		if err != nil {
			response.JSON(c, "File not found", http.StatusNotFound, nil, errors.New("File not found", http.StatusNotFound))
			return
		}

		response.JSON(c, "File status retrieved", http.StatusOK, file, nil)
	}
}

// handleGetUserUploads returns all uploads for the authenticated user
func (s *Server) handleGetUserUploads() gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDValue, exists := c.Get("userID")
		if !exists {
			response.JSON(c, "Unauthorized", http.StatusUnauthorized, nil, errors.New("User not authenticated", http.StatusUnauthorized))
			return
		}

		userID, ok := userIDValue.(uint)
		if !ok {
			response.JSON(c, "Invalid user ID", http.StatusInternalServerError, nil, errors.ErrInternalServerError)
			return
		}

		// Create deterministic UUID from user ID
		userUUID := uuid.NewSHA1(uuid.NameSpaceOID, []byte(fmt.Sprintf("user-%d", userID)))

		// Get pagination parameters
		limit := 20
		offset := 0

		files, err := s.UploadRepository.GetUploadedFilesByUserID(userUUID, limit, offset)
		if err != nil {
			response.JSON(c, "Failed to retrieve uploads", http.StatusInternalServerError, nil, errors.ErrInternalServerError)
			return
		}

		response.JSON(c, "Uploads retrieved successfully", http.StatusOK, files, nil)
	}
}

// isValidFileType checks if the file type is valid
func isValidFileType(fileType models.FileType) bool {
	switch fileType {
	case models.FileTypeCSV, models.FileTypePDF, models.FileTypeImage:
		return true
	default:
		return false
	}
}

// validateFileExtension validates that the file extension matches the declared type
func validateFileExtension(filename string, fileType models.FileType) error {
	ext := strings.ToLower(filepath.Ext(filename))

	switch fileType {
	case models.FileTypeCSV:
		if ext != ".csv" {
			return fmt.Errorf("CSV files must have .csv extension")
		}
	case models.FileTypePDF:
		if ext != ".pdf" {
			return fmt.Errorf("PDF files must have .pdf extension")
		}
	case models.FileTypeImage:
		validImageExts := map[string]bool{
			".jpg":  true,
			".jpeg": true,
			".png":  true,
			".gif":  true,
			".bmp":  true,
			".webp": true,
		}
		if !validImageExts[ext] {
			return fmt.Errorf("image files must have valid image extension (.jpg, .jpeg, .png, .gif, .bmp, .webp)")
		}
	}

	return nil
}
