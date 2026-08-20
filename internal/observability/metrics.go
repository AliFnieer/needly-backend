package observability

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics holds all Prometheus metric collectors for the application.
type Metrics struct {
	// HTTP metrics
	HTTPRequestsTotal   *prometheus.CounterVec
	HTTPRequestDuration *prometheus.HistogramVec
	HTTPActiveRequests  prometheus.Gauge

	// Database metrics
	DBQueriesTotal   *prometheus.CounterVec
	DBQueryDuration *prometheus.HistogramVec
	DBErrorsTotal   *prometheus.CounterVec

	// Cache metrics
	CacheHitsTotal   prometheus.Counter
	CacheMissesTotal prometheus.Counter

	// WebSocket metrics
	WSActiveConnections prometheus.Gauge
	WSMessagesSent      prometheus.Counter
	WSMessagesReceived  prometheus.Counter

	// Business metrics (gauges set by periodic collector)
	UsersTotal         prometheus.Gauge
	ShoppingListsTotal prometheus.Gauge
	ShoppingItemsTotal prometheus.Gauge

	registry *prometheus.Registry
}

// NewMetrics creates and registers all Prometheus metrics.
func NewMetrics() *Metrics {
	reg := prometheus.NewRegistry()
	reg.MustRegister(prometheus.NewGoCollector())
	reg.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))

	m := &Metrics{
		HTTPRequestsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		}, []string{"method", "path", "status"}),

		HTTPRequestDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		}, []string{"method", "path", "status"}),

		HTTPActiveRequests: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "http_active_requests",
			Help: "Number of currently active HTTP requests",
		}),

		DBQueriesTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "db_queries_total",
			Help: "Total number of database queries",
		}, []string{"operation"}),

		DBQueryDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "db_query_duration_seconds",
			Help:    "Database query duration in seconds",
			Buckets: prometheus.DefBuckets,
		}, []string{"operation"}),

		DBErrorsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "db_errors_total",
			Help: "Total number of database errors",
		}, []string{"operation"}),

		CacheHitsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "cache_hits_total",
			Help: "Total number of cache hits",
		}),

		CacheMissesTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "cache_misses_total",
			Help: "Total number of cache misses",
		}),

		WSActiveConnections: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "ws_active_connections",
			Help: "Number of active WebSocket connections",
		}),

		WSMessagesSent: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "ws_messages_sent_total",
			Help: "Total WebSocket messages sent",
		}),

		WSMessagesReceived: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "ws_messages_received_total",
			Help: "Total WebSocket messages received",
		}),

		UsersTotal: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "users_total",
			Help: "Total number of users",
		}),

		ShoppingListsTotal: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "shopping_lists_total",
			Help: "Total number of shopping lists",
		}),

		ShoppingItemsTotal: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "shopping_items_total",
			Help: "Total number of shopping items",
		}),

		registry: reg,
	}

	// Register all collectors
	reg.MustRegister(
		m.HTTPRequestsTotal,
		m.HTTPRequestDuration,
		m.HTTPActiveRequests,
		m.DBQueriesTotal,
		m.DBQueryDuration,
		m.DBErrorsTotal,
		m.CacheHitsTotal,
		m.CacheMissesTotal,
		m.WSActiveConnections,
		m.WSMessagesSent,
		m.WSMessagesReceived,
		m.UsersTotal,
		m.ShoppingListsTotal,
		m.ShoppingItemsTotal,
	)

	return m
}

// Handler returns an HTTP handler that serves Prometheus metrics.
func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	})
}

// MustRegisterCustom registers a custom collector (e.g. DB pool stats).
func (m *Metrics) MustRegisterCustom(c prometheus.Collector) {
	m.registry.MustRegister(c)
}
