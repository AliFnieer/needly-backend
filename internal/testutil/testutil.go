package testutil

import (
	"testing"
	"time"

	"github.com/AliFnieer/needly-backend/internal/auth"
	"github.com/AliFnieer/needly-backend/internal/category"
	"github.com/AliFnieer/needly-backend/internal/config"
	"github.com/AliFnieer/needly-backend/internal/history"
	"github.com/AliFnieer/needly-backend/internal/household"
	"github.com/AliFnieer/needly-backend/internal/idempotency"
	"github.com/AliFnieer/needly-backend/internal/shoppingitem"
	"github.com/AliFnieer/needly-backend/internal/shoppinglist"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// TestConfig returns a config suitable for tests.
func TestConfig() *config.Config {
	return &config.Config{
		Server: config.ServerConfig{
			Port:    "0",
			GinMode: "test",
		},
		JWT: config.JWTConfig{
			Secret:               "test-secret-key-for-testing-only-32chars!",
			ExpirationHours:      1,
			RefreshTokenTTLHours: 720,
			Issuer:               "needly-api",
		},
		Notification: config.NotificationConfig{
			Enabled:          false,
			WebSocketEnabled: false,
			HistoryLimit:     50,
		},
		RateLimit: config.RateLimitConfig{
			Enabled: false,
		},
	}
}

// SetupTestDB creates an in-memory SQLite database with all models migrated.
func SetupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger:                 logger.Default.LogMode(logger.Silent),
		SkipDefaultTransaction: true,
		// Match production: write auto-timestamps in UTC.
		NowFunc: func() time.Time { return time.Now().UTC() },
	})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	// Run all migrations (Notification is not a DB model — it's broadcast via WebSocket)
	err = db.AutoMigrate(
		&auth.User{},
		&auth.RefreshToken{},
		&auth.PasswordResetToken{},
		&auth.EmailVerificationToken{},
		&household.Household{},
		&household.HouseholdMember{},
		&category.Category{},
		&history.ShoppingHistory{},
		&shoppinglist.ShoppingList{},
		&shoppingitem.ShoppingItem{},
		&idempotency.IdempotencyKey{},
	)
	if err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}

	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	})

	return db
}

// SeedUser creates a test user and returns it.
func SeedUser(t *testing.T, db *gorm.DB, email, password string) *auth.User {
	t.Helper()
	_, err := auth.NewService(db, TestConfig(), nil).Register(&auth.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     email,
		Password:  password,
	})
	if err != nil {
		t.Fatalf("failed to seed user: %v", err)
	}
	var u auth.User
	if err := db.Where("email = ?", email).First(&u).Error; err != nil {
		t.Fatalf("failed to fetch seeded user: %v", err)
	}
	return &u
}
