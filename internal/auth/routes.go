package auth

import (
	"github.com/AliFnieer/needly-backend/internal/config"
	"github.com/AliFnieer/needly-backend/internal/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterRoutes registers auth routes on the given router group.
func RegisterRoutes(router *gin.RouterGroup, db *gorm.DB, cfg *config.Config) {
	service := NewService(db, cfg)
	controller := NewController(service)

	// Public routes
	authGroup := router.Group("/auth")
	{
		authGroup.POST("/register", controller.Register)
		authGroup.POST("/login", controller.Login)
	}

	// Protected routes (require JWT)
	protected := router.Group("")
	protected.Use(middleware.AuthMiddleware(cfg))
	{
		protected.GET("/auth/me", controller.Me)
	}
}