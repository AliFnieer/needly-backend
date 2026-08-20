package database

import (
	"fmt"
	"log"
	"time"

	"github.com/AliFnieer/needly-backend/internal/auth"
	"github.com/AliFnieer/needly-backend/internal/category"
	"github.com/AliFnieer/needly-backend/internal/config"
	"github.com/AliFnieer/needly-backend/internal/history"
	"github.com/AliFnieer/needly-backend/internal/household"
	"github.com/AliFnieer/needly-backend/internal/shoppingitem"
	"github.com/AliFnieer/needly-backend/internal/shoppinglist"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// InitPostgres establishes a connection to PostgreSQL using GORM.
func InitPostgres(cfg *config.Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=%s",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Name,
		cfg.Database.SSLMode,
		cfg.Database.TimeZone,
	)

	gormLogger := logger.Default.LogMode(logger.Warn)
	if cfg.Server.GinMode == "debug" {
		gormLogger = logger.Default.LogMode(logger.Info)
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB from gorm: %w", err)
	}

	// Connection pool settings (configurable for production tuning)
	sqlDB.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.Database.ConnMaxLifetimeMinutes) * time.Minute)

	// Verify connection
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Auto-migrate models in development only — production uses migration CLI
	if cfg.Server.GinMode == gin.ReleaseMode {
		log.Println("production mode: skipping auto-migrations (use migrate CLI)")
	} else {
		if err := autoMigrate(db); err != nil {
			return nil, fmt.Errorf("failed to run auto-migrations: %w", err)
		}
	}

	log.Println("database connected successfully")
	return db, nil
}

// autoMigrate runs GORM auto-migrations for all models.
func autoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&auth.User{},
		&auth.RefreshToken{},
		&household.Household{},
		&household.HouseholdMember{},
		&category.Category{},
		&shoppinglist.ShoppingList{},
		&shoppingitem.ShoppingItem{},
		&history.ShoppingHistory{},
	)
}
