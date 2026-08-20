package middleware

import (
	"context"
	"fmt"
	"log"
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

// Middleware returns a Gin handler that enforces the rate limit.
// When rate limiting is disabled, a no-op middleware is returned.
func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	if rl == nil || rl.cfg == nil || !rl.cfg.Enabled {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	return func(c *gin.Context) {
		key := rl.clientKey(c)
		if key == "" {
			c.Next()
			return
		}

		allowed, remaining, retryAfter, err := rl.allow(c.Request.Context(), key)
		if err != nil {
			// Fail closed when Redis is unavailable to prevent abuse.
			log.Printf("rate limit: redis error for key %s: %v", key, err)
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "service temporarily unavailable",
			})
			c.Abort()
			return
		}

		c.Header("X-RateLimit-Limit", strconv.Itoa(rl.cfg.Requests))
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
func (rl *RateLimiter) allow(ctx context.Context, key string) (bool, int, int, error) {
	window := time.Now().Unix() / int64(rl.cfg.WindowSeconds)
	windowKey := fmt.Sprintf("%s%s:%d", rateLimitKeyPrefix, key, window)

	count, err := rl.client.Incr(ctx, windowKey).Result()
	if err != nil {
		return false, 0, 0, fmt.Errorf("failed to increment rate limit counter: %w", err)
	}

	if count == 1 {
		if err := rl.client.Expire(ctx, windowKey, time.Duration(rl.cfg.WindowSeconds)*time.Second).Err(); err != nil {
			log.Printf("rate limit: failed to set expiry on key %s: %v", windowKey, err)
		}
	}

	remaining := rl.cfg.Requests - int(count)
	if remaining < 0 {
		remaining = 0
	}

	if count > int64(rl.cfg.Requests) {
		elapsed := time.Now().Unix() % int64(rl.cfg.WindowSeconds)
		retryAfter := int(int64(rl.cfg.WindowSeconds) - elapsed)
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