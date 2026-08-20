package notification

import (
	"github.com/AliFnieer/needly-backend/internal/config"
	"github.com/AliFnieer/needly-backend/internal/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterRoutes registers notification routes on the given router group.
func RegisterRoutes(router *gin.RouterGroup, service *Service, cfg *config.Config, db *gorm.DB) {
	controller := NewController(service)

	// All notification routes require authentication + household membership
	notificationGroup := router.Group("/households")
	notificationGroup.Use(middleware.AuthMiddleware(cfg))
	{
		notificationGroup.GET("/:id/notifications",
			middleware.RequireMembership(db, middleware.HouseholdFromParam("id")),
			controller.ListHouseholdHistory)
	}
}