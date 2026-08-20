package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/AliFnieer/needly-backend/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

const (
	// rateLimitKeyPrefix is the Redis key prefix for rate limit counters.
	rateLimitKeyPrefix = "ratelimit:"
)

// RateLimiter provides Redis-backed fixed-window rate limiting.
type RateLimiter struct {
	client *redis.Client
	cfg    *config.RateLimitConfig
}

// NewRateLimiter creates a new Redis-backed rate limiter.
func NewRateLimiter(client *redis.Client, cfg *config.RateLimitConfig) *RateLimiter {
	return &RateLimiter{
		client: client,
		cfg:    cfg,
	}
}

// Middleware returns a Gin handler that enforces the global rate limit.
func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	if rl == nil || rl.cfg == nil || !rl.cfg.Enabled {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	return rl.makeMiddleware(rl.cfg)
}

// StrictMiddleware returns a rate limiter with tighter limits for sensitive
// endpoints like auth (register, login, refresh).
func (rl *RateLimiter) StrictMiddleware() gin.HandlerFunc {
	if rl == nil || rl.cfg == nil || !rl.cfg.Enabled {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	strict := &config.RateLimitConfig{
		Enabled:       true,
		Requests:      10,
		WindowSeconds: 60,
	}

	return rl.makeMiddleware(strict)
}

// makeMiddleware creates a rate limiting middleware with the given config.
func (rl *RateLimiter) makeMiddleware(cfg *config.RateLimitConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := rl.clientKey(c)
		if key == "" {
			c.Next()
			return
		}

		allowed, remaining, retryAfter, err := rl.allow(c.Request.Context(), key, cfg)
		if err != nil {
			slog.Error("rate limit redis error", "key", key, "error", err)
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "service temporarily unavailable",
			})
			c.Abort()
			return
		}

		c.Header("X-RateLimit-Limit", strconv.Itoa(cfg.Requests))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		if retryAfter > 0 {
			c.Header("Retry-After", strconv.Itoa(retryAfter))
		}

		if !allowed {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "rate limit exceeded",
				"retry_after": retryAfter,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// allow increments the counter for the given key and returns whether the
// request is permitted, the remaining allowance, and seconds until the
// window resets when rejected.
func (rl *RateLimiter) allow(ctx context.Context, key string, cfg *config.RateLimitConfig) (bool, int, int, error) {
	window := time.Now().Unix() / int64(cfg.WindowSeconds)
	windowKey := fmt.Sprintf("%s%s:%d", rateLimitKeyPrefix, key, window)

	count, err := rl.client.Incr(ctx, windowKey).Result()
	if err != nil {
		return false, 0, 0, fmt.Errorf("failed to increment rate limit counter: %w", err)
	}

	if count == 1 {
		if err := rl.client.Expire(ctx, windowKey, time.Duration(cfg.WindowSeconds)*time.Second).Err(); err != nil {
			slog.Error("rate limit failed to set expiry", "key", windowKey, "error", err)
		}
	}

	remaining := cfg.Requests - int(count)
	if remaining < 0 {
		remaining = 0
	}

	if count > int64(cfg.Requests) {
		elapsed := time.Now().Unix() % int64(cfg.WindowSeconds)
		retryAfter := int(int64(cfg.WindowSeconds) - elapsed)
		if retryAfter < 1 {
			retryAfter = 1
		}
		return false, remaining, retryAfter, nil
	}

	return true, remaining, 0, nil
}

// clientKey builds a stable identifier for the caller, preferring the
// authenticated user ID when present, otherwise falling back to the IP.
func (rl *RateLimiter) clientKey(c *gin.Context) string {
	if userID, exists := c.Get("user_id"); exists {
		switch v := userID.(type) {
		case float64:
			return fmt.Sprintf("user:%d", uint(v))
		case uint:
			return fmt.Sprintf("user:%d", v)
		case string:
			return "user:" + v
		}
	}

	ip := c.ClientIP()
	if ip == "" {
		ip = c.Request.RemoteAddr
	}
	return "ip:" + ip
}
