package category

import (
	"github.com/AliFnieer/needly-backend/internal/cache"
	"github.com/AliFnieer/needly-backend/internal/config"
	"github.com/AliFnieer/needly-backend/internal/idempotency"
	"github.com/AliFnieer/needly-backend/internal/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterRoutes registers category routes scoped to a household.
func RegisterRoutes(router *gin.RouterGroup, db *gorm.DB, cfg *config.Config, cacheClient *cache.Cache) {
	service := NewService(db, cacheClient)
	controller := NewController(service)

	// Categories are scoped to a household and require membership
	categoryGroup := router.Group("/households/:id/categories")
	categoryGroup.Use(middleware.AuthMiddleware(cfg))
	categoryGroup.Use(idempotency.Middleware(db))
	categoryGroup.Use(middleware.RequireMembership(db, middleware.HouseholdFromParam("id")))
	{
		categoryGroup.GET("", controller.List)
		categoryGroup.POST("", controller.Create)
		categoryGroup.GET("/:categoryId", controller.GetByID)
		categoryGroup.PUT("/:categoryId", controller.Update)
		categoryGroup.DELETE("/:categoryId", controller.Delete)
	}
}
