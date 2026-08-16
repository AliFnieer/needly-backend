package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Controller handles auth HTTP requests.
type Controller struct {
    service *Service
}

// NewController creates a new auth controller.
func NewController(service *Service) *Controller {
    return &Controller{
        service: service,
    }
}

// Register handles POST /api/v1/auth/register
func (c *Controller) Register(ctx *gin.Context) {
    var req RegisterRequest

    // Bind and validate JSON body
    if err := ctx.ShouldBindJSON(&req); err != nil {
        ctx.JSON(http.StatusBadRequest, gin.H{
            "error": err.Error(),
        })
        return
    }

    resp, err := c.service.Register(&req)
    if err != nil {
        ctx.JSON(http.StatusConflict, gin.H{
            "error": err.Error(),
        })
        return
    }

    ctx.JSON(http.StatusCreated, resp)
}

// Login handles POST /api/v1/auth/login
func (c *Controller) Login(ctx *gin.Context) {
    var req LoginRequest

    // Bind and validate JSON body
    if err := ctx.ShouldBindJSON(&req); err != nil {
        ctx.JSON(http.StatusBadRequest, gin.H{
            "error": err.Error(),
        })
        return
    }

    resp, err := c.service.Login(&req)
    if err != nil {
        ctx.JSON(http.StatusUnauthorized, gin.H{
            "error": err.Error(),
        })
        return
    }

    ctx.JSON(http.StatusOK, resp)
}

// Me handles GET /api/v1/auth/me
func (c *Controller) Me(ctx *gin.Context) {
    // Get user ID from middleware context
    userID, exists := ctx.Get("user_id")
    if !exists {
        ctx.JSON(http.StatusUnauthorized, gin.H{
            "error": "user not authenticated",
        })
        return
    }

    user, err := c.service.GetByID(userID)
    if err != nil {
        ctx.JSON(http.StatusNotFound, gin.H{
            "error": err.Error(),
        })
        return
    }

    ctx.JSON(http.StatusOK, user)
}
