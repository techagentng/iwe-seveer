package queue

import (
	"log"
	"sync"

	"github.com/techagentng/iweapp/db"
	"github.com/techagentng/iweapp/processors"
	"github.com/techagentng/iweapp/services/ai"
	"github.com/techagentng/iweapp/websocket"
	"gorm.io/gorm"
)

// WorkerPool manages a pool of workers
type WorkerPool struct {
	workers      []*Worker
	numWorkers   int
	queueManager *QueueManager
	uploadRepo   db.UploadRepository
	wsHub        *websocket.Hub
	aiService    *ai.OpenAIService
	gormDB       *gorm.DB
	stopChan     chan bool
	wg           sync.WaitGroup
}

// WorkerPoolConfig holds configuration for the worker pool
type WorkerPoolConfig struct {
	NumWorkers   int
	QueueManager *QueueManager
	UploadRepo   db.UploadRepository
	WSHub        *websocket.Hub
	AIService    *ai.OpenAIService
	DB           *gorm.DB
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(config WorkerPoolConfig) *WorkerPool {
	if config.NumWorkers <= 0 {
		config.NumWorkers = 3 // Default to 3 workers
	}

	return &WorkerPool{
		workers:      make([]*Worker, 0, config.NumWorkers),
		numWorkers:   config.NumWorkers,
		queueManager: config.QueueManager,
		uploadRepo:   config.UploadRepo,
		wsHub:        config.WSHub,
		aiService:    config.AIService,
		gormDB:       config.DB,
		stopChan:     make(chan bool),
	}
}

// Start initializes and starts all workers in the pool
func (wp *WorkerPool) Start() {
	log.Printf("🚀 Starting worker pool with %d workers", wp.numWorkers)

	for i := 0; i < wp.numWorkers; i++ {
		// Create processors for each worker
		mediaProcessor := processors.NewMediaProcessor(wp.uploadRepo)
		csvProcessor := processors.NewCSVProcessor(wp.uploadRepo)

		// Create worker
		worker := NewWorker(WorkerConfig{
			ID:             i + 1,
			QueueManager:   wp.queueManager,
			UploadRepo:     wp.uploadRepo,
			WSHub:          wp.wsHub,
			MediaProcessor: mediaProcessor,
			CSVProcessor:   csvProcessor,
			AIService:      wp.aiService,
			DB:             wp.gormDB,
		})

		wp.workers = append(wp.workers, worker)

		// Start worker in goroutine
		wp.wg.Add(1)
		go func(w *Worker) {
			defer wp.wg.Done()
			w.Start()
		}(worker)
	}

	log.Printf("✅ Worker pool started with %d workers", wp.numWorkers)
}

// Stop gracefully stops all workers in the pool
func (wp *WorkerPool) Stop() {
	log.Println("🛑 Stopping worker pool...")

	// Stop all workers
	for _, worker := range wp.workers {
		worker.Stop()
	}

	// Wait for all workers to finish
	wp.wg.Wait()

	log.Println("✅ Worker pool stopped")
}

// GetQueueLength returns the current queue length
func (wp *WorkerPool) GetQueueLength() (int64, error) {
	return wp.queueManager.GetQueueLength()
}

// GetStats returns worker pool statistics
func (wp *WorkerPool) GetStats() map[string]interface{} {
	queueLength, _ := wp.GetQueueLength()

	return map[string]interface{}{
		"num_workers":  wp.numWorkers,
		"queue_length": queueLength,
		"active":       len(wp.workers),
	}
}
