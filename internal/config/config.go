package config

import (
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	Server       ServerConfig
	Database     DatabaseConfig
	Redis        RedisConfig
	JWT          JWTConfig
	CORS         CORSConfig
	RateLimit    RateLimitConfig
	Notification NotificationConfig
	Tracing      TracingConfig
}

type ServerConfig struct {
	Port    string
	GinMode string
}

type DatabaseConfig struct {
	Host                   string
	Port                   string
	User                   string
	Password               string
	Name                   string
	SSLMode                string
	TimeZone               string
	MaxOpenConns           int
	MaxIdleConns           int
	ConnMaxLifetimeMinutes int
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

type JWTConfig struct {
	Secret              string
	ExpirationHours     int
	RefreshTokenTTLHours int
	Issuer              string
}

type CORSConfig struct {
	AllowedOrigins []string
}

// RateLimitConfig configures the Redis-backed API rate limiter.
type RateLimitConfig struct {
	Enabled       bool
	Requests      int
	WindowSeconds int
}

// NotificationConfig configures push notification delivery.
type NotificationConfig struct {
	Enabled          bool
	WebSocketEnabled bool
	HistoryLimit     int
}

// TracingConfig configures OpenTelemetry distributed tracing.
type TracingConfig struct {
	OTLPEndpoint string
	ServiceName  string
	LogLevel     string
}

const (
	// defaultJWTSecret is the placeholder secret that must not be used in production.
	defaultJWTSecret = "your_super_secret_jwt_key_change_me"
)

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
			Host:                   getEnv("DB_HOST", "localhost"),
			Port:                   getEnv("DB_PORT", "5432"),
			User:                   getEnv("DB_USER", "postgres"),
			Password:               getEnv("DB_PASSWORD", "postgres"),
			Name:                   getEnv("DB_NAME", "needly"),
			SSLMode:                getEnv("DB_SSLMODE", "disable"),
			TimeZone:               getEnv("DB_TIMEZONE", "Africa/Tripoli"),
			MaxOpenConns:           getEnvAsInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:           getEnvAsInt("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetimeMinutes: getEnvAsInt("DB_CONN_MAX_LIFETIME_MINUTES", 5),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
		},
		JWT: JWTConfig{
			Secret:              getEnv("JWT_SECRET", "your_super_secret_jwt_key_change_me"),
			ExpirationHours:     getEnvAsInt("JWT_EXPIRATION_HOURS", 1),
			RefreshTokenTTLHours: getEnvAsInt("JWT_REFRESH_TOKEN_TTL_HOURS", 720),
			Issuer:              getEnv("JWT_ISSUER", "needly-api"),
		},
		CORS: CORSConfig{
			AllowedOrigins: splitCSV(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:5173")),
		},
		RateLimit: RateLimitConfig{
			Enabled:       getEnvAsBool("RATE_LIMIT_ENABLED", true),
			Requests:      getEnvAsInt("RATE_LIMIT_REQUESTS", 100),
			WindowSeconds: getEnvAsInt("RATE_LIMIT_WINDOW_SECONDS", 60),
		},
		Notification: NotificationConfig{
			Enabled:          getEnvAsBool("NOTIFICATIONS_ENABLED", true),
			WebSocketEnabled: getEnvAsBool("NOTIFICATIONS_WEBSOCKET_ENABLED", true),
			HistoryLimit:     getEnvAsInt("NOTIFICATIONS_HISTORY_LIMIT", 50),
		},
		Tracing: TracingConfig{
			OTLPEndpoint: getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", ""),
			ServiceName:  getEnv("OTEL_SERVICE_NAME", "needly-api"),
			LogLevel:     getEnv("LOG_LEVEL", "info"),
		},
	}

	// Validate critical secrets in production
	if cfg.Server.GinMode == "release" {
		if cfg.JWT.Secret == defaultJWTSecret {
			slog.Error("FATAL: JWT_SECRET is using the default value. Set a strong random secret in production.")
			os.Exit(1)
		}
	}

	slog.Info("config loaded",
		"port", cfg.Server.Port,
		"db_host", cfg.Database.Host,
		"redis_host", cfg.Redis.Host,
		"rate_limit_enabled", cfg.RateLimit.Enabled,
	)
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
		slog.Warn("invalid integer for env var", "key", key, "default", defaultValue)
	}
	return defaultValue
}

// getEnvAsBool returns the environment variable as a boolean or a default.
func getEnvAsBool(key string, defaultValue bool) bool {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
		slog.Warn("invalid boolean for env var", "key", key, "default", defaultValue)
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