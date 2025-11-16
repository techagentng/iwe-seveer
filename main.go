package main

import (
	"log"

	"github.com/techagentng/iweapp/config"
	"github.com/techagentng/iweapp/db"
	"github.com/techagentng/iweapp/mailingservices"
	"github.com/techagentng/iweapp/server"
	"github.com/techagentng/iweapp/services"
	"github.com/go-redis/redis/v8"
)

func main() {
	// Load configuration
	conf, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}



	// Initialize Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379", // Adjust to your Redis server address
		Password: "",               // No password by default
		DB:       0,                // Default DB
	})

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

	// Server setup
	s := &server.Server{
		Mail:             mailgunClient,
		Config:           conf,
		AuthRepository:   authRepo,
		AuthService:      authService,
		UploadRepository: uploadRepo,
		DB:               gormDB.DB,
		RedisClient:      redisClient,
	}

	// Start server
	s.Start()
}
