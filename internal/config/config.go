package config

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	JWT      JWTConfig
	CORS     CORSConfig
}

type ServerConfig struct {
	Port    string
	GinMode string
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
	TimeZone string
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

type JWTConfig struct {
	Secret            string
	ExpirationHours   int
}

type CORSConfig struct {
	AllowedOrigins []string
}

// Load reads configuration from .env file and environment variables.
func Load() *Config {
	// Try to load .env file if it exists (ignore error for production env vars)
	_ = godotenv.Load()

	cfg := &Config{
		Server: ServerConfig{
			Port:    getEnv("PORT", "8080"),
			GinMode: getEnv("GIN_MODE", "debug"),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "postgres"),
			Name:     getEnv("DB_NAME", "needly"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
			TimeZone: getEnv("DB_TIMEZONE", "Africa/Tripoli"),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
		},
		JWT: JWTConfig{
			Secret:          getEnv("JWT_SECRET", "your_super_secret_jwt_key_change_me"),
			ExpirationHours: getEnvAsInt("JWT_EXPIRATION_HOURS", 24),
		},
		CORS: CORSConfig{
			AllowedOrigins: splitCSV(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:5173")),
		},
	}

	log.Printf("config loaded: server port=%s, db host=%s, redis host=%s", cfg.Server.Port, cfg.Database.Host, cfg.Redis.Host)
	return cfg
}

// getEnv returns the value of the environment variable or a default if not set.
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsInt returns the environment variable as an integer or a default.
func getEnvAsInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
		log.Printf("invalid integer for %s, using default %d", key, defaultValue)
	}
	return defaultValue
}

// splitCSV splits a comma-separated string into a slice, trimming whitespace.
func splitCSV(value string) []string {
	var result []string
	if value == "" {
		return result
	}
	parts := strings.Split(value, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}
