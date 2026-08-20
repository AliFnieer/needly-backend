package category

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Controller handles category HTTP requests.
type Controller struct {
	service *Service
}

// NewController creates a new category controller.
func NewController(service *Service) *Controller {
	return &Controller{
		service: service,
	}
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

// extractHouseholdID extracts the household ID from the URL path.
func extractHouseholdID(c *gin.Context) (uint, bool) {
	return parseIDParam(c, "id")
}

// Create handles POST /api/v1/households/:id/categories
func (ctl *Controller) Create(c *gin.Context) {
	householdID, ok := extractHouseholdID(c)
	if !ok {
		return
	}

	var req CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	category, err := ctl.service.Create(householdID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, category)
}

// GetByID handles GET /api/v1/households/:hid/categories/:id
func (ctl *Controller) GetByID(c *gin.Context) {
	householdID, ok := extractHouseholdID(c)
	if !ok {
		return
	}

	id, err := strconv.ParseUint(c.Param("categoryId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid category id"})
		return
	}

	category, err := ctl.service.GetByID(uint(id), householdID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, category)
}

// List handles GET /api/v1/households/:id/categories
func (ctl *Controller) List(c *gin.Context) {
	householdID, ok := extractHouseholdID(c)
	if !ok {
		return
	}

	categories, err := ctl.service.List(householdID)
	if err != nil {
		slog.Error("category list failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, categories)
}

// Update handles PUT /api/v1/households/:hid/categories/:categoryId
func (ctl *Controller) Update(c *gin.Context) {
	householdID, ok := extractHouseholdID(c)
	if !ok {
		return
	}

	catID, err := strconv.ParseUint(c.Param("categoryId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid category id"})
		return
	}

	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	category, err := ctl.service.Update(uint(catID), householdID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, category)
}

// Delete handles DELETE /api/v1/households/:hid/categories/:categoryId
func (ctl *Controller) Delete(c *gin.Context) {
	householdID, ok := extractHouseholdID(c)
	if !ok {
		return
	}

	catID, err := strconv.ParseUint(c.Param("categoryId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid category id"})
		return
	}

	if err := ctl.service.Delete(uint(catID), householdID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
