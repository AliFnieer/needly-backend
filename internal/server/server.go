package server

import (
	"time"

	"github.com/AliFnieer/needly-backend/internal/auth"
	"github.com/AliFnieer/needly-backend/internal/cache"
	"github.com/AliFnieer/needly-backend/internal/category"
	"github.com/AliFnieer/needly-backend/internal/config"
	"github.com/AliFnieer/needly-backend/internal/history"
	"github.com/AliFnieer/needly-backend/internal/household"
	"github.com/AliFnieer/needly-backend/internal/middleware"
	"github.com/AliFnieer/needly-backend/internal/notification"
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
	s.engine.Use(middleware.LoggerMiddleware())
	s.engine.Use(gin.Recovery())

	// CORS middleware
	s.engine.Use(s.corsMiddleware())

	// Redis-backed rate limiting applied to all requests
	s.engine.Use(s.rateLimiter.Middleware())
}

// setupRoutes registers all API route groups.
func (s *Server) setupRoutes() {
	// Health check
	s.engine.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
		})
	})

	// API v1 routes
	apiV1 := s.engine.Group("/api/v1")

	// Register feature routes
	auth.RegisterRoutes(apiV1, s.db, s.cfg)
	category.RegisterRoutes(apiV1, s.db, s.cfg)
	history.RegisterRoutes(apiV1, s.db, s.cfg)
	household.RegisterRoutes(apiV1, s.db, s.cfg, s.notificationSvc)
	shoppinglist.RegisterRoutes(apiV1, s.db, s.cfg, s.cache, s.notificationSvc)
	shoppingitem.RegisterRoutes(apiV1, s.db, s.cfg, s.cache, s.notificationSvc)
	notification.RegisterRoutes(apiV1, s.notificationSvc, s.cfg)

	// Register websocket routes
	websocket.RegisterRoutes(apiV1, s.hub)
}

// Run starts the HTTP server on the given address.
func (s *Server) Run(addr string) error {
	return s.engine.Run(addr)
}