package sync

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// Controller handles sync HTTP requests.
type Controller struct {
	service *Service
}

// NewController creates a new sync controller.
func NewController(service *Service) *Controller {
	return &Controller{
		service: service,
	}
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

// Sync handles GET /api/v1/households/:id/sync?since=RFC3339
func (ctl *Controller) Sync(c *gin.Context) {
	householdID, ok := parseUintParam(c, "id")
	if !ok {
		return
	}

	var since *time.Time
	if raw := c.Query("since"); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid since parameter (use RFC3339 timestamp)",
			})
			return
		}
		since = &parsed
	}

	snapshot, err := ctl.service.Snapshot(c.Request.Context(), householdID, since)
	if err != nil {
		slog.Error("household sync failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
		return
	}

	c.JSON(http.StatusOK, snapshot)
}
