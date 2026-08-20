package cache

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/AliFnieer/needly-backend/internal/config"
	"github.com/redis/go-redis/v9"
)

// InitRedis establishes a connection to Redis.
func InitRedis(cfg *config.Config) (*redis.Client, error) {
	addr := fmt.Sprintf("%s:%s", cfg.Redis.Host, cfg.Redis.Port)

	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Verify connection
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close() // #nosec G104 - best-effort cleanup on error
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	slog.Info("redis connected", "addr", addr)
	return client, nil
}
