package database

import (
	"fmt"
	"log"

	"github.com/AliFnieer/needly-backend/internal/auth"
	"github.com/AliFnieer/needly-backend/internal/category"
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

// DefaultSeedCategories returns the default set of categories.
func DefaultSeedCategories() []category.Category {
	return []category.Category{
		{Name: "Groceries"},
		{Name: "Produce"},
		{Name: "Dairy"},
		{Name: "Meat & Seafood"},
		{Name: "Bakery"},
		{Name: "Frozen"},
		{Name: "Beverages"},
		{Name: "Snacks"},
		{Name: "Household"},
		{Name: "Personal Care"},
		{Name: "Other"},
	}
}

// SeedCategories inserts default categories into the database if they do not already exist.
// It is idempotent: categories with an existing name are skipped.
func SeedCategories(db *gorm.DB, categories []category.Category) error {
	if len(categories) == 0 {
		categories = DefaultSeedCategories()
	}

	created := 0
	skipped := 0

	for _, cat := range categories {
		var existing category.Category
		err := db.Where("name = ?", cat.Name).First(&existing).Error
		if err == nil {
			skipped++
			continue
		} else if err != gorm.ErrRecordNotFound {
			return fmt.Errorf("failed to check existing category %s: %w", cat.Name, err)
		}

		if err := db.Create(&cat).Error; err != nil {
			return fmt.Errorf("failed to create category %s: %w", cat.Name, err)
		}

		created++
	}

	log.Printf("seed categories complete: %d created, %d skipped", created, skipped)
	return nil
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