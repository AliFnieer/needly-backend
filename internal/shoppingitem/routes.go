package shoppingitem

import (
	"github.com/AliFnieer/needly-backend/internal/cache"
	"github.com/AliFnieer/needly-backend/internal/config"
	"github.com/AliFnieer/needly-backend/internal/history"
	"github.com/AliFnieer/needly-backend/internal/middleware"
	"github.com/AliFnieer/needly-backend/internal/notification"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterRoutes registers shopping item routes on the given router group.
func RegisterRoutes(router *gin.RouterGroup, db *gorm.DB, cfg *config.Config, cache *cache.Cache, notificationSvc *notification.Service) {
	historySvc := history.NewService(db)
	service := NewService(db, cache, historySvc, notificationSvc)
	controller := NewController(service)

	// All shopping item routes require authentication + household membership
	itemRoutes := router.Group("")
	itemRoutes.Use(middleware.AuthMiddleware(cfg))
	{
		itemRoutes.POST("/lists/:id/items",
			middleware.RequireMembership(db, middleware.HouseholdFromList(db)),
			controller.Create)
		itemRoutes.GET("/lists/:id/items",
			middleware.RequireMembership(db, middleware.HouseholdFromList(db)),
			controller.List)
		itemRoutes.GET("/items/:id",
			middleware.RequireMembership(db, middleware.HouseholdFromItem(db)),
			controller.GetByID)
		itemRoutes.PUT("/items/:id",
			middleware.RequireMembership(db, middleware.HouseholdFromItem(db)),
			controller.Update)
		itemRoutes.PATCH("/items/:id/completed",
			middleware.RequireMembership(db, middleware.HouseholdFromItem(db)),
			controller.SetCompleted)
		itemRoutes.DELETE("/items/:id",
			middleware.RequireMembership(db, middleware.HouseholdFromItem(db)),
			controller.Delete)
		itemRoutes.POST("/history/:id/re-add",
			middleware.RequireMembership(db, middleware.HouseholdFromHistory(db)),
			controller.ReAddFromHistory)
	}
}
