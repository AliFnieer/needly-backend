package category

import (
	"github.com/AliFnieer/needly-backend/internal/config"
	"github.com/AliFnieer/needly-backend/internal/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterRoutes registers category routes on the given router group.
func RegisterRoutes(router *gin.RouterGroup, db *gorm.DB, cfg *config.Config) {
	service := NewService(db)
	controller := NewController(service)

	// All category routes require authentication
	categoryGroup := router.Group("/categories")
	categoryGroup.Use(middleware.AuthMiddleware(cfg))
	{
		categoryGroup.GET("", controller.List)
		categoryGroup.POST("", controller.Create)
		categoryGroup.GET("/:id", controller.GetByID)
		categoryGroup.PUT("/:id", controller.Update)
		categoryGroup.DELETE("/:id", controller.Delete)
	}
}