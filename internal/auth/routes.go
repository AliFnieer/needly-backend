package auth

import (
	"github.com/AliFnieer/needly-backend/internal/config"
	"github.com/AliFnieer/needly-backend/internal/mailer"
	"github.com/AliFnieer/needly-backend/internal/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterRoutes registers auth routes on the given router group.
func RegisterRoutes(router *gin.RouterGroup, db *gorm.DB, cfg *config.Config, rl *middleware.RateLimiter) {
	service := NewService(db, cfg, mailer.New(cfg))
	controller := NewController(service)

	// Public routes — stricter rate limit (10 req/min)
	authGroup := router.Group("/auth")
	if rl != nil {
		authGroup.Use(rl.StrictMiddleware())
	}
	{
		authGroup.POST("/register", controller.Register)
		authGroup.POST("/login", controller.Login)
		authGroup.POST("/refresh", controller.Refresh)
		authGroup.POST("/forgot-password", controller.ForgotPassword)
		authGroup.POST("/reset-password", controller.ResetPassword)
		authGroup.GET("/verify-email", controller.VerifyEmail)
	}

	// Protected routes (require JWT) — global rate limit applies
	protected := router.Group("")
	protected.Use(middleware.AuthMiddleware(cfg))
	{
		protected.GET("/auth/me", controller.Me)
		protected.POST("/auth/logout", controller.Logout)
		protected.POST("/auth/resend-verification", controller.ResendVerification)
	}
}
