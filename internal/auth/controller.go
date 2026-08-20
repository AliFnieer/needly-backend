package auth

import (
	"log/slog"
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
		msg := err.Error()
		// Keep business-level errors descriptive; hide internal errors
		if msg != "email already registered" {
			slog.Error("auth register failed", "error", err)
			msg = "internal server error"
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": msg})
			return
		}
		ctx.JSON(http.StatusConflict, gin.H{"error": msg})
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
		msg := err.Error()
		// Keep auth failures descriptive; hide internal errors
		if msg != "invalid email or password" {
			slog.Error("auth login failed", "error", err)
			msg = "internal server error"
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": msg})
			return
		}
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": msg})
		return
	}

	ctx.JSON(http.StatusOK, resp)
}

// Refresh handles POST /api/v1/auth/refresh
func (c *Controller) Refresh(ctx *gin.Context) {
	var req RefreshRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	resp, err := c.service.Refresh(&req)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"error": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, resp)
}

// Logout handles POST /api/v1/auth/logout
func (c *Controller) Logout(ctx *gin.Context) {
	userID, ok := getCurrentUserID(ctx)
	if !ok {
		return
	}

	var req LogoutRequest
	// Body is optional — may be empty for "logout all"
	_ = ctx.ShouldBindJSON(&req)

	if err := c.service.Logout(userID, &req); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "logged out successfully",
	})
}

// getCurrentUserID extracts the authenticated user ID from context.
func getCurrentUserID(c *gin.Context) (uint, bool) {
	value, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "user not authenticated",
		})
		return 0, false
	}

	id, ok := value.(float64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "invalid user id in context",
		})
		return 0, false
	}

	return uint(id), true
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
