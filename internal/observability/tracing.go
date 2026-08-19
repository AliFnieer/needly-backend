package observability

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// TraceStatus represents the status of a span.
type TraceStatus string

const (
	TraceStatusOK       TraceStatus = "ok"
	TraceStatusError    TraceStatus = "error"
	TraceStatusNotFound TraceStatus = "not_found"
)

// Span represents a single tracing span.
type Span struct {
	Name       string
	StartTime  time.Time
	EndTime    time.Time
	Status     TraceStatus
	Attributes map[string]any
}

// Duration returns the span duration.
func (s *Span) Duration() time.Duration {
	return s.EndTime.Sub(s.StartTime)
}

// Tracer provides lightweight distributed tracing support.
// This implementation uses context propagation and can be
// integrated with OpenTelemetry later.
type Tracer struct {
	// spans collects all completed spans for this process.
	spans []*Span
}

// NewTracer creates a new tracer.
func NewTracer() *Tracer {
	return &Tracer{
		spans: make([]*Span, 0),
	}
}

// StartSpan starts a new span and returns it along with a derived context.
func (t *Tracer) StartSpan(ctx context.Context, name string, attrs map[string]any) (context.Context, *Span) {
	span := &Span{
		Name:       name,
		StartTime:  time.Now(),
		Status:     TraceStatusOK,
		Attributes: attrs,
	}

	// Store the span in the context
	ctx = context.WithValue(ctx, spanKey{}, span)

	return ctx, span
}

// End marks a span as complete and records it.
func (t *Tracer) End(span *Span, status TraceStatus) {
	if span == nil {
		return
	}
	span.EndTime = time.Now()
	if status != "" {
		span.Status = status
	}
	t.spans = append(t.spans, span)
}

// SpanFromContext retrieves the current span from a context, if any.
func SpanFromContext(ctx context.Context) *Span {
	span, _ := ctx.Value(spanKey{}).(*Span)
	return span
}

// AddSpanAttribute adds an attribute to the active span in the context.
func AddSpanAttribute(ctx context.Context, key string, value any) {
	span := SpanFromContext(ctx)
	if span == nil {
		return
	}
	if span.Attributes == nil {
		span.Attributes = make(map[string]any)
	}
	span.Attributes[key] = value
}

// Spans returns a copy of all completed spans.
func (t *Tracer) Spans() []*Span {
	result := make([]*Span, len(t.spans))
	copy(result, t.spans)
	return result
}

// Clear resets all collected spans.
func (t *Tracer) Clear() {
	t.spans = t.spans[:0]
}

// Snapshot returns tracing summary data for metrics reporting.
func (t *Tracer) Snapshot() map[string]any {
	total := int64(len(t.spans))
	errors := int64(0)
	var totalDuration time.Duration

	for _, s := range t.spans {
		totalDuration += s.Duration()
		if s.Status == TraceStatusError {
			errors++
		}
	}

	avgDuration := float64(0)
	if total > 0 {
		avgDuration = totalDuration.Seconds() / float64(total)
	}

	return map[string]any{
		"spans_total":      total,
		"spans_errors":     errors,
		"avg_duration_ms":  avgDuration * 1000,
	}
}

// TraceHandler returns an HTTP handler exposing tracing summary.
func (t *Tracer) TraceHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		snapshot := t.Snapshot()
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(snapshot); err != nil {
			http.Error(w, "failed to encode tracing snapshot", http.StatusInternalServerError)
			return
		}
	})
}

// spanKey is the context key for the current span.
type spanKey struct{}
