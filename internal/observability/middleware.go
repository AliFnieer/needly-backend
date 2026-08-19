package observability

import (
	"time"

	"github.com/gin-gonic/gin"
)

// Middleware provides Gin middleware that integrates logging, metrics, and tracing.
type Middleware struct {
	logger  *Logger
	metrics *Metrics
	tracer  *Tracer
}

// NewMiddleware creates a new observability middleware.
func NewMiddleware(logger *Logger, metrics *Metrics, tracer *Tracer) *Middleware {
	return &Middleware{
		logger:  logger,
		metrics: metrics,
		tracer:  tracer,
	}
}

// GinMiddleware returns a Gin handler that records metrics and traces for each request.
func (m *Middleware) GinMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		// Start a trace span
		ctx, span := m.tracer.StartSpan(c.Request.Context(), method+" "+path, map[string]any{
			"method": method,
			"path":   path,
		})

		// Replace the request context with the traced context
		c.Request = c.Request.WithContext(ctx)

		// Record the request
		m.metrics.IncHTTPRequest(method, path)

		// Process request
		c.Next()

		// Record metrics after processing
		duration := time.Since(start)
		m.metrics.ObserveHTTPDuration(method, path, duration)
		m.metrics.DecHTTPRequest()

		status := c.Writer.Status()

		// Log the request
		if m.logger != nil {
			fields := []any{
				"method", method,
				"path", path,
				"status", status,
				"duration_ms", duration.Milliseconds(),
				"client_ip", c.ClientIP(),
				"user_agent", c.Request.UserAgent(),
			}

			// Add user ID if authenticated
			if userID, exists := c.Get("user_id"); exists {
				fields = append(fields, "user_id", userID)
			}

			// Add request ID if present
			if requestID := c.GetString("request_id"); requestID != "" {
				fields = append(fields, "request_id", requestID)
			}

			if status >= 500 {
				m.logger.Error("http request failed", fields...)
			} else if status >= 400 {
				m.logger.Warn("http request warning", fields...)
			} else {
				m.logger.Info("http request", fields...)
			}
		}

		// End the trace span
		spanStatus := TraceStatusOK
		if status >= 500 {
			spanStatus = TraceStatusError
		} else if status == 404 {
			spanStatus = TraceStatusNotFound
		}
		m.tracer.End(span, spanStatus)
	}
}

// DBQuery returns a function to record a database query with metrics.
func (m *Middleware) DBQuery(operation string) func() {
	start := time.Now()
	return func() {
		duration := time.Since(start)
		m.metrics.IncDBQuery(operation)
		m.metrics.ObserveDBDuration(operation, duration)
	}
}

// DBCacheHit records a cache hit.
func (m *Middleware) DBCacheHit() {
	m.metrics.IncCacheHit()
}

// DBCacheMiss records a cache miss.
func (m *Middleware) DBCacheMiss() {
	m.metrics.IncCacheMiss()
}