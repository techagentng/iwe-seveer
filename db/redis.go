package db

import (
	"context"
	"fmt"
	"log"

	"github.com/redis/go-redis/v9"
	"github.com/techagentng/iweapp/config"
)

// InitRedis initializes and returns a Redis client
func InitRedis(cfg *config.Config) (*redis.Client, error) {
	ctx := context.Background()

	// Create Redis client
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.GetRedisAddr(),
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	// Test connection
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	log.Printf("✅ Connected to Redis at %s", cfg.GetRedisAddr())
	return client, nil
}

// CloseRedis gracefully closes the Redis connection
func CloseRedis(client *redis.Client) error {
	if client != nil {
		return client.Close()
	}
	return nil
}
