package server

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"time"

	"github.com/AliFnieer/needly-backend/internal/auth"
	"github.com/AliFnieer/needly-backend/internal/cache"
	"github.com/AliFnieer/needly-backend/internal/category"
	"github.com/AliFnieer/needly-backend/internal/config"
	"github.com/AliFnieer/needly-backend/internal/docs"
	"github.com/AliFnieer/needly-backend/internal/history"
	"github.com/AliFnieer/needly-backend/internal/household"
	"github.com/AliFnieer/needly-backend/internal/middleware"
	"github.com/AliFnieer/needly-backend/internal/notification"
	"github.com/AliFnieer/needly-backend/internal/observability"
	"github.com/AliFnieer/needly-backend/internal/shoppingitem"
	"github.com/AliFnieer/needly-backend/internal/shoppinglist"
	"github.com/AliFnieer/needly-backend/internal/sync"
	"github.com/AliFnieer/needly-backend/internal/websocket"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

const (
	defaultCacheTTL = 5 * time.Minute
)

type Server struct {
	engine *gin.Engine
	cfg    *config.Config
	db     *gorm.DB
	redis  *redis.Client
	cache  *cache.Cache
	hub    *websocket.Hub

	rateLimiter     *middleware.RateLimiter
	notificationSvc *notification.Service
	obsMiddleware   *observability.Middleware
	metrics         *observability.Metrics
}

func NewServer(cfg *config.Config, db *gorm.DB, redisClient *redis.Client) *Server {
	gin.SetMode(cfg.Server.GinMode)
	engine := gin.New()

	srv := &Server{
		engine: engine,
		cfg:    cfg,
		db:     db,
		redis:  redisClient,
		cache:  cache.NewCache(redisClient, defaultCacheTTL),
	}

	srv.metrics = observability.NewMetrics()
	srv.obsMiddleware = observability.NewMiddleware(srv.metrics)

	// Register DB pool stats as a custom Prometheus collector
	srv.registerDBPoolStats()

	srv.hub = websocket.NewHub(redisClient)
	go srv.hub.Run()

	srv.rateLimiter = middleware.NewRateLimiter(redisClient, &cfg.RateLimit)
	srv.notificationSvc = notification.NewService(srv.hub, redisClient, &cfg.Notification)

	srv.setupMiddleware()
	srv.setupRoutes()

	return srv
}

func (s *Server) registerDBPoolStats() {
	sqlDB, err := s.db.DB()
	if err != nil {
		slog.Warn("failed to get sql.DB for pool stats", "error", err)
		return
	}

	statsCollector := &dbPoolStatsCollector{db: sqlDB}
	s.metrics.MustRegisterCustom(statsCollector)
	slog.Info("database pool stats collector registered")
}

type dbPoolStatsCollector struct {
	db *sql.DB
}

func (c *dbPoolStatsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- prometheus.NewDesc("db_pool_open_connections", "Number of open database connections", nil, nil)
	ch <- prometheus.NewDesc("db_pool_in_use", "Number of connections currently in use", nil, nil)
	ch <- prometheus.NewDesc("db_pool_idle", "Number of idle database connections", nil, nil)
	ch <- prometheus.NewDesc("db_pool_wait_count", "Total number of connections waited for", nil, nil)
	ch <- prometheus.NewDesc("db_pool_wait_duration_seconds", "Total time waited for connections", nil, nil)
	ch <- prometheus.NewDesc("db_pool_max_idle_closed", "Connections closed because of max idle", nil, nil)
	ch <- prometheus.NewDesc("db_pool_max_lifetime_closed", "Connections closed because of max lifetime", nil, nil)
}

func (c *dbPoolStatsCollector) Collect(ch chan<- prometheus.Metric) {
	stats := c.db.Stats()

	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc("db_pool_open_connections", "Number of open database connections", nil, nil),
		prometheus.GaugeValue, float64(stats.OpenConnections),
	)
	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc("db_pool_in_use", "Number of connections currently in use", nil, nil),
		prometheus.GaugeValue, float64(stats.InUse),
	)
	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc("db_pool_idle", "Number of idle database connections", nil, nil),
		prometheus.GaugeValue, float64(stats.Idle),
	)
	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc("db_pool_wait_count", "Total number of connections waited for", nil, nil),
		prometheus.CounterValue, float64(stats.WaitCount),
	)
	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc("db_pool_wait_duration_seconds", "Total time waited for connections", nil, nil),
		prometheus.CounterValue, stats.WaitDuration.Seconds(),
	)
	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc("db_pool_max_idle_closed", "Connections closed because of max idle", nil, nil),
		prometheus.CounterValue, float64(stats.MaxIdleClosed),
	)
	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc("db_pool_max_lifetime_closed", "Connections closed because of max lifetime", nil, nil),
		prometheus.CounterValue, float64(stats.MaxLifetimeClosed),
	)
}

func (s *Server) setupMiddleware() {
	s.engine.Use(middleware.RequestTimeoutMiddleware(30 * time.Second))
	s.engine.Use(middleware.RequestIDMiddleware())
	s.engine.Use(s.obsMiddleware.GinMiddleware())
	s.engine.Use(gin.Recovery())
	s.engine.Use(middleware.SecurityMiddleware(s.cfg.Server.GinMode))
	s.engine.Use(middleware.CompressionMiddleware())
	s.engine.Use(s.corsMiddleware())
	s.engine.Use(s.rateLimiter.Middleware())
	s.engine.Use(middleware.APIVersionMiddleware())

}

func (s *Server) setupRoutes() {
	s.engine.GET("/health", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()

		status := gin.H{"status": "ok"}
		healthy := true

		sqlDB, err := s.db.DB()
		if err != nil {
			status["database"] = "error"
			healthy = false
		} else if err := sqlDB.PingContext(ctx); err != nil {
			status["database"] = "error"
			healthy = false
		} else {
			status["database"] = "ok"
		}

		if err := s.redis.Ping(ctx).Err(); err != nil {
			status["redis"] = "error"
			healthy = false
		} else {
			status["redis"] = "ok"
		}

		httpStatus := http.StatusOK
		if !healthy {
			status["status"] = "degraded"
			httpStatus = http.StatusServiceUnavailable
		}
		c.JSON(httpStatus, status)
	})

	// Internal endpoints — auth-protected
	internal := s.engine.Group("")
	internal.Use(middleware.AuthMiddleware(s.cfg))
	{
		internal.GET("/metrics", gin.WrapH(s.metrics.Handler()))
	}

	// API documentation
	s.engine.GET("/docs", gin.WrapH(docs.SwaggerUIHandler()))
	s.engine.GET("/docs/redoc", gin.WrapH(docs.RedocUIHandler()))
	s.engine.GET("/docs/openapi.json", gin.WrapH(http.HandlerFunc(docs.ServeOpenAPIHandler)))

	// API v1 routes — scoped circuit breaker so that a failing dependency
	// (e.g. DB outage) only affects the API group, never /health or /metrics.
	apiV1 := s.engine.Group("/api/v1")
	cb := middleware.NewCircuitBreaker(&middleware.CircuitBreakerConfig{
		Threshold:    5,
		ResetTimeout: 30 * time.Second,
		HalfOpenMax:  2,
	})
	apiV1.Use(cb.Middleware())

	auth.RegisterRoutes(apiV1, s.db, s.cfg, s.rateLimiter)
	category.RegisterRoutes(apiV1, s.db, s.cfg, s.cache)
	history.RegisterRoutes(apiV1, s.db, s.cfg)
	household.RegisterRoutes(apiV1, s.db, s.cfg, s.notificationSvc, s.cache)
	shoppinglist.RegisterRoutes(apiV1, s.db, s.cfg, s.cache, s.notificationSvc)
	shoppingitem.RegisterRoutes(apiV1, s.db, s.cfg, s.cache, s.notificationSvc)
	sync.RegisterRoutes(apiV1, s.db, s.cfg)
	notification.RegisterRoutes(apiV1, s.notificationSvc, s.cfg, s.db)
	websocket.RegisterRoutes(apiV1, s.hub, s.db, s.cfg)
}

func (s *Server) Engine() *gin.Engine {
	return s.engine
}
