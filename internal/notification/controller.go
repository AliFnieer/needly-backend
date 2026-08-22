package notification

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Controller handles notification HTTP requests.
type Controller struct {
	service *Service
}

// NewController creates a new notification controller.
func NewController(service *Service) *Controller {
	return &Controller{
		service: service,
	}
}

// ListHouseholdHistory handles GET /api/v1/households/:id/notifications
func (ctl *Controller) ListHouseholdHistory(c *gin.Context) {
	householdID, ok := parseUintParam(c, "id")
	if !ok {
		return
	}

	items, err := ctl.service.HistoryByHousehold(context.Background(), householdID)
	if err != nil {
		slog.Error("notification list failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
		return
	}

	if items == nil {
		items = []*Notification{}
	}

	c.JSON(http.StatusOK, items)
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
