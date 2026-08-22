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

	// All history routes require authentication + household membership
	historyGroup := router.Group("")
	historyGroup.Use(middleware.AuthMiddleware(cfg))
	{
		historyGroup.GET("/lists/:id/history",
			middleware.RequireMembership(db, middleware.HouseholdFromList(db)),
			controller.ListByListID)
		historyGroup.GET("/households/:id/history",
			middleware.RequireMembership(db, middleware.HouseholdFromParam("id")),
			controller.ListByHouseholdID)
		historyGroup.GET("/history/:id",
			middleware.RequireMembership(db, middleware.HouseholdFromHistory(db)),
			controller.GetByID)
		historyGroup.DELETE("/history/:id",
			middleware.RequireMembership(db, middleware.HouseholdFromHistory(db)),
			controller.Delete)
	}
}
