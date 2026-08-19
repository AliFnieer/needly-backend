package shoppinglist

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Controller handles shopping list HTTP requests.
type Controller struct {
	service *Service
}

// NewController creates a new shopping list controller.
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

// Create handles POST /api/v1/households/:id/lists
func (ctl *Controller) Create(c *gin.Context) {
	userID, ok := getCurrentUserID(c)
	if !ok {
		return
	}

	householdID, ok := parseUintParam(c, "id")
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

	list, err := ctl.service.Create(householdID, userID, &req)
	if err != nil {
		log.Printf("shopping list create: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
		return
	}

	c.JSON(http.StatusCreated, list)
}

// List handles GET /api/v1/households/:id/lists
func (ctl *Controller) List(c *gin.Context) {
	householdID, ok := parseUintParam(c, "id")
	if !ok {
		return
	}

	lists, err := ctl.service.ListByHouseholdID(householdID)
	if err != nil {
		log.Printf("shopping list list: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
		return
	}

	c.JSON(http.StatusOK, lists)
}

// GetByID handles GET /api/v1/lists/:id
func (ctl *Controller) GetByID(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}

	list, err := ctl.service.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, list)
}

// Update handles PUT /api/v1/lists/:id
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

	list, err := ctl.service.Update(id, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, list)
}

// Delete handles DELETE /api/v1/lists/:id
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