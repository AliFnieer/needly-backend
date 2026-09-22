package shoppingitem_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/AliFnieer/needly-backend/internal/history"
	"github.com/AliFnieer/needly-backend/internal/shoppingitem"
	"github.com/AliFnieer/needly-backend/internal/testutil"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestController_UpdateClearsRecurrenceRule(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := testutil.SetupTestDB(t)
	user := testutil.SeedUser(t, db, "bind_clear@example.com", "password123")
	listID := seedList(t, db, user.ID)

	svc := shoppingitem.NewService(db, nil, history.NewService(db), nil)
	ctl := shoppingitem.NewController(svc)

	created, err := svc.Create(t.Context(), listID, user.ID, &shoppingitem.CreateRequest{Name: "Bread"})
	require.NoError(t, err)

	router := gin.New()
	router.PUT("/items/:id", func(c *gin.Context) {
		c.Set("user_id", float64(user.ID))
		ctl.Update(c)
	})

	perform := func(body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(
			http.MethodPut,
			"/items/"+strconv.FormatUint(uint64(created.ID), 10),
			strings.NewReader(body),
		)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w
	}

	monthly := "monthly"
	_, err = svc.Update(t.Context(), created.ID, user.ID, &shoppingitem.UpdateRequest{RecurrenceRule: &monthly})
	require.NoError(t, err)

	w := perform(`{"recurrence_rule":""}`)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 when clearing recurrence, got %d (%s)", w.Code, w.Body.String())
	}
	var item shoppingitem.ShoppingItem
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &item))
	assert.Empty(t, item.RecurrenceRule)
	assert.Nil(t, item.NextDueAt)

	w = perform(`{"recurrence_rule":"hourly"}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid recurrence rule, got %d (%s)", w.Code, w.Body.String())
	}
}