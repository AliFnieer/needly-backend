package websocket

import "github.com/gin-gonic/gin"

// RegisterRoutes registers websocket endpoints under the given router group.
func RegisterRoutes(rg *gin.RouterGroup, hub *Hub) {
    // WebSocket connection for a specific household
    rg.GET("/ws/:household_id", func(c *gin.Context) {
        ServeWS(hub, c)
    })

    // Generic ws without household id
    rg.GET("/ws", func(c *gin.Context) {
        ServeWS(hub, c)
    })
}
