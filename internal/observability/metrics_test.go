package observability

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestNewMetrics(t *testing.T) {
	m := NewMetrics()
	if m == nil {
		t.Fatal("NewMetrics returned nil")
	}
	if m.registry == nil {
		t.Fatal("registry is nil")
	}
}

func TestMetricsHTTPCounters(t *testing.T) {
	m := NewMetrics()

	m.HTTPRequestsTotal.WithLabelValues("GET", "/health", "200").Inc()
	m.HTTPRequestsTotal.WithLabelValues("POST", "/api/v1/auth/login", "201").Inc()
	m.HTTPRequestDuration.WithLabelValues("GET", "/health", "200").Observe(0.05)
	m.HTTPActiveRequests.Inc()
	m.HTTPActiveRequests.Dec()

	// Verify the metrics were recorded by querying the registry
	families, err := m.registry.Gather()
	if err != nil {
		t.Fatalf("failed to gather metrics: %v", err)
	}

	if len(families) == 0 {
		t.Fatal("no metric families gathered")
	}

	found := false
	for _, fam := range families {
		if fam.GetName() == "http_requests_total" {
			found = true
			if len(fam.GetMetric()) < 2 {
				t.Errorf("expected at least 2 http_requests_total metrics, got %d", len(fam.GetMetric()))
			}
		}
	}
	if !found {
		t.Error("http_requests_total metric not found")
	}
}

func TestMetricsHandler(t *testing.T) {
	m := NewMetrics()
	m.HTTPRequestsTotal.WithLabelValues("GET", "/test", "200").Inc()

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()

	handler := m.Handler()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	body := w.Body.String()
	if body == "" {
		t.Error("empty metrics response")
	}
	// Prometheus format should contain the metric name
	if !contains(body, "http_requests_total") {
		t.Errorf("response doesn't contain http_requests_total: %s", body)
	}
}

func TestMetricsDBCounters(t *testing.T) {
	m := NewMetrics()

	m.DBQueriesTotal.WithLabelValues("query").Inc()
	m.DBQueryDuration.WithLabelValues("query").Observe(0.1)
	m.DBErrorsTotal.WithLabelValues("query").Inc()
	m.CacheHitsTotal.Inc()
	m.CacheMissesTotal.Inc()

	families, err := m.registry.Gather()
	if err != nil {
		t.Fatalf("failed to gather metrics: %v", err)
	}

	names := make(map[string]bool)
	for _, fam := range families {
		names[fam.GetName()] = true
	}

	for _, expected := range []string{"db_queries_total", "db_query_duration_seconds", "db_errors_total", "cache_hits_total", "cache_misses_total"} {
		if !names[expected] {
			t.Errorf("metric %q not found", expected)
		}
	}
}

func TestMetricsWSCounters(t *testing.T) {
	m := NewMetrics()

	m.WSActiveConnections.Inc()
	m.WSActiveConnections.Dec()
	m.WSMessagesSent.Inc()
	m.WSMessagesReceived.Inc()

	families, err := m.registry.Gather()
	if err != nil {
		t.Fatalf("failed to gather metrics: %v", err)
	}

	names := make(map[string]bool)
	for _, fam := range families {
		names[fam.GetName()] = true
	}

	for _, expected := range []string{"ws_active_connections", "ws_messages_sent_total", "ws_messages_received_total"} {
		if !names[expected] {
			t.Errorf("metric %q not found", expected)
		}
	}
}

func TestMetricsBusinessGauges(t *testing.T) {
	m := NewMetrics()

	m.UsersTotal.Set(42)
	m.ShoppingListsTotal.Set(100)
	m.ShoppingItemsTotal.Set(500)

	families, err := m.registry.Gather()
	if err != nil {
		t.Fatalf("failed to gather metrics: %v", err)
	}

	for _, fam := range families {
		switch fam.GetName() {
		case "users_total":
			val := fam.GetMetric()[0].GetGauge().GetValue()
			if val != 42 {
				t.Errorf("users_total: expected 42, got %f", val)
			}
		case "shopping_lists_total":
			val := fam.GetMetric()[0].GetGauge().GetValue()
			if val != 100 {
				t.Errorf("shopping_lists_total: expected 100, got %f", val)
			}
		case "shopping_items_total":
			val := fam.GetMetric()[0].GetGauge().GetValue()
			if val != 500 {
				t.Errorf("shopping_items_total: expected 500, got %f", val)
			}
		}
	}
}

func TestHistogramBuckets(t *testing.T) {
	m := NewMetrics()

	// Observe several durations
	for _, d := range []float64{0.01, 0.05, 0.1, 0.5, 1.0} {
		m.HTTPRequestDuration.WithLabelValues("GET", "/test", "200").Observe(d)
	}

	families, err := m.registry.Gather()
	if err != nil {
		t.Fatalf("failed to gather metrics: %v", err)
	}

	for _, fam := range families {
		if fam.GetName() == "http_request_duration_seconds" {
			metric := fam.GetMetric()[0]
			hist := metric.GetHistogram()
			if hist.GetSampleCount() != 5 {
				t.Errorf("expected 5 observations, got %d", hist.GetSampleCount())
			}
			if len(hist.GetBucket()) == 0 {
				t.Error("no histogram buckets found")
			}
			return
		}
	}
	t.Error("http_request_duration_seconds not found")
}

func TestMustRegisterCustom(t *testing.T) {
	m := NewMetrics()

	collector := prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "test_custom_gauge",
		Help: "A test custom gauge",
	}, func() float64 { return 42 })

	m.MustRegisterCustom(collector)

	families, err := m.registry.Gather()
	if err != nil {
		t.Fatalf("failed to gather metrics: %v", err)
	}

	for _, fam := range families {
		if fam.GetName() == "test_custom_gauge" {
			val := fam.GetMetric()[0].GetGauge().GetValue()
			if val != 42 {
				t.Errorf("expected 42, got %f", val)
			}
			return
		}
	}
	t.Error("test_custom_gauge not found")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
