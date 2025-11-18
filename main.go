package main

import (
	"log"

	"github.com/techagentng/iweapp/config"
	"github.com/techagentng/iweapp/db"
	"github.com/techagentng/iweapp/mailingservices"
	"github.com/techagentng/iweapp/queue"
	"github.com/techagentng/iweapp/server"
	"github.com/techagentng/iweapp/services"
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
	
	// Queue Manager
	queueManager := queue.NewQueueManager(redisClient)
	log.Println("✅ Queue Manager initialized")

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
	}

	// Start server
	s.Start()
}
