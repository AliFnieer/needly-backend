package shoppinglist

import (
	"github.com/AliFnieer/needly-backend/internal/cache"
	"github.com/AliFnieer/needly-backend/internal/config"
	"github.com/AliFnieer/needly-backend/internal/idempotency"
	"github.com/AliFnieer/needly-backend/internal/middleware"
	"github.com/AliFnieer/needly-backend/internal/notification"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterRoutes registers shopping list routes on the given router group.
func RegisterRoutes(router *gin.RouterGroup, db *gorm.DB, cfg *config.Config, cache *cache.Cache, notificationSvc *notification.Service) {
	service := NewService(db, cache, notificationSvc)
	controller := NewController(service)

	// All shopping list routes require authentication + household membership
	listRoutes := router.Group("")
	listRoutes.Use(middleware.AuthMiddleware(cfg))
	listRoutes.Use(idempotency.Middleware(db))
	{
		listRoutes.POST("/households/:id/lists",
			middleware.RequireMembership(db, middleware.HouseholdFromParam("id")),
			controller.Create)
		listRoutes.GET("/households/:id/lists",
			middleware.RequireMembership(db, middleware.HouseholdFromParam("id")),
			controller.List)
		listRoutes.GET("/lists/:id",
			middleware.RequireMembership(db, middleware.HouseholdFromList(db)),
			controller.GetByID)
		listRoutes.PUT("/lists/:id",
			middleware.RequireMembership(db, middleware.HouseholdFromList(db)),
			controller.Update)
		listRoutes.DELETE("/lists/:id",
			middleware.RequireMembership(db, middleware.HouseholdFromList(db)),
			controller.Delete)
	}
}
