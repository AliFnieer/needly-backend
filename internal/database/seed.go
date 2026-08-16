package database

import (
	"fmt"
	"log"

	"github.com/AliFnieer/needly-backend/internal/auth"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// SeedUser represents a demo user to be inserted by the seeder.
type SeedUser struct {
	FirstName string
	LastName  string
	Email     string
	Password  string
}

// DefaultSeedUsers returns the default set of demo users.
func DefaultSeedUsers() []SeedUser {
	return []SeedUser{
		{
			FirstName: "Alice",
			LastName:  "Johnson",
			Email:     "alice@example.com",
			Password:  "password123",
		},
		{
			FirstName: "Bob",
			LastName:  "Smith",
			Email:     "bob@example.com",
			Password:  "password123",
		},
		{
			FirstName: "Carol",
			LastName:  "Williams",
			Email:     "carol@example.com",
			Password:  "password123",
		},
		{
			FirstName: "David",
			LastName:  "Brown",
			Email:     "david@example.com",
			Password:  "password123",
		},
		{
			FirstName: "Emma",
			LastName:  "Davis",
			Email:     "emma@example.com",
			Password:  "password123",
		},
	}
}

// SeedUsers inserts demo users into the database if they do not already exist.
// It is idempotent: users with an existing email are skipped.
func SeedUsers(db *gorm.DB, users []SeedUser) error {
	if len(users) == 0 {
		users = DefaultSeedUsers()
	}

	created := 0
	skipped := 0

	for _, su := range users {
		var existing auth.User
		err := db.Where("email = ?", su.Email).First(&existing).Error
		if err == nil {
			skipped++
			continue
		} else if err != gorm.ErrRecordNotFound {
			return fmt.Errorf("failed to check existing user %s: %w", su.Email, err)
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(su.Password), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("failed to hash password for %s: %w", su.Email, err)
		}

		user := auth.User{
			FirstName:    su.FirstName,
			LastName:     su.LastName,
			Email:        su.Email,
			PasswordHash: string(hashedPassword),
		}

		if err := db.Create(&user).Error; err != nil {
			return fmt.Errorf("failed to create user %s: %w", su.Email, err)
		}

		created++
	}

	log.Printf("seed users complete: %d created, %d skipped", created, skipped)
	return nil
}