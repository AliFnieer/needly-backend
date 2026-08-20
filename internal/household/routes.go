package household

import (
	"github.com/AliFnieer/needly-backend/internal/cache"
	"github.com/AliFnieer/needly-backend/internal/config"
	"github.com/AliFnieer/needly-backend/internal/middleware"
	"github.com/AliFnieer/needly-backend/internal/notification"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterRoutes registers household routes on the given router group.
func RegisterRoutes(router *gin.RouterGroup, db *gorm.DB, cfg *config.Config, notificationSvc *notification.Service, cacheClient *cache.Cache) {
	service := NewService(db, notificationSvc, cacheClient)
	controller := NewController(service)

	// All household routes require authentication
	householdGroup := router.Group("/households")
	householdGroup.Use(middleware.AuthMiddleware(cfg))
	{
		householdGroup.GET("", controller.List)
		householdGroup.POST("", controller.Create)
		householdGroup.GET("/:id",
			middleware.RequireMembership(db, middleware.HouseholdFromParam("id")),
			controller.GetByID)
		householdGroup.PUT("/:id", controller.Update)
		householdGroup.DELETE("/:id", controller.Delete)
		householdGroup.POST("/:id/members", controller.AddMember)
		householdGroup.DELETE("/:id/members/:userId", controller.RemoveMember)
	}
}