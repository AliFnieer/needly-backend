package notification

import (
	"github.com/AliFnieer/needly-backend/internal/config"
	"github.com/AliFnieer/needly-backend/internal/middleware"
	"github.com/gin-gonic/gin"
)

// RegisterRoutes registers notification routes on the given router group.
func RegisterRoutes(router *gin.RouterGroup, service *Service, cfg *config.Config) {
	controller := NewController(service)

	// All notification routes require authentication
	notificationGroup := router.Group("/households")
	notificationGroup.Use(middleware.AuthMiddleware(cfg))
	{
		notificationGroup.GET("/:id/notifications", controller.ListHouseholdHistory)
	}
}