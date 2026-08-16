package server

import (
	"github.com/AliFnieer/needly-backend/internal/auth"
	"github.com/AliFnieer/needly-backend/internal/config"
	"github.com/AliFnieer/needly-backend/internal/household"
	"github.com/AliFnieer/needly-backend/internal/middleware"
	"github.com/AliFnieer/needly-backend/internal/shoppingitem"
	"github.com/AliFnieer/needly-backend/internal/shoppinglist"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// Server represents the HTTP server with its dependencies.
type Server struct {
	engine *gin.Engine
	cfg    *config.Config
	db     *gorm.DB
	redis  *redis.Client
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
	}

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
	household.RegisterRoutes(apiV1, s.db, s.cfg)
	shoppinglist.RegisterRoutes(apiV1, s.db, s.cfg)
	shoppingitem.RegisterRoutes(apiV1, s.db, s.cfg)
}

// Run starts the HTTP server on the given address.
func (s *Server) Run(addr string) error {
	return s.engine.Run(addr)
}