package shoppingitem

import (
	"github.com/AliFnieer/needly-backend/internal/config"
	"github.com/AliFnieer/needly-backend/internal/history"
	"github.com/AliFnieer/needly-backend/internal/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterRoutes registers shopping item routes on the given router group.
func RegisterRoutes(router *gin.RouterGroup, db *gorm.DB, cfg *config.Config) {
	historySvc := history.NewService(db)
	service := NewService(db, historySvc)
	controller := NewController(service)

	// All shopping item routes require authentication
	itemRoutes := router.Group("")
	itemRoutes.Use(middleware.AuthMiddleware(cfg))
	{
		itemRoutes.POST("/lists/:id/items", controller.Create)
		itemRoutes.GET("/lists/:id/items", controller.List)
		itemRoutes.GET("/items/:id", controller.GetByID)
		itemRoutes.PUT("/items/:id", controller.Update)
		itemRoutes.PATCH("/items/:id/completed", controller.SetCompleted)
		itemRoutes.DELETE("/items/:id", controller.Delete)
		itemRoutes.POST("/history/:id/re-add", controller.ReAddFromHistory)
	}
}