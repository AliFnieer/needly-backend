package observability

import (
	"net/http"
	"strconv"
	"sync"
	"time"
)

// Metrics provides lightweight in-memory metrics collection without external dependencies.
// This can be replaced with Prometheus client library if needed.
type Metrics struct {
	mu sync.RWMutex

	// HTTP metrics
	httpRequestsTotal   map[string]int64     // method:path -> count
	httpRequestDuration map[string][]float64 // method:path -> durations in seconds
	httpActiveRequests  int64

	// Database metrics
	dbQueriesTotal   map[string]int64 // operation -> count
	dbQueryDuration  map[string][]float64
	dbErrorsTotal    map[string]int64

	// Cache metrics
	cacheHits   int64
	cacheMisses int64

	// WebSocket metrics
	wsConnectionsTotal   int64
	wsActiveConnections  int64
	wsMessagesSentTotal  int64
	wsMessagesRecvTotal  int64

	// Business metrics
	usersTotal        int64
	shoppingListsTotal int64
	shoppingItemsTotal int64
}

// NewMetrics creates a new metrics collector.
func NewMetrics() *Metrics {
	return &Metrics{
		httpRequestsTotal:   make(map[string]int64),
		httpRequestDuration: make(map[string][]float64),
		dbQueriesTotal:      make(map[string]int64),
		dbQueryDuration:     make(map[string][]float64),
		dbErrorsTotal:       make(map[string]int64),
	}
}

// IncHTTPRequest increments the request counter for a method+path.
func (m *Metrics) IncHTTPRequest(method, path string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := method + " " + path
	m.httpRequestsTotal[key]++
	m.httpActiveRequests++
}

// DecHTTPRequest decrements the active request counter.
func (m *Metrics) DecHTTPRequest() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.httpActiveRequests > 0 {
		m.httpActiveRequests--
	}
}

// ObserveHTTPDuration records the duration of an HTTP request.
func (m *Metrics) ObserveHTTPDuration(method, path string, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := method + " " + path
	m.httpRequestDuration[key] = append(m.httpRequestDuration[key], duration.Seconds())
}

// IncDBQuery increments the query counter for an operation.
func (m *Metrics) IncDBQuery(operation string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.dbQueriesTotal[operation]++
}

// ObserveDBDuration records the duration of a database query.
func (m *Metrics) ObserveDBDuration(operation string, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.dbQueryDuration[operation] = append(m.dbQueryDuration[operation], duration.Seconds())
}

// IncDBError increments the error counter for an operation.
func (m *Metrics) IncDBError(operation string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.dbErrorsTotal[operation]++
}

// IncCacheHit increments the cache hit counter.
func (m *Metrics) IncCacheHit() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cacheHits++
}

// IncCacheMiss increments the cache miss counter.
func (m *Metrics) IncCacheMiss() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cacheMisses++
}

// IncWSConnection increments the WebSocket connection counter.
func (m *Metrics) IncWSConnection() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.wsConnectionsTotal++
	m.wsActiveConnections++
}

// DecWSConnection decrements the active WebSocket connection counter.
func (m *Metrics) DecWSConnection() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.wsActiveConnections > 0 {
		m.wsActiveConnections--
	}
}

// IncWSMessageSent increments the WebSocket message sent counter.
func (m *Metrics) IncWSMessageSent() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.wsMessagesSentTotal++
}

// IncWSMessageRecv increments the WebSocket message received counter.
func (m *Metrics) IncWSMessageRecv() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.wsMessagesRecvTotal++
}

// SetUsersTotal sets the total number of users.
func (m *Metrics) SetUsersTotal(count int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.usersTotal = count
}

// SetShoppingListsTotal sets the total number of shopping lists.
func (m *Metrics) SetShoppingListsTotal(count int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shoppingListsTotal = count
}

// SetShoppingItemsTotal sets the total number of shopping items.
func (m *Metrics) SetShoppingItemsTotal(count int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shoppingItemsTotal = count
}

// Snapshot returns a copy of all metrics for reporting.
func (m *Metrics) Snapshot() map[string]any {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Calculate averages for durations
	httpAvg := make(map[string]float64)
	for k, v := range m.httpRequestDuration {
		if len(v) > 0 {
			var sum float64
			for _, d := range v {
				sum += d
			}
			httpAvg[k] = sum / float64(len(v))
		}
	}

	dbAvg := make(map[string]float64)
	for k, v := range m.dbQueryDuration {
		if len(v) > 0 {
			var sum float64
			for _, d := range v {
				sum += d
			}
			dbAvg[k] = sum / float64(len(v))
		}
	}

	return map[string]any{
		"http": map[string]any{
			"requests_total":    m.httpRequestsTotal,
			"request_duration_avg": httpAvg,
			"active_requests":   m.httpActiveRequests,
		},
		"db": map[string]any{
			"queries_total":  m.dbQueriesTotal,
			"query_duration_avg": dbAvg,
			"errors_total":   m.dbErrorsTotal,
		},
		"cache": map[string]any{
			"hits":   m.cacheHits,
			"misses": m.cacheMisses,
			"hit_ratio": func() float64 {
				total := m.cacheHits + m.cacheMisses
				if total == 0 {
					return 0
				}
				return float64(m.cacheHits) / float64(total)
			}(),
		},
		"websocket": map[string]any{
			"connections_total":  m.wsConnectionsTotal,
			"active_connections": m.wsActiveConnections,
			"messages_sent":      m.wsMessagesSentTotal,
			"messages_received":  m.wsMessagesRecvTotal,
		},
		"business": map[string]any{
			"users_total":         m.usersTotal,
			"shopping_lists_total": m.shoppingListsTotal,
			"shopping_items_total": m.shoppingItemsTotal,
		},
	}
}

// MetricsHandler returns an HTTP handler that exposes metrics in Prometheus text format.
func (m *Metrics) MetricsHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		snapshot := m.Snapshot()

		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

		// HTTP metrics
		httpData := snapshot["http"].(map[string]any)
		requests := httpData["requests_total"].(map[string]int64)
		for k, v := range requests {
			// Sanitize key for Prometheus label format
			parts := splitMethodPath(k)
			if len(parts) == 2 {
				writeMetric(w, "http_requests_total", v, map[string]string{"method": parts[0], "path": parts[1]})
			}
		}
		writeMetric(w, "http_active_requests", httpData["active_requests"].(int64), nil)

		// DB metrics
		dbData := snapshot["db"].(map[string]any)
		queries := dbData["queries_total"].(map[string]int64)
		for k, v := range queries {
			writeMetric(w, "db_queries_total", v, map[string]string{"operation": k})
		}
		errors := dbData["errors_total"].(map[string]int64)
		for k, v := range errors {
			writeMetric(w, "db_errors_total", v, map[string]string{"operation": k})
		}

		// Cache metrics
		cacheData := snapshot["cache"].(map[string]any)
		writeMetric(w, "cache_hits_total", cacheData["hits"].(int64), nil)
		writeMetric(w, "cache_misses_total", cacheData["misses"].(int64), nil)
		writeMetric(w, "cache_hit_ratio", cacheData["hit_ratio"].(float64), nil)

		// WebSocket metrics
		wsData := snapshot["websocket"].(map[string]any)
		writeMetric(w, "ws_connections_total", wsData["connections_total"].(int64), nil)
		writeMetric(w, "ws_active_connections", wsData["active_connections"].(int64), nil)
		writeMetric(w, "ws_messages_sent_total", wsData["messages_sent"].(int64), nil)
		writeMetric(w, "ws_messages_received_total", wsData["messages_received"].(int64), nil)

		// Business metrics
		bizData := snapshot["business"].(map[string]any)
		writeMetric(w, "users_total", bizData["users_total"].(int64), nil)
		writeMetric(w, "shopping_lists_total", bizData["shopping_lists_total"].(int64), nil)
		writeMetric(w, "shopping_items_total", bizData["shopping_items_total"].(int64), nil)
	})
}

func splitMethodPath(key string) []string {
	for i := 0; i < len(key); i++ {
		if key[i] == ' ' {
			return []string{key[:i], key[i+1:]}
		}
	}
	return []string{key}
}

func writeMetric(w http.ResponseWriter, name string, value any, labels map[string]string) {
	labelStr := ""
	if len(labels) > 0 {
		parts := make([]string, 0, len(labels))
		for k, v := range labels {
			parts = append(parts, k+"=\""+v+"\"")
		}
		labelStr = "{" + joinStrings(parts, ",") + "}"
	}

	var valStr string
	switch v := value.(type) {
	case int64:
		valStr = strconv.FormatInt(v, 10)
	case float64:
		valStr = strconv.FormatFloat(v, 'f', 6, 64)
	default:
		valStr = "0"
	}

	w.Write([]byte(name + labelStr + " " + valStr + "\n"))
}

func joinStrings(parts []string, sep string) string {
	result := ""
	for i, p := range parts {
		if i > 0 {
			result += sep
		}
		result += p
	}
	return result
}