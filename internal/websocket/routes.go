package websocket

import (
	"github.com/AliFnieer/needly-backend/internal/config"
	"github.com/AliFnieer/needly-backend/internal/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterRoutes registers authenticated websocket endpoints under the given router group.
func RegisterRoutes(rg *gin.RouterGroup, hub *Hub, db *gorm.DB, cfg *config.Config) {
	ws := rg.Group("/ws")
	ws.Use(middleware.AuthMiddleware(cfg))
	{
		// WebSocket connection for a specific household (requires membership)
		ws.GET("/:household_id", func(c *gin.Context) {
			ServeWS(hub, c, db, cfg)
		})
	}
}
