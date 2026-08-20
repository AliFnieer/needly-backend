package observability

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// JaegerExporter sends completed spans to a Jaeger collector via the
// Jaeger Thrift HTTP endpoint. It is safe for concurrent use.
type JaegerExporter struct {
	endpoint   string
	serviceName string
	client     *http.Client
}

// JaegerConfig configures the Jaeger trace exporter.
type JaegerConfig struct {
	Endpoint    string // Jaeger collector HTTP endpoint (e.g. http://jaeger:14268/api/traces)
	ServiceName string // Service name shown in Jaeger UI
	Timeout     time.Duration
}

// NewJaegerExporter creates a new Jaeger exporter. Returns nil if endpoint is empty.
func NewJaegerExporter(cfg *JaegerConfig) *JaegerExporter {
	if cfg == nil || cfg.Endpoint == "" {
		return nil
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	serviceName := cfg.ServiceName
	if serviceName == "" {
		serviceName = "needly-api"
	}

	return &JaegerExporter{
		endpoint:    cfg.Endpoint,
		serviceName: serviceName,
		client:      &http.Client{Timeout: timeout},
	}
}

// jaegerSpan is the Jaeger Thrift JSON span format.
type jaegerSpan struct {
	TraceID       string            `json:"traceID"`
	SpanID        string            `json:"spanID"`
	ParentSpanID  string            `json:"parentSpanID,omitempty"`
	OperationName string            `json:"operationName"`
	StartTime     int64             `json:"startTime"`
	Duration      int64             `json:"duration"`
	Tags          map[string]string `json:"tags,omitempty"`
	Logs          []jaegerLog       `json:"logs,omitempty"`
}

type jaegerLog struct {
	Timestamp int64             `json:"timestamp"`
	Fields    map[string]string `json:"fields"`
}

type jaegerBatch struct {
	Service  string      `json:"service"`
	Spans    []jaegerSpan `json:"spans"`
}

// Export sends a completed span to Jaeger.
func (e *JaegerExporter) Export(span *Span) {
	if e == nil || span == nil {
		return
	}

	tags := make(map[string]string)
	for k, v := range span.Attributes {
		tags[k] = fmt.Sprintf("%v", v)
	}
	tags["status"] = string(span.Status)

	batch := jaegerBatch{
		Service: e.serviceName,
		Spans: []jaegerSpan{{
			TraceID:       generateTraceID(),
			SpanID:        generateSpanID(),
			OperationName: span.Name,
			StartTime:     span.StartTime.UnixMicro(),
			Duration:      span.Duration().Microseconds(),
			Tags:          tags,
		}},
	}

	body, err := json.Marshal(batch)
	if err != nil {
		slog.Error("jaeger: failed to marshal span", "error", err)
		return
	}

	req, err := http.NewRequest(http.MethodPost, e.endpoint, bytes.NewReader(body))
	if err != nil {
		slog.Error("jaeger: failed to create request", "error", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	go func() {
		resp, err := e.client.Do(req)
		if err != nil {
			slog.Error("jaeger: failed to send span", "error", err)
			return
		}
		resp.Body.Close()
		if resp.StatusCode >= 300 {
			slog.Error("jaeger: unexpected status", "status", resp.StatusCode)
		}
	}()
}

// ExportBatch sends multiple spans at once.
func (e *JaegerExporter) ExportBatch(spans []*Span) {
	if e == nil || len(spans) == 0 {
		return
	}

	batch := jaegerBatch{
		Service: e.serviceName,
		Spans:   make([]jaegerSpan, 0, len(spans)),
	}

	for _, span := range spans {
		tags := make(map[string]string)
		for k, v := range span.Attributes {
			tags[k] = fmt.Sprintf("%v", v)
		}
		tags["status"] = string(span.Status)

		batch.Spans = append(batch.Spans, jaegerSpan{
			TraceID:       generateTraceID(),
			SpanID:        generateSpanID(),
			OperationName: span.Name,
			StartTime:     span.StartTime.UnixMicro(),
			Duration:      span.Duration().Microseconds(),
			Tags:          tags,
		})
	}

	body, err := json.Marshal(batch)
	if err != nil {
		slog.Error("jaeger: failed to marshal batch", "error", err)
		return
	}

	req, err := http.NewRequest(http.MethodPost, e.endpoint, bytes.NewReader(body))
	if err != nil {
		slog.Error("jaeger: failed to create batch request", "error", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	go func() {
		resp, err := e.client.Do(req)
		if err != nil {
			slog.Error("jaeger: failed to send batch", "error", err)
			return
		}
		resp.Body.Close()
	}()
}

func generateTraceID() string {
	return generateHex(16)
}

func generateSpanID() string {
	return generateHex(8)
}

func generateHex(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = "0123456789abcdef"[time.Now().UnixNano()%16]
		time.Sleep(time.Nanosecond)
	}
	return string(b)
}
