package history

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Controller handles shopping history HTTP requests.
type Controller struct {
	service *Service
}

// NewController creates a new shopping history controller.
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

// ListByListID handles GET /api/v1/lists/:id/history
func (ctl *Controller) ListByListID(c *gin.Context) {
	listID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	limit, err := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if err != nil || limit < 0 {
		limit = 50
	}
	offset, err := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if err != nil || offset < 0 {
		offset = 0
	}

	entries, err := ctl.service.ListByListID(listID, limit, offset)
	if err != nil {
		slog.Error("history list by list failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
		return
	}

	c.JSON(http.StatusOK, entries)
}

// ListByHouseholdID handles GET /api/v1/households/:id/history
func (ctl *Controller) ListByHouseholdID(c *gin.Context) {
	householdID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	limit, err := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if err != nil || limit < 0 {
		limit = 50
	}
	offset, err := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if err != nil || offset < 0 {
		offset = 0
	}

	entries, err := ctl.service.ListByHouseholdID(householdID, limit, offset)
	if err != nil {
		slog.Error("history list by household failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
		return
	}

	c.JSON(http.StatusOK, entries)
}

// GetByID handles GET /api/v1/history/:id
func (ctl *Controller) GetByID(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	entry, err := ctl.service.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, entry)
}

// Delete handles DELETE /api/v1/history/:id
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