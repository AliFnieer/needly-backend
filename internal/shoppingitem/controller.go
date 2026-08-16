package shoppingitem

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Controller handles shopping item HTTP requests.
type Controller struct {
	service *Service
}

// NewController creates a new shopping item controller.
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


	
	id, ok := value.(float64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "invalid user id in context",
		})
		return 0, false
	}

	return uint(id), true
}

// parseUintParam parses a uint path parameter.
func parseUintParam(c *gin.Context, name string) (uint, bool) {
	value, err := strconv.ParseUint(c.Param(name), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid " + name + " parameter",
		})
		return 0, false
	}
	return uint(value), true
}

// Create handles POST /api/v1/lists/:id/items
func (ctl *Controller) Create(c *gin.Context) {
	userID, ok := getCurrentUserID(c)
	if !ok {
		return
	}

	listID, ok := parseUintParam(c, "id")
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

	item, err := ctl.service.Create(listID, userID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, item)
}

// List handles GET /api/v1/lists/:id/items
func (ctl *Controller) List(c *gin.Context) {
	listID, ok := parseUintParam(c, "id")
	if !ok {
		return
	}

	items, err := ctl.service.ListByListID(listID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, items)
}

// GetByID handles GET /api/v1/items/:id
func (ctl *Controller) GetByID(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}

	item, err := ctl.service.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, item)
}

// Update handles PUT /api/v1/items/:id
func (ctl *Controller) Update(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
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

	item, err := ctl.service.Update(id, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, item)
}

// SetCompleted handles PATCH /api/v1/items/:id/completed
func (ctl *Controller) SetCompleted(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}

	var req struct {
		IsCompleted bool `json:"is_completed" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	item, err := ctl.service.UpdateCompleted(id, req.IsCompleted)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, item)
}

// Delete handles DELETE /api/v1/items/:id
func (ctl *Controller) Delete(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}

	if err := ctl.service.Delete(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}