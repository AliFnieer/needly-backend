package sync

import (
	"github.com/AliFnieer/needly-backend/internal/config"
	"github.com/AliFnieer/needly-backend/internal/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterRoutes registers sync routes on the given router group.
func RegisterRoutes(router *gin.RouterGroup, db *gorm.DB, cfg *config.Config) {
	service := NewService(db)
	controller := NewController(service)

	// Sync routes require authentication + household membership
	syncRoutes := router.Group("")
	syncRoutes.Use(middleware.AuthMiddleware(cfg))
	{
		syncRoutes.GET("/households/:id/sync",
			middleware.RequireMembership(db, middleware.HouseholdFromParam("id")),
			controller.Sync)
	}
}
