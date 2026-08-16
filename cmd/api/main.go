package main

import (
	"log"

	"github.com/AliFnieer/needly-backend/internal/cache"
	"github.com/AliFnieer/needly-backend/internal/config"
	"github.com/AliFnieer/needly-backend/internal/database"
	"github.com/AliFnieer/needly-backend/internal/server"
)

func main() {
	// Load configuration from environment variables
	cfg := config.Load()

	// Initialize PostgreSQL database connection
	db, err := database.InitPostgres(cfg)
	if err != nil {
		log.Fatalf("failed to initialize database: %v", err)
	}

	// Initialize Redis cache connection
	redisClient, err := cache.InitRedis(cfg)
	if err != nil {
		log.Fatalf("failed to initialize redis: %v", err)
	}
	defer redisClient.Close()

	// Create and configure the Gin server
	srv := server.NewServer(cfg, db, redisClient)

	// Start the HTTP server
	if err := srv.Run(":" + cfg.Server.Port); err != nil {
		log.Fatalf("server failed to start: %v", err)
	}
}