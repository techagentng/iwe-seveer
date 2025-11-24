package main

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/techagentng/iweapp/config"
	"github.com/techagentng/iweapp/db"
	"github.com/techagentng/iweapp/mailingservices"
	"github.com/techagentng/iweapp/queue"
	"github.com/techagentng/iweapp/server"
	"github.com/techagentng/iweapp/services"
	"github.com/techagentng/iweapp/services/ai"
	"github.com/techagentng/iweapp/websocket"
	"github.com/google/uuid"
)

func main() {
	// Load configuration
	conf, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	// Initialize Redis
	redisClient, err := db.InitRedis(conf)
	if err != nil {
		log.Fatalf("Failed to initialize Redis: %v", err)
	}
	defer db.CloseRedis(redisClient)

	// Initialize Mailgun client
	mailgunClient := &mailingservices.Mailgun{}
	mailgunClient.Init()

	// Initialize database
	gormDB := db.GetDB(conf)

	// Seed roles
	if err := db.SeedRoles(gormDB.DB); err != nil {
		log.Fatalf("error seeding roles: %v", err)
	}

	// Repositories
	authRepo := db.NewAuthRepo(gormDB)
	uploadRepo := db.NewUploadRepository(gormDB)

	// Services
	authService := services.NewAuthService(authRepo, conf)
	
	// OpenAI Service
	aiService := ai.NewOpenAIService(conf.OpenAIAPIKey)
	
	// Queue Manager
	queueManager := queue.NewQueueManager(redisClient)
	log.Println("✅ Queue Manager initialized")
	
	// WebSocket Hub
	wsHub := websocket.NewHub()
	go wsHub.Run() // Start hub in background
	log.Println("✅ WebSocket Hub started")

	// Wire AI streaming over WebSocket using Hub.OnMessage
	type wsClientMessage struct {
		Type         string `json:"type"`
		MessageID    string `json:"messageId"`
		Content      string `json:"content"`
		JobID        string `json:"jobId"`
		Filename     string `json:"filename"`
		FileType     string `json:"fileType"`
		Timestamp    int64  `json:"timestamp"`
		DocumentText string `json:"documentText"`
	}
	// Track cancel functions per message
	var aiCancels sync.Map // key: messageId, val: context.CancelFunc

	wsHub.OnMessage = func(userID uuid.UUID, payload []byte) {
		var msg wsClientMessage
		if err := json.Unmarshal(payload, &msg); err != nil {
			return
		}
		switch msg.Type {
		case "user_message":
			if msg.MessageID == "" || msg.Content == "" {
				// send error back if desired
				return
			}
			// Ack and typing indicator
			if b, err := json.Marshal(map[string]any{"type":"ack", "messageId": msg.MessageID}); err == nil {
				wsHub.SendToUser(userID, b)
			}
			if b, err := json.Marshal(map[string]any{"type":"assistant_typing", "messageId": msg.MessageID}); err == nil {
				wsHub.SendToUser(userID, b)
			}
			// Start streaming
			ctx, cancel := context.WithCancel(context.Background())
			aiCancels.Store(msg.MessageID, cancel)
			go func(messageID string, prompt string, initialDoc string, jobIDStr string, uid uuid.UUID) {
				var full string
				// Resolve document text: prefer provided DocumentText; otherwise, load by jobId
				docText := initialDoc
				if docText == "" && jobIDStr != "" {
					if jID, err := uuid.Parse(jobIDStr); err == nil {
						if job, err := uploadRepo.GetProcessingJobByID(jID); err == nil && job != nil {
							if job.ExtractedText != "" {
								docText = job.ExtractedText
							} else {
								// fallback: reconstruct from chunks by fileID
								if chunks, err := uploadRepo.GetDocumentChunksByFileID(job.FileID); err == nil {
									var combined string
									for _, ch := range chunks {
										combined += ch.Content + "\n\n"
									}
									docText = combined
								}
							}
						}
					}
				}

				streamErr := aiService.AnalyzeDocumentStream(ctx, docText, prompt, func(chunk string) {
					full += chunk
					// stream_chunk (isLastChunk=false)
					resp := map[string]any{
						"type":       "stream_chunk",
						"messageId":  messageID,
						"content":    chunk,
						"isLastChunk": false,
					}
					if b, err := json.Marshal(resp); err == nil {
						wsHub.SendToUser(uid, b)
					}
				})
				// Final message
				if streamErr != nil {
					if b, err := json.Marshal(map[string]any{"type":"error", "code":"ai_failed", "message": streamErr.Error(), "messageId": messageID}); err == nil {
						wsHub.SendToUser(uid, b)
					}
				} else {
					final := map[string]any{
						"type":        "assistant_message",
						"messageId":   messageID,
						"content":     full,
						"isComplete":  true,
						"timestamp":   time.Now().UnixMilli(),
					}
					if b, err := json.Marshal(final); err == nil {
						wsHub.SendToUser(uid, b)
					}
				}
				aiCancels.Delete(messageID)
			}(msg.MessageID, msg.Content, msg.DocumentText, msg.JobID, userID)

		case "cancel":
			if v, ok := aiCancels.Load(msg.MessageID); ok {
				if cancel, ok2 := v.(context.CancelFunc); ok2 {
					cancel()
				}
				aiCancels.Delete(msg.MessageID)
				if b, err := json.Marshal(map[string]any{"type":"error", "code":"canceled", "message":"Request canceled", "messageId": msg.MessageID}); err == nil {
					wsHub.SendToUser(userID, b)
				}
			}

		case "file_uploaded":
			// Optional: ack file notification; real job updates should be sent by workers as they progress
			if msg.JobID != "" {
				if b, err := json.Marshal(map[string]any{"type":"ack", "jobId": msg.JobID}); err == nil {
					wsHub.SendToUser(userID, b)
				}
			}
		}
	}
	
	// Worker Pool
	workerPool := queue.NewWorkerPool(queue.WorkerPoolConfig{
		NumWorkers:   3, // Start with 3 workers
		QueueManager: queueManager,
		UploadRepo:   uploadRepo,
		WSHub:        wsHub,
		AIService:    aiService,
		DB:           gormDB.DB,
	})
	go workerPool.Start() // Start workers in background
	log.Println("✅ Worker Pool started with 3 workers")

	// Server setup
	s := &server.Server{
		Mail:             mailgunClient,
		Config:           conf,
		AuthRepository:   authRepo,
		AuthService:      authService,
		UploadRepository: uploadRepo,
		DB:               gormDB.DB,
		RedisClient:      redisClient,
		QueueManager:     queueManager,
		WSHub:            wsHub,
		AIService:        aiService,
	}

	// Start server
	s.Start()
}
