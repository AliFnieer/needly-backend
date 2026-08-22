package main

import (
	"log"

	"github.com/AliFnieer/needly-backend/internal/config"
	"github.com/AliFnieer/needly-backend/internal/database"
)

func main() {
	// Load configuration from environment variables
	cfg := config.Load()

	// Initialize PostgreSQL database connection
	db, err := database.InitPostgres(cfg)
	if err != nil {
		log.Fatalf("failed to initialize database: %v", err)
	}

	// Seed demo users
	if err := database.SeedUsers(db, nil); err != nil {
		log.Fatalf("failed to seed users: %v", err)
	}

	// Seed default categories
	if err := database.SeedCategories(db, nil); err != nil {
		log.Fatalf("failed to seed categories: %v", err)
	}

	log.Println("seeding complete")
}
