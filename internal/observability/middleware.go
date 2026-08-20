package observability

import (
	"context"
	"log/slog"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	oteltrace "go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("needly-api")

// Middleware provides Gin middleware that integrates OTel tracing, Prometheus metrics, and structured logging.
type Middleware struct {
	metrics *Metrics
}

// NewMiddleware creates a new observability middleware.
func NewMiddleware(metrics *Metrics) *Middleware {
	return &Middleware{
		metrics: metrics,
	}
}

// GinMiddleware returns a Gin handler that records metrics and traces for each request.
func (m *Middleware) GinMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		// Start OTel span
		ctx, span := tracer.Start(c.Request.Context(), method+" "+path,
			oteltrace.WithAttributes(
				attribute.String("http.method", method),
				attribute.String("http.url", path),
				attribute.String("http.user_agent", c.Request.UserAgent()),
				attribute.String("http.client_ip", c.ClientIP()),
			),
		)
		defer span.End()

		c.Request = c.Request.WithContext(ctx)

		// Track active requests
		m.metrics.HTTPActiveRequests.Inc()

		// Process request
		c.Next()

		// Record metrics after processing
		duration := time.Since(start)
		status := c.Writer.Status()
		statusStr := strconv.Itoa(status)

		m.metrics.HTTPActiveRequests.Dec()
		m.metrics.HTTPRequestsTotal.WithLabelValues(method, path, statusStr).Inc()
		m.metrics.HTTPRequestDuration.WithLabelValues(method, path, statusStr).Observe(duration.Seconds())

		// Set span attributes
		span.SetAttributes(
			attribute.Int("http.status_code", status),
			attribute.Float64("http.duration_ms", float64(duration.Milliseconds())),
		)

		// Set span status
		if status >= 500 {
			span.SetStatus(codes.Error, "server error")
			span.RecordError(c.Errors.Last().Err)
		} else if status >= 400 {
			span.SetStatus(codes.Error, "client error")
		} else {
			span.SetStatus(codes.Ok, "")
		}

		// Structured logging
		fields := []any{
			"method", method,
			"path", path,
			"status", status,
			"duration_ms", duration.Milliseconds(),
			"client_ip", c.ClientIP(),
			"user_agent", c.Request.UserAgent(),
		}

		if userID, exists := c.Get("user_id"); exists {
			fields = append(fields, "user_id", userID)
		}
		if requestID := c.GetString("request_id"); requestID != "" {
			fields = append(fields, "request_id", requestID)
		}
		if traceID := span.SpanContext().TraceID(); traceID.IsValid() {
			fields = append(fields, "trace_id", traceID.String())
		}

		switch {
		case status >= 500:
			slog.Error("http request failed", fields...)
		case status >= 400:
			slog.Warn("http request error", fields...)
		default:
			slog.Info("http request", fields...)
		}
	}
}

// StartSpan creates a new child span from the given context.
func StartSpan(ctx context.Context, name string, opts ...oteltrace.SpanStartOption) (context.Context, oteltrace.Span) {
	return tracer.Start(ctx, name, opts...)
}
