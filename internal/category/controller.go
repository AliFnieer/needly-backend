package category

import (
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

// Create handles POST /api/v1/categories
func (ctl *Controller) Create(c *gin.Context) {
	var req CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	category, err := ctl.service.Create(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, category)
}

// GetByID handles GET /api/v1/categories/:id
func (ctl *Controller) GetByID(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	category, err := ctl.service.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, category)
}

// List handles GET /api/v1/categories
func (ctl *Controller) List(c *gin.Context) {
	categories, err := ctl.service.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, categories)
}

// Update handles PUT /api/v1/categories/:id
func (ctl *Controller) Update(c *gin.Context) {
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

	category, err := ctl.service.Update(id, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, category)
}

// Delete handles DELETE /api/v1/categories/:id
func (ctl *Controller) Delete(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
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