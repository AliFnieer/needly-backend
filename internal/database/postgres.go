package database

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
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
		// Write auto-timestamps (created_at/updated_at) in UTC so stored
		// representations stay consistent across environments.
		NowFunc: func() time.Time { return time.Now().UTC() },
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
		slog.Info("production mode: skipping auto-migrations (use migrate CLI)")
	} else {
		if err := autoMigrate(db); err != nil {
			return nil, fmt.Errorf("failed to run auto-migrations: %w", err)
		}
	}

	slog.Info("database connected successfully")
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

// RunMigrations applies pending SQL migration files from the given directory.
// It creates a migrations table to track applied migrations and applies any
// that haven't been run yet, in order.
func RunMigrations(db *gorm.DB, migrationsDir string) error {
	// Create migrations tracking table
	type Migration struct {
		ID        uint      `gorm:"primaryKey"`
		Version   string    `gorm:"size:255;uniqueIndex"`
		AppliedAt time.Time `gorm:"not null"`
	}
	if err := db.AutoMigrate(&Migration{}); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		if os.IsNotExist(err) {
			slog.Info("migrations directory not found, skipping", "dir", migrationsDir)
			return nil
		}
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	// Collect applied versions
	var applied []Migration
	if err := db.Find(&applied).Error; err != nil {
		return fmt.Errorf("failed to list applied migrations: %w", err)
	}
	appliedSet := make(map[string]bool, len(applied))
	for _, m := range applied {
		appliedSet[m.Version] = true
	}

	// Apply pending migrations
	appliedCount := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		version := entry.Name()
		if appliedSet[version] {
			continue
		}

		// Only process .sql files
		if filepath.Ext(version) != ".sql" {
			continue
		}

		filePath := filepath.Join(migrationsDir, version)
		// #nosec G304 -- version comes from os.ReadDir of the migrations
		// directory, never from user input; the .sql extension is validated above.
		content, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", version, err)
		}

		// Split by goose up/down markers — only apply "up" section
		sql := extractUpMigration(string(content))
		if sql == "" {
			slog.Info("migration has no up section, skipping", "file", version)
			continue
		}

		if err := db.Exec(sql).Error; err != nil {
			return fmt.Errorf("failed to apply migration %s: %w", version, err)
		}

		record := Migration{
			Version:   version,
			AppliedAt: time.Now(),
		}
		if err := db.Create(&record).Error; err != nil {
			return fmt.Errorf("failed to record migration %s: %w", version, err)
		}

		appliedCount++
		slog.Info("applied migration", "file", version)
	}

	if appliedCount > 0 {
		slog.Info("migrations complete", "applied", appliedCount)
	} else {
		slog.Info("database is up to date, no pending migrations")
	}

	return nil
}

// extractUpMigration extracts the SQL between "-- +goose Up" and "-- +goose Down" markers.
func extractUpMigration(content string) string {
	upMarker := "-- +goose Up"
	downMarker := "-- +goose Down"

	upIdx := -1
	for i := 0; i < len(content)-len(upMarker); i++ {
		if content[i:i+len(upMarker)] == upMarker {
			upIdx = i + len(upMarker)
			break
		}
	}
	if upIdx == -1 {
		return ""
	}

	downIdx := len(content)
	for i := upIdx; i < len(content)-len(downMarker); i++ {
		if content[i:i+len(downMarker)] == downMarker {
			downIdx = i
			break
		}
	}

	return content[upIdx:downIdx]
}
