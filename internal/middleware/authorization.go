package middleware

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// AuthzMiddleware creates an authorization middleware that verifies the
// authenticated user is a member of the target household.
// resolveHousehold extracts the household ID from the request context.
type resolveHousehold func(c *gin.Context, db *gorm.DB) (uint, bool)

// RequireMembership returns a middleware that enforces household membership.
// It resolves the household ID via the provided function and checks the
// household_members table.
func RequireMembership(db *gorm.DB, resolve resolveHousehold) gin.HandlerFunc {
	return func(c *gin.Context) {
		uid, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
			c.Abort()
			return
		}

		userID, ok := uid.(float64)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user id"})
			c.Abort()
			return
		}

		householdID, ok := resolve(c, db)
		if !ok {
			return // response already written by resolve
		}

		var count int64
		if err := db.Table("household_members").
			Where("household_id = ? AND user_id = ?", householdID, uint(userID)).
			Count(&count).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			c.Abort()
			return
		}

		if count == 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": "you are not a member of this household"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// HouseholdFromParam extracts the household ID directly from a URL parameter.
func HouseholdFromParam(param string) resolveHousehold {
	return func(c *gin.Context, _ *gorm.DB) (uint, bool) {
		id, err := strconv.ParseUint(c.Param(param), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid household id"})
			c.Abort()
			return 0, false
		}
		return uint(id), true
	}
}

// HouseholdFromList resolves the household ID via the shopping_lists table.
func HouseholdFromList(db *gorm.DB) resolveHousehold {
	// Pre-prepare the query to avoid repeated DB reference issues.
	return func(c *gin.Context, _ *gorm.DB) (uint, bool) {
		id, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			c.Abort()
			return 0, false
		}

		var result struct {
			HouseholdID uint
		}
		if err := db.Table("shopping_lists").
			Select("household_id").
			Where("id = ?", id).
			Scan(&result).Error; err != nil || result.HouseholdID == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "list not found"})
			c.Abort()
			return 0, false
		}
		return result.HouseholdID, true
	}
}

// HouseholdFromItem resolves the household ID via shopping_items → shopping_lists.
func HouseholdFromItem(db *gorm.DB) resolveHousehold {
	return func(c *gin.Context, _ *gorm.DB) (uint, bool) {
		id, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			c.Abort()
			return 0, false
		}

		var result struct {
			HouseholdID uint
		}
		if err := db.Table("shopping_items").
			Joins("JOIN shopping_lists ON shopping_lists.id = shopping_items.list_id").
			Select("shopping_lists.household_id").
			Where("shopping_items.id = ?", id).
			Scan(&result).Error; err != nil || result.HouseholdID == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "item not found"})
			c.Abort()
			return 0, false
		}
		return result.HouseholdID, true
	}
}

// HouseholdFromHistory resolves the household ID via shopping_history → shopping_lists.
func HouseholdFromHistory(db *gorm.DB) resolveHousehold {
	return func(c *gin.Context, _ *gorm.DB) (uint, bool) {
		id, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			c.Abort()
			return 0, false
		}

		var result struct {
			HouseholdID uint
		}
		if err := db.Table("shopping_history").
			Joins("JOIN shopping_lists ON shopping_lists.id = shopping_history.list_id").
			Select("shopping_lists.household_id").
			Where("shopping_history.id = ?", id).
			Scan(&result).Error; err != nil || result.HouseholdID == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "history entry not found"})
			c.Abort()
			return 0, false
		}
		return result.HouseholdID, true
	}
}
