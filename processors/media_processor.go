package processors

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	vision "cloud.google.com/go/vision/apiv1"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/textract"
	"github.com/aws/aws-sdk-go-v2/service/textract/types"
	"github.com/google/uuid"
	"github.com/techagentng/iweapp/db"
	"github.com/techagentng/iweapp/models"
)

// OCRProvider represents the OCR service provider
type OCRProvider string

const (
	OCRProviderTextract OCRProvider = "aws_textract"  // For PDFs and bank statements
	OCRProviderVision   OCRProvider = "google_vision" // For images and handwritten notes
)

// MediaProcessor handles PDF and image file processing with hybrid OCR
type MediaProcessor struct {
	uploadRepo      db.UploadRepository
	textractEnabled bool // AWS Textract for PDFs
	visionEnabled   bool // Google Cloud Vision for images
}

// NewMediaProcessor creates a new MediaProcessor instance
func NewMediaProcessor(uploadRepo db.UploadRepository) *MediaProcessor {
	return &MediaProcessor{
		uploadRepo:      uploadRepo,
		textractEnabled: true, // AWS Textract enabled for PDFs
		visionEnabled:   true, // Google Cloud Vision enabled for images
	}
}

// ProcessMediaFile processes a PDF or image file asynchronously
func (p *MediaProcessor) ProcessMediaFile(fileID uuid.UUID, fileURL string, fileType models.FileType) error {
	log.Printf("Starting media processing for file: %s (type: %s)", fileID, fileType)

	// Update file status to processing
	file, err := p.uploadRepo.GetUploadedFileByID(fileID)
	if err != nil {
		return fmt.Errorf("failed to get file: %w", err)
	}

	file.Status = models.FileStatusProcessing
	if err := p.uploadRepo.UpdateUploadedFile(file); err != nil {
		log.Printf("Failed to update file status to processing: %v", err)
	}

	// Determine OCR provider based on file type and name
	ocrProvider := p.selectOCRProvider(file.FileName, fileType)
	log.Printf("Selected OCR provider: %s for file: %s", ocrProvider, file.FileName)

	// Perform OCR using the selected provider
	extractedText, err := p.performHybridOCR(fileURL, fileType, ocrProvider)
	if err != nil {
		return p.handleProcessingError(file, fmt.Errorf("OCR failed: %w", err))
	}

	// Chunk the extracted text
	chunks := p.chunkText(extractedText, 1000) // 1000 characters per chunk

	// Create document chunks
	documentChunks := make([]models.DocumentChunk, len(chunks))
	for i, chunk := range chunks {
		documentChunks[i] = models.DocumentChunk{
			FileID:     fileID,
			ChunkIndex: i,
			Content:    chunk,
		}
	}

	// Batch insert chunks
	if err := p.uploadRepo.BatchCreateDocumentChunks(documentChunks); err != nil {
		return p.handleProcessingError(file, fmt.Errorf("failed to insert chunks: %w", err))
	}

	// Update file status to completed
	now := time.Now()
	file.Status = models.FileStatusCompleted
	file.ProcessedAt = &now
	if err := p.uploadRepo.UpdateUploadedFile(file); err != nil {
		log.Printf("Failed to update file status to completed: %v", err)
	}

	log.Printf("Media processing completed for file: %s. Total chunks: %d", fileID, len(chunks))
	return nil
}

// selectOCRProvider determines which OCR provider to use based on file type and content
func (p *MediaProcessor) selectOCRProvider(fileName string, fileType models.FileType) OCRProvider {
	// Strategy:
	// - PDFs & files with "statement", "invoice", "receipt" → AWS Textract (better for structured documents)
	// - Images & files with "handwritten", "note", "scan" → Google Cloud Vision (better for handwriting)

	fileNameLower := strings.ToLower(fileName)

	// Check for bank statements, invoices, receipts (use Textract)
	if strings.Contains(fileNameLower, "statement") ||
		strings.Contains(fileNameLower, "bank") ||
		strings.Contains(fileNameLower, "invoice") ||
		strings.Contains(fileNameLower, "receipt") ||
		fileType == models.FileTypePDF {
		return OCRProviderTextract
	}

	// Check for handwritten notes, scans (use Vision)
	if strings.Contains(fileNameLower, "handwritten") ||
		strings.Contains(fileNameLower, "note") ||
		strings.Contains(fileNameLower, "scan") ||
		strings.Contains(fileNameLower, "signature") {
		return OCRProviderVision
	}

	// Default routing by file type
	if fileType == models.FileTypePDF {
		return OCRProviderTextract // PDFs → Textract
	}
	return OCRProviderVision // Images → Vision
}

// performHybridOCR performs OCR using the appropriate provider
func (p *MediaProcessor) performHybridOCR(fileURL string, fileType models.FileType, provider OCRProvider) (string, error) {
	log.Printf("Performing hybrid OCR on file: %s (type: %s, provider: %s)", fileURL, fileType, provider)

	switch provider {
	case OCRProviderTextract:
		if p.textractEnabled {
			return p.performTextractOCR(fileURL)
		}
		return p.mockTextractOCR(fileURL)

	case OCRProviderVision:
		if p.visionEnabled {
			return p.performVisionOCR(fileURL)
		}
		return p.mockVisionOCR(fileURL)

	default:
		return "", fmt.Errorf("unsupported OCR provider: %s", provider)
	}
}

// performTextractOCR performs OCR using AWS Textract (production implementation)
func (p *MediaProcessor) performTextractOCR(fileURL string) (string, error) {
	ctx := context.TODO()
	log.Printf("[AWS TEXTRACT] Processing document from %s", fileURL)

	// Extract S3 bucket and key from URL
	// Format: https://citizenx.s3.eu-north-1.amazonaws.com/uploads/user-id/file-id.pdf
	bucket, key, err := extractS3BucketAndKey(fileURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse S3 URL: %w", err)
	}

	log.Printf("[AWS TEXTRACT] Bucket: %s, Key: %s", bucket, key)

	// Force Textract to use eu-west-1 as it's not available in all regions
	textractRegion := "eu-west-1"
	log.Printf("[AWS TEXTRACT] Using region: %s", textractRegion)

	// Load the AWS configuration with the Textract region
	cfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(textractRegion),
	)
	if err != nil {
		return "", fmt.Errorf("unable to load AWS config: %w", err)
	}

	// Create Textract client with the configuration
	svc := textract.NewFromConfig(cfg)

	// Call DetectDocumentText for simple text extraction
	input := &textract.DetectDocumentTextInput{
		Document: &types.Document{
			S3Object: &types.S3Object{
				Bucket: aws.String(bucket),
				Name:   aws.String(key),
			},
		},
	}

	// Execute the Textract API call
	result, err := svc.DetectDocumentText(ctx, input)
	if err != nil {
		return "", fmt.Errorf("textract API error: %w", err)
	}

	// Extract text from blocks
	var extractedText strings.Builder
	lineCount := 0

	for _, block := range result.Blocks {
		if block.BlockType == types.BlockTypeLine {
			if block.Text != nil {
				extractedText.WriteString(*block.Text)
				extractedText.WriteString("\n")
				lineCount++
			}
		}
	}

	log.Printf("[AWS TEXTRACT] Extracted %d lines of text", lineCount)

	if extractedText.Len() == 0 {
		return "", fmt.Errorf("no text found in document")
	}

	return extractedText.String(), nil
}

// extractS3BucketAndKey extracts bucket and key from S3 URL
// Supports formats:
// 1. https://bucket.s3.region.amazonaws.com/key
// 2. https://s3.region.amazonaws.com/bucket/key
// 3. https://s3.amazonaws.com/bucket/key
func extractS3BucketAndKey(fileURL string) (string, string, error) {
	// Remove protocol if present
	fileURL = strings.TrimPrefix(fileURL, "https://")
	fileURL = strings.TrimPrefix(fileURL, "http://")

	parts := strings.Split(fileURL, "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid S3 URL format: %s", fileURL)
	}

	var bucket, key string
	host := parts[0]
	// Path-style URL if host starts with s3.* or s3-*
	if strings.HasPrefix(host, "s3.") || strings.HasPrefix(host, "s3-") || host == "s3.amazonaws.com" {
		if len(parts) < 3 {
			return "", "", fmt.Errorf("missing bucket or key in path-style URL: %s", fileURL)
		}
		bucket = parts[1]
		key = strings.Join(parts[2:], "/")
	} else {
		// Virtual-hosted style URL (bucket.s3.region.amazonaws.com/key)
		domainParts := strings.Split(host, ".")
		bucket = domainParts[0]
		key = strings.Join(parts[1:], "/")
	}

	if bucket == "" {
		return "", "", fmt.Errorf("could not extract bucket from URL: %s", fileURL)
	}

	// Remove any query parameters from the key
	if questionMark := strings.Index(key, "?"); questionMark != -1 {
		key = key[:questionMark]
	}

	return bucket, key, nil
}

// performVisionOCR performs OCR using Google Cloud Vision (production implementation)
func (p *MediaProcessor) performVisionOCR(fileURL string) (string, error) {
	log.Printf("[GOOGLE VISION] Processing image from %s", fileURL)

	ctx := context.Background()

	// Create Vision API client
	client, err := vision.NewImageAnnotatorClient(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to create vision client: %w", err)
	}
	defer client.Close()

	// Create image from URI (S3 URL)
	image := vision.NewImageFromURI(fileURL)

	// Detect document text (best for handwriting and dense text)
	annotation, err := client.DetectDocumentText(ctx, image, nil)
	if err != nil {
		return "", fmt.Errorf("vision API error: %w", err)
	}

	// Extract text from annotation
	if annotation == nil || annotation.Text == "" {
		return "", fmt.Errorf("no text found in image")
	}

	log.Printf("[GOOGLE VISION] Extracted %d characters of text", len(annotation.Text))

	return annotation.Text, nil
}

// mockTextractOCR simulates AWS Textract OCR (for PDFs and structured documents)
func (p *MediaProcessor) mockTextractOCR(fileURL string) (string, error) {
	log.Printf("[MOCK AWS TEXTRACT] Processing structured document from %s", fileURL)

	mockText := `
═══════════════════════════════════════════════════════════
              AWS TEXTRACT - MOCK OCR RESULT
═══════════════════════════════════════════════════════════

DOCUMENT TYPE: Bank Statement / Invoice / Receipt
OCR PROVIDER: AWS Textract (Optimized for structured documents)
PROCESSING DATE: ` + time.Now().Format("2006-01-02 15:04:05") + `

───────────────────────────────────────────────────────────
EXTRACTED CONTENT:
───────────────────────────────────────────────────────────

BANK STATEMENT
Account Number: 1234567890
Statement Period: January 1, 2024 - January 31, 2024

ACCOUNT SUMMARY:
  Opening Balance:        $10,000.00
  Total Deposits:         $5,500.00
  Total Withdrawals:      $3,200.00
  Closing Balance:        $12,300.00

TRANSACTION DETAILS:

Date       | Description              | Debit    | Credit   | Balance
-----------|--------------------------|----------|----------|----------
2024-01-05 | Salary Deposit          |          | 5,000.00 | 15,000.00
2024-01-08 | Grocery Store           | 150.00   |          | 14,850.00
2024-01-12 | Utility Payment         | 200.00   |          | 14,650.00
2024-01-15 | Online Transfer         | 500.00   |          | 14,150.00
2024-01-20 | ATM Withdrawal          | 300.00   |          | 13,850.00
2024-01-25 | Interest Credit         |          | 500.00   | 14,350.00
2024-01-28 | Restaurant              | 75.00    |          | 14,275.00
2024-01-30 | Gas Station             | 50.00    |          | 14,225.00

KEY-VALUE PAIRS DETECTED:
  - Account Holder: John Doe
  - Bank Name: Sample Bank Inc.
  - Branch: Downtown Branch
  - Currency: USD
  - Statement ID: STMT-2024-001

TABLES DETECTED: 1 table with 8 rows and 5 columns
FORMS DETECTED: Account summary form
CONFIDENCE SCORE: 98.5%

───────────────────────────────────────────────────────────
NOTE: This is a MOCK response. In production, AWS Textract
would provide actual extracted text with high accuracy for:
  ✓ Printed text in PDFs
  ✓ Scanned documents
  ✓ Forms and tables
  ✓ Key-value pairs
  ✓ Multi-column layouts
═══════════════════════════════════════════════════════════
	`

	return mockText, nil
}

// mockVisionOCR simulates Google Cloud Vision OCR (for images and handwritten notes)
func (p *MediaProcessor) mockVisionOCR(fileURL string) (string, error) {
	log.Printf("[MOCK GOOGLE VISION] Processing image/handwritten document from %s", fileURL)

	mockText := `
═══════════════════════════════════════════════════════════
          GOOGLE CLOUD VISION - MOCK OCR RESULT
═══════════════════════════════════════════════════════════

DOCUMENT TYPE: Image / Handwritten Note / Scan
OCR PROVIDER: Google Cloud Vision (Optimized for handwriting & images)
PROCESSING DATE: ` + time.Now().Format("2006-01-02 15:04:05") + `

───────────────────────────────────────────────────────────
EXTRACTED CONTENT:
───────────────────────────────────────────────────────────

HANDWRITTEN NOTE:

Dear Team,

Please review the following action items from today's meeting:

1. Update the quarterly financial report by Friday
2. Schedule follow-up meeting with the client
3. Review and approve the new budget proposal
4. Prepare presentation slides for next week

Important dates:
  - Project deadline: March 15, 2024
  - Client meeting: March 20, 2024
  - Budget review: March 25, 2024

Notes:
  • Increase marketing budget by 15%
  • Hire 2 new developers
  • Upgrade server infrastructure

Contact: john.doe@example.com
Phone: +1 (555) 123-4567

Signature: [Handwritten signature detected]

TEXT ANNOTATIONS:
  - Language detected: English (en)
  - Handwriting confidence: 94.2%
  - Total words: 87
  - Total characters: 512

DETECTED ENTITIES:
  - Dates: 3 instances
  - Email addresses: 1 instance
  - Phone numbers: 1 instance
  - Signatures: 1 instance

IMAGE PROPERTIES:
  - Dominant colors: Blue, Black, White
  - Image quality: High
  - Orientation: Portrait

───────────────────────────────────────────────────────────
NOTE: This is a MOCK response. In production, Google Cloud
Vision would provide actual extracted text with high accuracy for:
  ✓ Handwritten text (cursive & print)
  ✓ Printed text in images
  ✓ Multi-language support
  ✓ Rotated or skewed text
  ✓ Low-quality scans
  ✓ Signatures and symbols
═══════════════════════════════════════════════════════════
	`

	return mockText, nil
}

// chunkText splits text into chunks of specified size
func (p *MediaProcessor) chunkText(text string, chunkSize int) []string {
	// Remove excessive whitespace
	text = strings.TrimSpace(text)
	text = strings.Join(strings.Fields(text), " ")

	if len(text) == 0 {
		return []string{}
	}

	var chunks []string
	runes := []rune(text)

	for i := 0; i < len(runes); i += chunkSize {
		end := i + chunkSize
		if end > len(runes) {
			end = len(runes)
		}

		chunk := string(runes[i:end])

		// Try to break at word boundary if not at the end
		if end < len(runes) {
			lastSpace := strings.LastIndex(chunk, " ")
			if lastSpace > chunkSize/2 { // Only break if space is in latter half
				chunk = chunk[:lastSpace]
				i = i + lastSpace - chunkSize // Adjust index
			}
		}

		chunks = append(chunks, strings.TrimSpace(chunk))
	}

	return chunks
}

// handleProcessingError updates file status to failed and logs the error
func (p *MediaProcessor) handleProcessingError(file *models.UploadedFile, err error) error {
	log.Printf("Media processing failed for file %s: %v", file.ID, err)

	file.Status = models.FileStatusFailed
	file.ErrorMsg = err.Error()
	now := time.Now()
	file.ProcessedAt = &now

	if updateErr := p.uploadRepo.UpdateUploadedFile(file); updateErr != nil {
		log.Printf("Failed to update file status to failed: %v", updateErr)
	}

	return err
}

// IntegrateHybridOCR provides guidance for integrating the hybrid OCR system
func IntegrateHybridOCR() string {
	return `
═══════════════════════════════════════════════════════════
              HYBRID OCR INTEGRATION GUIDE
═══════════════════════════════════════════════════════════

STRATEGY:
  • PDFs & Bank Statements → AWS Textract
  • Images & Handwritten Notes → Google Cloud Vision

───────────────────────────────────────────────────────────
1. AWS TEXTRACT INTEGRATION (for PDFs & structured docs)
───────────────────────────────────────────────────────────

Installation:
  go get github.com/aws/aws-sdk-go/service/textract

Environment Variables:
  AWS_ACCESS_KEY_ID=your_access_key
  AWS_SECRET_ACCESS_KEY=your_secret_key
  AWS_REGION=us-east-1

Code Example:
  import (
      "github.com/aws/aws-sdk-go/aws"
      "github.com/aws/aws-sdk-go/aws/session"
      "github.com/aws/aws-sdk-go/service/textract"
  )

  sess := session.Must(session.NewSession(&aws.Config{
      Region: aws.String("us-east-1"),
  }))
  svc := textract.New(sess)

  input := &textract.DetectDocumentTextInput{
      Document: &textract.Document{
          S3Object: &textract.S3Object{
              Bucket: aws.String("your-bucket"),
              Name:   aws.String("file-key.pdf"),
          },
      },
  }

  result, err := svc.DetectDocumentText(input)
  // For advanced features (tables, forms, key-value pairs):
  // result, err := svc.AnalyzeDocument(input)

Best For:
  ✓ Bank statements
  ✓ Invoices and receipts
  ✓ Forms with tables
  ✓ Multi-column PDFs
  ✓ Printed documents

───────────────────────────────────────────────────────────
2. GOOGLE CLOUD VISION INTEGRATION (for images & handwriting)
───────────────────────────────────────────────────────────

Installation:
  go get cloud.google.com/go/vision/apiv1

Environment Variables:
  GOOGLE_APPLICATION_CREDENTIALS=/path/to/service-account-key.json

Code Example:
  import (
      "context"
      vision "cloud.google.com/go/vision/apiv1"
  )

  ctx := context.Background()
  client, err := vision.NewImageAnnotatorClient(ctx)
  if err != nil {
      return "", err
  }
  defer client.Close()

  image := vision.NewImageFromURI(fileURL)
  
  // For handwritten text:
  annotation, err := client.DetectDocumentText(ctx, image, nil)
  
  // For printed text:
  // texts, err := client.DetectText(ctx, image, nil, 10)

Best For:
  ✓ Handwritten notes
  ✓ Scanned images
  ✓ Photos of documents
  ✓ Signatures
  ✓ Low-quality scans
  ✓ Rotated/skewed images

───────────────────────────────────────────────────────────
3. ENABLE HYBRID OCR IN CODE
───────────────────────────────────────────────────────────

In media_processor.go, update NewMediaProcessor:

  func NewMediaProcessor(uploadRepo db.UploadRepository) *MediaProcessor {
      return &MediaProcessor{
          uploadRepo:      uploadRepo,
          textractEnabled: true,  // ← Enable AWS Textract
          visionEnabled:   true,  // ← Enable Google Cloud Vision
      }
  }

Then implement:
  - performTextractOCR() for real AWS Textract calls
  - performVisionOCR() for real Google Cloud Vision calls

───────────────────────────────────────────────────────────
4. ROUTING LOGIC
───────────────────────────────────────────────────────────

The system automatically routes files based on:

  AWS Textract (for structured documents):
    • File type: PDF
    • Filename contains: "statement", "bank", "invoice", "receipt"

  Google Cloud Vision (for handwriting & images):
    • File type: Image (jpg, png, etc.)
    • Filename contains: "handwritten", "note", "scan", "signature"

───────────────────────────────────────────────────────────
5. COST OPTIMIZATION
───────────────────────────────────────────────────────────

AWS Textract Pricing:
  • DetectDocumentText: $1.50 per 1,000 pages
  • AnalyzeDocument: $50-65 per 1,000 pages

Google Cloud Vision Pricing:
  • Text Detection: $1.50 per 1,000 images
  • Document Text Detection: $1.50 per 1,000 images

Tip: Use DetectDocumentText for simple text extraction,
     AnalyzeDocument only when you need tables/forms.

═══════════════════════════════════════════════════════════
	`
}
