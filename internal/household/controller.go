package household

import (
	"net/http"
	"log/slog"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Controller handles household HTTP requests.
type Controller struct {
	service *Service
}

// NewController creates a new household controller.
func NewController(service *Service) *Controller {
	return &Controller{
		service: service,
	}
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

	// user_id is stored as float64 from JWT claims
	id, ok := value.(float64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "invalid user id in context",
		})
		return 0, false
	}

	return uint(id), true
}

// parseIDParam parses a uint path parameter.
func parseIDParam(c *gin.Context, name string) (uint, bool) {
	id, err := strconv.ParseUint(c.Param(name), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid " + name + " parameter",
		})
		return 0, false
	}
	return uint(id), true
}

// Create handles POST /api/v1/households
func (ctl *Controller) Create(c *gin.Context) {
	userID, ok := getCurrentUserID(c)
	if !ok {
		return
	}

	var req CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	household, err := ctl.service.Create(userID, &req)
	if err != nil {
		slog.Error("household create failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
		return
	}

	c.JSON(http.StatusCreated, household)
}

// GetByID handles GET /api/v1/households/:id
func (ctl *Controller) GetByID(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	household, err := ctl.service.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, household)
}

// List handles GET /api/v1/households
func (ctl *Controller) List(c *gin.Context) {
	userID, ok := getCurrentUserID(c)
	if !ok {
		return
	}

	households, err := ctl.service.ListByUser(userID)
	if err != nil {
		slog.Error("household list failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
		return
	}

	c.JSON(http.StatusOK, households)
}

// Update handles PUT /api/v1/households/:id
func (ctl *Controller) Update(c *gin.Context) {
	userID, ok := getCurrentUserID(c)
	if !ok {
		return
	}

	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	household, err := ctl.service.Update(id, userID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, household)
}

// Delete handles DELETE /api/v1/households/:id
func (ctl *Controller) Delete(c *gin.Context) {
	userID, ok := getCurrentUserID(c)
	if !ok {
		return
	}

	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	if err := ctl.service.Delete(id, userID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// AddMember handles POST /api/v1/households/:id/members
func (ctl *Controller) AddMember(c *gin.Context) {
	userID, ok := getCurrentUserID(c)
	if !ok {
		return
	}

	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	var req AddMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	member, err := ctl.service.AddMember(id, userID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, member)
}

// RemoveMember handles DELETE /api/v1/households/:id/members/:userId
func (ctl *Controller) RemoveMember(c *gin.Context) {
	userID, ok := getCurrentUserID(c)
	if !ok {
		return
	}

	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	memberID, ok := parseIDParam(c, "userId")
	if !ok {
		return
	}

	if err := ctl.service.RemoveMember(id, userID, memberID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}