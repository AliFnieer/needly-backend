package middleware

import (
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// CircuitState represents the state of a circuit breaker.
type CircuitState int

const (
	CircuitClosed   CircuitState = iota // Normal operation
	CircuitOpen                         // Failing, reject requests
	CircuitHalfOpen                     // Testing if service recovered
)

// CircuitBreaker provides failure-rate-based circuit breaking.
type CircuitBreaker struct {
	mu               sync.Mutex
	state            CircuitState
	failures         int
	successes        int
	threshold        int           // failures before opening
	resetTimeout     time.Duration // time before trying half-open
	halfOpenMax      int           // successful requests needed to close
	lastFailureTime  time.Time
}

// CircuitBreakerConfig configures the circuit breaker.
type CircuitBreakerConfig struct {
	Threshold    int           // failures before opening (default: 5)
	ResetTimeout time.Duration // time before half-open (default: 30s)
	HalfOpenMax  int           // successes to close (default: 2)
}

// NewCircuitBreaker creates a new circuit breaker.
func NewCircuitBreaker(cfg *CircuitBreakerConfig) *CircuitBreaker {
	threshold := 5
	resetTimeout := 30 * time.Second
	halfOpenMax := 2

	if cfg != nil {
		if cfg.Threshold > 0 {
			threshold = cfg.Threshold
		}
		if cfg.ResetTimeout > 0 {
			resetTimeout = cfg.ResetTimeout
		}
		if cfg.HalfOpenMax > 0 {
			halfOpenMax = cfg.HalfOpenMax
		}
	}

	return &CircuitBreaker{
		state:        CircuitClosed,
		threshold:    threshold,
		resetTimeout: resetTimeout,
		halfOpenMax:  halfOpenMax,
	}
}

// State returns the current circuit state.
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == CircuitOpen && time.Since(cb.lastFailureTime) > cb.resetTimeout {
		cb.state = CircuitHalfOpen
		cb.successes = 0
	}

	return cb.state
}

// RecordSuccess records a successful request.
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitHalfOpen:
		cb.successes++
		if cb.successes >= cb.halfOpenMax {
			cb.state = CircuitClosed
			cb.failures = 0
			cb.successes = 0
			slog.Info("circuit breaker closed (recovered)")
		}
	case CircuitClosed:
		cb.failures = 0
	}
}

// RecordFailure records a failed request.
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	cb.lastFailureTime = time.Now()

	switch cb.state {
	case CircuitHalfOpen:
		cb.state = CircuitOpen
		slog.Warn("circuit breaker re-opened (half-open test failed)")
	case CircuitClosed:
		if cb.failures >= cb.threshold {
			cb.state = CircuitOpen
			slog.Warn("circuit breaker opened", "failures", cb.failures)
		}
	}
}

// Middleware returns a Gin middleware that uses the circuit breaker.
// Requests are rejected with 503 when the circuit is open.
func (cb *CircuitBreaker) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if cb.State() == CircuitOpen {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error":       "service temporarily unavailable",
				"retry_after": int(cb.resetTimeout.Seconds()),
			})
			c.Abort()
			return
		}

		c.Next()

		// Record outcome based on status code
		if c.Writer.Status() >= 500 {
			cb.RecordFailure()
		} else {
			cb.RecordSuccess()
		}
	}
}
