package server

import (
	"context"
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
	"github.com/AliFnieer/needly-backend/internal/websocket"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

const (
	// defaultCacheTTL is the default time-to-live for cached data.
	defaultCacheTTL = 5 * time.Minute
)

// Server represents the HTTP server with its dependencies.
type Server struct {
	engine *gin.Engine
	cfg    *config.Config
	db     *gorm.DB
	redis  *redis.Client
	cache  *cache.Cache
	hub    *websocket.Hub

	rateLimiter    *middleware.RateLimiter
	notificationSvc *notification.Service
	obsMiddleware  *observability.Middleware
	metrics        *observability.Metrics
	tracer         *observability.Tracer
	logger         *observability.Logger
}

// NewServer creates a new Gin server with all routes registered.
func NewServer(cfg *config.Config, db *gorm.DB, redisClient *redis.Client) *Server {
	// Set Gin mode
	gin.SetMode(cfg.Server.GinMode)

	engine := gin.New()

	srv := &Server{
		engine: engine,
		cfg:    cfg,
		db:     db,
		redis:  redisClient,
		cache:  cache.NewCache(redisClient, defaultCacheTTL),
	}

	// Initialize observability (logging, metrics, tracing)
	srv.logger = observability.NewLogger("info", nil)
	srv.metrics = observability.NewMetrics()
	srv.tracer = observability.NewTracer()
	srv.obsMiddleware = observability.NewMiddleware(srv.logger, srv.metrics, srv.tracer)

	// Create and start websocket hub with Redis Pub/Sub distribution
	srv.hub = websocket.NewHub(redisClient)
	go srv.hub.Run()

	// Create the rate limiter backed by Redis
	srv.rateLimiter = middleware.NewRateLimiter(redisClient, &cfg.RateLimit)

	// Create the push notification service backed by the WebSocket hub and Redis
	srv.notificationSvc = notification.NewService(srv.hub, redisClient, &cfg.Notification)

	// Global middleware
	srv.setupMiddleware()

	// Register routes
	srv.setupRoutes()

	return srv
}

// setupMiddleware configures global middleware.
func (s *Server) setupMiddleware() {
	// Request timeout — applied early so it covers handler execution
	s.engine.Use(middleware.RequestTimeoutMiddleware(30 * time.Second))

	// Request ID — first so it's available to all downstream middleware/logs
	s.engine.Use(middleware.RequestIDMiddleware())

	// Observability (logging + metrics + tracing) — uses request_id
	s.engine.Use(s.obsMiddleware.GinMiddleware())

	s.engine.Use(gin.Recovery())

	// Security headers
	s.engine.Use(middleware.SecurityMiddleware(s.cfg.Server.GinMode))

	// Response compression (gzip for clients that accept it)
	s.engine.Use(middleware.CompressionMiddleware())

	// CORS middleware
	s.engine.Use(s.corsMiddleware())

	// Redis-backed rate limiting applied to all requests
	s.engine.Use(s.rateLimiter.Middleware())

	// API version header
	s.engine.Use(middleware.VersionMiddleware())

	// Circuit breaker for downstream protection
	cb := middleware.NewCircuitBreaker(&middleware.CircuitBreakerConfig{
		Threshold:    5,
		ResetTimeout: 30 * time.Second,
		HalfOpenMax:  2,
	})
	s.engine.Use(cb.Middleware())
}

// setupRoutes registers all API route groups.
func (s *Server) setupRoutes() {
	// Health check — verifies DB and Redis connectivity
	s.engine.GET("/health", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()

		status := gin.H{"status": "ok"}
		healthy := true

		// Check database
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

		// Check Redis
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

	isRelease := s.cfg.Server.GinMode == gin.ReleaseMode

	// Internal observability endpoints — protected in release mode
	internal := s.engine.Group("")
	if isRelease {
		internal.Use(middleware.AuthMiddleware(s.cfg))
	}
	{
		// Metrics endpoint
		internal.GET("/metrics", gin.WrapH(s.metrics.MetricsHandler()))

		// Tracing summary endpoint
		internal.GET("/debug/traces", gin.WrapH(s.tracer.TraceHandler()))
	}

	// API documentation (always public)
	s.engine.GET("/docs", gin.WrapH(docs.SwaggerUIHandler()))
	s.engine.GET("/docs/openapi.json", gin.WrapH(http.HandlerFunc(docs.ServeOpenAPIHandler)))

	// API v1 routes
	apiV1 := s.engine.Group("/api/v1")

	// Register feature routes
	auth.RegisterRoutes(apiV1, s.db, s.cfg)
	category.RegisterRoutes(apiV1, s.db, s.cfg)
	history.RegisterRoutes(apiV1, s.db, s.cfg)
	household.RegisterRoutes(apiV1, s.db, s.cfg, s.notificationSvc)
	shoppinglist.RegisterRoutes(apiV1, s.db, s.cfg, s.cache, s.notificationSvc)
	shoppingitem.RegisterRoutes(apiV1, s.db, s.cfg, s.cache, s.notificationSvc)
	notification.RegisterRoutes(apiV1, s.notificationSvc, s.cfg, s.db)

	// Register websocket routes
	websocket.RegisterRoutes(apiV1, s.hub, s.db, s.cfg)
}

// Engine returns the underlying Gin engine (for external http.Server wrapping).
func (s *Server) Engine() *gin.Engine {
	return s.engine
}