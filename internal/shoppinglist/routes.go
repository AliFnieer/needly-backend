package shoppinglist

import (
	"github.com/AliFnieer/needly-backend/internal/cache"
	"github.com/AliFnieer/needly-backend/internal/config"
	"github.com/AliFnieer/needly-backend/internal/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterRoutes registers shopping list routes on the given router group.
func RegisterRoutes(router *gin.RouterGroup, db *gorm.DB, cfg *config.Config, cache *cache.Cache) {
	service := NewService(db, cache)
	controller := NewController(service)

	// All shopping list routes require authentication
	listRoutes := router.Group("")
	listRoutes.Use(middleware.AuthMiddleware(cfg))
	{
		listRoutes.POST("/households/:id/lists", controller.Create)
		listRoutes.GET("/households/:id/lists", controller.List)
		listRoutes.GET("/lists/:id", controller.GetByID)
		listRoutes.PUT("/lists/:id", controller.Update)
		listRoutes.DELETE("/lists/:id", controller.Delete)
	}
}