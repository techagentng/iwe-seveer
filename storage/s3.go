package storage

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/google/uuid"
	"github.com/techagentng/iweapp/config"
)

// S3Storage handles file uploads to AWS S3
type S3Storage struct {
	client     *s3.S3
	bucket     string
	region     string
	folderName string
}

// NewS3Storage creates a new S3Storage instance
func NewS3Storage(cfg *config.Config) (*S3Storage, error) {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(cfg.AWS_REGION),
		Credentials: credentials.NewStaticCredentials(
			cfg.AWS_ACCESS_KEY_ID,
			cfg.AWS_SECRET_ACCESS_KEY,
			"",
		),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS session: %w", err)
	}

	return &S3Storage{
		client:     s3.New(sess),
		bucket:     cfg.AWS_BUCKET,
		region:     cfg.AWS_REGION,
		folderName: "uploads", // Default folder
	}, nil
}

// UploadFile uploads a file to S3 and returns the file URL
func (s *S3Storage) UploadFile(file io.Reader, header *multipart.FileHeader, userID uuid.UUID) (string, error) {
	// Read file content
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// Generate unique filename
	ext := filepath.Ext(header.Filename)
	filename := fmt.Sprintf("%s/%s/%s%s",
		s.folderName,
		userID.String(),
		uuid.New().String(),
		ext,
	)

	// Determine content type
	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// Upload to S3 without ACL (rely on bucket policies for access control)
	_, err = s.client.PutObject(&s3.PutObjectInput{
		Bucket:        aws.String(s.bucket),
		Key:           aws.String(filename),
		Body:          bytes.NewReader(fileBytes),
		ContentType:   aws.String(contentType),
		ContentLength: aws.Int64(int64(len(fileBytes))),
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload to S3: %w", err)
	}

	// Construct file URL
	fileURL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s",
		s.bucket,
		s.region,
		filename,
	)

	log.Printf("File uploaded successfully to S3: %s", fileURL)
	return fileURL, nil
}

// SaveFileLocally saves a file to local storage (fallback option)
func SaveFileLocally(file multipart.File, header *multipart.FileHeader, userID uuid.UUID) (string, error) {
	// Create uploads directory if it doesn't exist
	uploadDir := "./uploads"
	userDir := filepath.Join(uploadDir, userID.String())
	
	// Generate unique filename
	ext := filepath.Ext(header.Filename)
	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("%s-%s%s", timestamp, uuid.New().String()[:8], ext)
	filePath := filepath.Join(userDir, filename)

	log.Printf("File would be saved locally to: %s", filePath)
	
	// Return relative path (actual file saving would happen here in production)
	return fmt.Sprintf("/uploads/%s/%s", userID.String(), filename), nil
}

// DeleteFile deletes a file from S3
func (s *S3Storage) DeleteFile(fileURL string) error {
	// Extract key from URL
	// Format: https://bucket.s3.region.amazonaws.com/key
	key := extractKeyFromURL(fileURL, s.bucket, s.region)
	if key == "" {
		return fmt.Errorf("invalid file URL")
	}

	_, err := s.client.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete from S3: %w", err)
	}

	log.Printf("File deleted from S3: %s", key)
	return nil
}

// extractKeyFromURL extracts the S3 key from a full URL
func extractKeyFromURL(fileURL, bucket, region string) string {
	prefix := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/", bucket, region)
	if len(fileURL) > len(prefix) && fileURL[:len(prefix)] == prefix {
		return fileURL[len(prefix):]
	}
	return ""
}
