package history

import (
	"github.com/AliFnieer/needly-backend/internal/config"
	"github.com/AliFnieer/needly-backend/internal/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterRoutes registers shopping history routes on the given router group.
func RegisterRoutes(router *gin.RouterGroup, db *gorm.DB, cfg *config.Config) {
	service := NewService(db)
	controller := NewController(service)

	// All history routes require authentication
	historyGroup := router.Group("")
	historyGroup.Use(middleware.AuthMiddleware(cfg))
	{
		historyGroup.GET("/lists/:id/history", controller.ListByListID)
		historyGroup.GET("/households/:id/history", controller.ListByHouseholdID)
		historyGroup.GET("/history/:id", controller.GetByID)
		historyGroup.DELETE("/history/:id", controller.Delete)
	}
}