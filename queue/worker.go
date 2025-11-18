package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/techagentng/iweapp/db"
	"github.com/techagentng/iweapp/models"
	"github.com/techagentng/iweapp/processors"
	"github.com/techagentng/iweapp/websocket"
	"gorm.io/gorm"
)

// Worker processes jobs from the queue
type Worker struct {
	id               int
	queueManager     *QueueManager
	uploadRepo       db.UploadRepository
	wsHub            *websocket.Hub
	mediaProcessor   *processors.MediaProcessor
	csvProcessor     *processors.CSVProcessor
	db               *gorm.DB
	stopChan         chan bool
	ctx              context.Context
}

// WorkerConfig holds configuration for creating a worker
type WorkerConfig struct {
	ID             int
	QueueManager   *QueueManager
	UploadRepo     db.UploadRepository
	WSHub          *websocket.Hub
	MediaProcessor *processors.MediaProcessor
	CSVProcessor   *processors.CSVProcessor
	DB             *gorm.DB
}

// NewWorker creates a new worker instance
func NewWorker(config WorkerConfig) *Worker {
	return &Worker{
		id:             config.ID,
		queueManager:   config.QueueManager,
		uploadRepo:     config.UploadRepo,
		wsHub:          config.WSHub,
		mediaProcessor: config.MediaProcessor,
		csvProcessor:   config.CSVProcessor,
		db:             config.DB,
		stopChan:       make(chan bool),
		ctx:            context.Background(),
	}
}

// Start begins processing jobs
func (w *Worker) Start() {
	log.Printf("🔧 Worker #%d started", w.id)

	for {
		select {
		case <-w.stopChan:
			log.Printf("🛑 Worker #%d stopped", w.id)
			return
		default:
			w.processNextJob()
		}
	}
}

// Stop gracefully stops the worker
func (w *Worker) Stop() {
	w.stopChan <- true
}

// processNextJob dequeues and processes the next job
func (w *Worker) processNextJob() {
	// Dequeue job with 5 second timeout
	job, err := w.queueManager.DequeueJob(5 * time.Second)
	if err != nil {
		log.Printf("Worker #%d: error dequeuing job: %v", w.id, err)
		return
	}

	if job == nil {
		// No jobs available, continue
		return
	}

	log.Printf("🔧 Worker #%d processing job: %s", w.id, job.ID)

	// Process the job
	if err := w.processJob(job); err != nil {
		log.Printf("❌ Worker #%d: job %s failed: %v", w.id, job.ID, err)
		w.handleJobFailure(job, err)
	} else {
		log.Printf("✅ Worker #%d: job %s completed", w.id, job.ID)
	}
}

// processJob handles the complete job processing pipeline
func (w *Worker) processJob(job *models.ProcessingJob) error {
	// Update job status to processing
	if err := w.updateJobStatus(job, models.JobStatusProcessing, 10, "Starting job processing..."); err != nil {
		return err
	}

	// Get the uploaded file
	uploadedFile, err := w.uploadRepo.GetUploadedFileByID(job.FileID)
	if err != nil {
		return fmt.Errorf("failed to get uploaded file: %w", err)
	}

	// Step 1: Extract text based on file type
	extractedText, err := w.extractText(job, uploadedFile)
	if err != nil {
		return fmt.Errorf("text extraction failed: %w", err)
	}

	job.ExtractedText = extractedText
	w.updateJobStatus(job, models.JobStatusProcessing, 60, "Text extracted successfully")

	// Step 2: Process with AI (placeholder for now)
	aiResponse, err := w.processWithAI(job, extractedText)
	if err != nil {
		return fmt.Errorf("AI processing failed: %w", err)
	}

	job.AIResponse = aiResponse
	w.updateJobStatus(job, models.JobStatusProcessing, 90, "AI analysis complete")

	// Step 3: Mark job as completed
	job.Status = models.JobStatusCompleted
	job.Progress = 100
	now := time.Now()
	job.CompletedAt = &now

	// Save to database
	if err := w.uploadRepo.UpdateProcessingJob(job); err != nil {
		return fmt.Errorf("failed to update job in database: %w", err)
	}

	// Update in Redis and notify via WebSocket
	if err := w.queueManager.UpdateJob(job); err != nil {
		log.Printf("Warning: failed to update job in Redis: %v", err)
	}

	w.notifyJobComplete(job)

	return nil
}

// extractText extracts text from the uploaded file
func (w *Worker) extractText(job *models.ProcessingJob, file *models.UploadedFile) (string, error) {
	w.updateJobStatus(job, models.JobStatusProcessing, 20, "Extracting text from file...")

	switch file.FileType {
	case models.FileTypeCSV:
		// CSV files are already processed during upload
		// Retrieve the bank statements
		statements, err := w.uploadRepo.GetBankStatementsByFileID(file.ID)
		if err != nil {
			return "", fmt.Errorf("failed to get bank statements: %w", err)
		}

		// Convert statements to text
		text := w.csvProcessor.ConvertStatementsToText(statements)
		return text, nil

	case models.FileTypePDF, models.FileTypeImage:
		// Use media processor for OCR
		w.updateJobStatus(job, models.JobStatusProcessing, 30, "Running OCR on document...")

		// Get document chunks (already processed during upload)
		chunks, err := w.uploadRepo.GetDocumentChunksByFileID(file.ID)
		if err != nil {
			return "", fmt.Errorf("failed to get document chunks: %w", err)
		}

		// Combine chunks into full text
		var fullText string
		for _, chunk := range chunks {
			fullText += chunk.Content + "\n\n"
		}

		return fullText, nil

	default:
		return "", fmt.Errorf("unsupported file type: %s", file.FileType)
	}
}

// processWithAI processes the extracted text with AI
func (w *Worker) processWithAI(job *models.ProcessingJob, extractedText string) (string, error) {
	w.updateJobStatus(job, models.JobStatusProcessing, 70, "Analyzing with AI...")

	// TODO: Integrate with OpenAI GPT-4o-mini
	// For now, return a placeholder response

	// Simulate AI processing time
	time.Sleep(2 * time.Second)

	// Stream AI response chunks via WebSocket
	response := w.generatePlaceholderAIResponse(job, extractedText)

	return response, nil
}

// generatePlaceholderAIResponse generates a mock AI response
// TODO: Replace with actual OpenAI integration
func (w *Worker) generatePlaceholderAIResponse(job *models.ProcessingJob, text string) string {
	prompt := job.Prompt
	if prompt == "" {
		prompt = "Analyze this document"
	}

	// Mock response
	response := fmt.Sprintf(`Based on the document analysis for your question: "%s"

Document Summary:
- Total characters: %d
- Estimated word count: ~%d words

Key Observations:
1. The document has been successfully processed and extracted.
2. Text extraction completed using OCR technology.
3. Content is ready for detailed analysis.

Note: This is a placeholder response. Real AI analysis will be integrated with OpenAI GPT-4o-mini.

Next Steps:
- Review the extracted text below
- Ask specific questions about the content
- Request detailed analysis on specific sections

Extracted Text Preview:
%s

---
[AI Analysis Placeholder - OpenAI Integration Pending]`,
		prompt,
		len(text),
		len(text)/5, // Rough word count estimate
		w.truncateText(text, 500),
	)

	// Stream response in chunks via WebSocket
	w.streamAIResponse(job, response)

	return response
}

// streamAIResponse streams AI response chunks to the user via WebSocket
func (w *Worker) streamAIResponse(job *models.ProcessingJob, response string) {
	// Split response into chunks for streaming effect
	chunkSize := 50
	for i := 0; i < len(response); i += chunkSize {
		end := i + chunkSize
		if end > len(response) {
			end = len(response)
		}

		chunk := response[i:end]

		// Send chunk via WebSocket
		message := map[string]interface{}{
			"type":    "ai_chunk",
			"job_id":  job.ID.String(),
			"chunk":   chunk,
			"partial": response[:end],
		}

		w.sendWebSocketMessage(job.UserID, message)

		// Small delay to simulate streaming
		time.Sleep(50 * time.Millisecond)
	}
}

// updateJobStatus updates job status and sends WebSocket notification
func (w *Worker) updateJobStatus(job *models.ProcessingJob, status models.JobStatus, progress int, message string) error {
	job.Status = status
	job.Progress = progress

	// Update timestamps
	now := time.Now()
	if status == models.JobStatusProcessing && job.StartedAt == nil {
		job.StartedAt = &now
	}

	// Update in Redis
	if err := w.queueManager.UpdateJob(job); err != nil {
		log.Printf("Warning: failed to update job in Redis: %v", err)
	}

	// Update in database
	if err := w.uploadRepo.UpdateProcessingJob(job); err != nil {
		log.Printf("Warning: failed to update job in database: %v", err)
	}

	// Send WebSocket notification
	wsMessage := map[string]interface{}{
		"type":     "job_update",
		"job_id":   job.ID.String(),
		"status":   string(status),
		"progress": progress,
		"message":  message,
	}

	w.sendWebSocketMessage(job.UserID, wsMessage)

	log.Printf("📊 Job %s: %d%% - %s", job.ID, progress, message)

	return nil
}

// handleJobFailure handles job failures
func (w *Worker) handleJobFailure(job *models.ProcessingJob, err error) {
	job.Status = models.JobStatusFailed
	job.ErrorMsg = err.Error()
	job.RetryCount++

	now := time.Now()
	job.CompletedAt = &now

	// Update in database
	if updateErr := w.uploadRepo.UpdateProcessingJob(job); updateErr != nil {
		log.Printf("Error updating failed job in database: %v", updateErr)
	}

	// Update in Redis
	if updateErr := w.queueManager.UpdateJob(job); updateErr != nil {
		log.Printf("Error updating failed job in Redis: %v", updateErr)
	}

	// Notify user via WebSocket
	wsMessage := map[string]interface{}{
		"type":    "job_failed",
		"job_id":  job.ID.String(),
		"status":  "failed",
		"error":   err.Error(),
		"retries": job.RetryCount,
	}

	w.sendWebSocketMessage(job.UserID, wsMessage)

	// Retry if eligible
	if job.CanRetry() {
		log.Printf("🔄 Retrying job %s (attempt %d/3)", job.ID, job.RetryCount)
		// Re-enqueue job
		job.Status = models.JobStatusQueued
		job.ErrorMsg = ""
		if err := w.queueManager.EnqueueJob(job); err != nil {
			log.Printf("Error re-enqueueing job: %v", err)
		}
	}
}

// notifyJobComplete sends completion notification via WebSocket
func (w *Worker) notifyJobComplete(job *models.ProcessingJob) {
	message := map[string]interface{}{
		"type":        "job_completed",
		"job_id":      job.ID.String(),
		"status":      "completed",
		"progress":    100,
		"ai_response": job.AIResponse,
		"duration":    job.Duration().Seconds(),
	}

	w.sendWebSocketMessage(job.UserID, message)
}

// sendWebSocketMessage sends a message to a user via WebSocket
func (w *Worker) sendWebSocketMessage(userID uuid.UUID, message interface{}) {
	if w.wsHub == nil {
		return
	}

	// Marshal message to JSON
	payload, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling WebSocket message: %v", err)
		return
	}

	// Broadcast to user
	w.wsHub.BroadcastToUser(userID, payload)
}

// truncateText truncates text to a maximum length
func (w *Worker) truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "..."
}
