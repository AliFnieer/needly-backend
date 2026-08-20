package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AliFnieer/needly-backend/internal/auth"
	"github.com/AliFnieer/needly-backend/internal/category"
	"github.com/AliFnieer/needly-backend/internal/config"
	"github.com/AliFnieer/needly-backend/internal/history"
	"github.com/AliFnieer/needly-backend/internal/household"
	"github.com/AliFnieer/needly-backend/internal/middleware"
	"github.com/AliFnieer/needly-backend/internal/shoppingitem"
	"github.com/AliFnieer/needly-backend/internal/shoppinglist"
	"github.com/AliFnieer/needly-backend/internal/testutil"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type testServer struct {
	engine *gin.Engine
	db     *gorm.DB
	cfg    *config.Config
}

func setupServer(t *testing.T) *testServer {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestConfig()

	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(middleware.APIVersionMiddleware())

	api := engine.Group("/api/v1")
	auth.RegisterRoutes(api, db, cfg, nil)
	household.RegisterRoutes(api, db, cfg, nil)
	shoppinglist.RegisterRoutes(api, db, cfg, nil, nil)
	shoppingitem.RegisterRoutes(api, db, cfg, nil, nil)
	category.RegisterRoutes(api, db, cfg)
	history.RegisterRoutes(api, db, cfg)

	return &testServer{engine: engine, db: db, cfg: cfg}
}

func (s *testServer) doRequest(method, path string, body interface{}, headers map[string]string) *httptest.ResponseRecorder {
	var reqBody *bytes.Buffer
	if body != nil {
		data, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(data)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}

	req, _ := http.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	w := httptest.NewRecorder()
	s.engine.ServeHTTP(w, req)
	return w
}

func parseJSON(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse response JSON: %v\nbody: %s", err, w.Body.String())
	}
	return result
}

func parseJSONArray(t *testing.T, w *httptest.ResponseRecorder) []map[string]interface{} {
	t.Helper()
	var result []map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse response JSON array: %v\nbody: %s", err, w.Body.String())
	}
	return result
}

func registerAndGetTokens(t *testing.T, s *testServer, email string) (accessToken, refreshToken string) {
	t.Helper()
	w := s.doRequest("POST", "/api/v1/auth/register", map[string]string{
		"first_name": "Test",
		"last_name":  "User",
		"email":      email,
		"password":   "securepass123",
	}, nil)

	if w.Code != http.StatusCreated {
		t.Fatalf("register failed: %d %s", w.Code, w.Body.String())
	}

	resp := parseJSON(t, w)
	return resp["access_token"].(string), resp["refresh_token"].(string)
}

func authHeaders(token string) map[string]string {
	return map[string]string{"Authorization": "Bearer " + token}
}

// --- AUTH TESTS ---

func TestAuth_Register(t *testing.T) {
	s := setupServer(t)

	w := s.doRequest("POST", "/api/v1/auth/register", map[string]string{
		"first_name": "Ali",
		"last_name":  "Fnier",
		"email":      "ali@test.com",
		"password":   "securepass123",
	}, nil)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseJSON(t, w)
	if resp["access_token"] == nil || resp["refresh_token"] == nil {
		t.Error("expected tokens in response")
	}
}

func TestAuth_Register_DuplicateEmail(t *testing.T) {
	s := setupServer(t)

	s.doRequest("POST", "/api/v1/auth/register", map[string]string{
		"first_name": "Ali", "last_name": "Fnier",
		"email": "dup@test.com", "password": "securepass123",
	}, nil)

	w := s.doRequest("POST", "/api/v1/auth/register", map[string]string{
		"first_name": "Ali", "last_name": "Fnier",
		"email": "dup@test.com", "password": "securepass123",
	}, nil)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuth_Login(t *testing.T) {
	s := setupServer(t)

	registerAndGetTokens(t, s, "login@test.com")

	w := s.doRequest("POST", "/api/v1/auth/login", map[string]string{
		"email":    "login@test.com",
		"password": "securepass123",
	}, nil)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuth_Login_WrongPassword(t *testing.T) {
	s := setupServer(t)

	registerAndGetTokens(t, s, "wrong@test.com")

	w := s.doRequest("POST", "/api/v1/auth/login", map[string]string{
		"email":    "wrong@test.com",
		"password": "badpassword",
	}, nil)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuth_RefreshToken(t *testing.T) {
	s := setupServer(t)

	_, refreshToken := registerAndGetTokens(t, s, "refresh@test.com")

	w := s.doRequest("POST", "/api/v1/auth/refresh", map[string]string{
		"refresh_token": refreshToken,
	}, nil)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseJSON(t, w)
	if resp["refresh_token"] == refreshToken {
		t.Error("expected new refresh token (rotation)")
	}
}

func TestAuth_Me(t *testing.T) {
	s := setupServer(t)

	accessToken, _ := registerAndGetTokens(t, s, "me@test.com")

	w := s.doRequest("GET", "/api/v1/auth/me", nil, authHeaders(accessToken))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseJSON(t, w)
	if resp["email"] != "me@test.com" {
		t.Errorf("expected email me@test.com, got %s", resp["email"])
	}
}

func TestAuth_Me_Unauthorized(t *testing.T) {
	s := setupServer(t)

	w := s.doRequest("GET", "/api/v1/auth/me", nil, nil)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAuth_Logout(t *testing.T) {
	s := setupServer(t)

	accessToken, refreshToken := registerAndGetTokens(t, s, "logout@test.com")

	// Logout with specific token
	w := s.doRequest("POST", "/api/v1/auth/logout", map[string]interface{}{
		"refresh_token": refreshToken,
	}, authHeaders(accessToken))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Refresh should fail after logout
	w2 := s.doRequest("POST", "/api/v1/auth/refresh", map[string]interface{}{
		"refresh_token": refreshToken,
	}, nil)

	if w2.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 after logout refresh, got %d", w2.Code)
	}
}

// --- API VERSION HEADER ---

func TestAPI_VersionHeader(t *testing.T) {
	s := setupServer(t)

	w := s.doRequest("GET", "/api/v1/auth/me", nil, nil)
	if w.Header().Get("API-Version") != middleware.CurrentAPIVersion {
		t.Errorf("expected API-Version: %s, got %s", middleware.CurrentAPIVersion, w.Header().Get("API-Version"))
	}
}

// --- HOUSEHOLD TESTS ---

func TestHousehold_Create(t *testing.T) {
	s := setupServer(t)
	token, _ := registerAndGetTokens(t, s, "hh@test.com")

	w := s.doRequest("POST", "/api/v1/households", map[string]string{
		"name": "My Household",
	}, authHeaders(token))

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseJSON(t, w)
	if resp["name"] != "My Household" {
		t.Errorf("expected name 'My Household', got %s", resp["name"])
	}
}

func TestHousehold_List(t *testing.T) {
	s := setupServer(t)
	token, _ := registerAndGetTokens(t, s, "hhlist@test.com")

	s.doRequest("POST", "/api/v1/households", map[string]string{"name": "HH1"}, authHeaders(token))
	s.doRequest("POST", "/api/v1/households", map[string]string{"name": "HH2"}, authHeaders(token))

	w := s.doRequest("GET", "/api/v1/households", nil, authHeaders(token))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	arr := parseJSONArray(t, w)
	if len(arr) != 2 {
		t.Fatalf("expected 2 households, got %d", len(arr))
	}
}

func TestHousehold_Update(t *testing.T) {
	s := setupServer(t)
	token, _ := registerAndGetTokens(t, s, "hhupd@test.com")

	w := s.doRequest("POST", "/api/v1/households", map[string]string{"name": "Old Name"}, authHeaders(token))
	hh := parseJSON(t, w)
	id := fmt.Sprintf("%.0f", hh["id"].(float64))

	w2 := s.doRequest("PUT", "/api/v1/households/"+id, map[string]string{"name": "New Name"}, authHeaders(token))

	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w2.Code, w2.Body.String())
	}
}

func TestHousehold_Delete(t *testing.T) {
	s := setupServer(t)
	token, _ := registerAndGetTokens(t, s, "hhdel@test.com")

	w := s.doRequest("POST", "/api/v1/households", map[string]string{"name": "To Delete"}, authHeaders(token))
	hh := parseJSON(t, w)
	id := fmt.Sprintf("%.0f", hh["id"].(float64))

	w2 := s.doRequest("DELETE", "/api/v1/households/"+id, nil, authHeaders(token))

	if w2.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w2.Code)
	}
}

func TestHousehold_Unauthorized(t *testing.T) {
	s := setupServer(t)

	w := s.doRequest("GET", "/api/v1/households", nil, nil)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

// --- CATEGORY TESTS ---

func TestCategory_CRUD(t *testing.T) {
	s := setupServer(t)
	token, _ := registerAndGetTokens(t, s, "cat@test.com")

	// Create a household first (categories are household-scoped)
	w := s.doRequest("POST", "/api/v1/households", map[string]string{"name": "Cat HH"}, authHeaders(token))
	if w.Code != http.StatusCreated {
		t.Fatalf("create household: %d", w.Code)
	}
	hh := parseJSON(t, w)
	hhID := fmt.Sprintf("%.0f", hh["id"].(float64))
	catBase := fmt.Sprintf("/api/v1/households/%s/categories", hhID)

	// Create
	w = s.doRequest("POST", catBase, map[string]string{"name": "Fruits"}, authHeaders(token))
	if w.Code != http.StatusCreated {
		t.Fatalf("create failed: %d %s", w.Code, w.Body.String())
	}
	cat := parseJSON(t, w)
	id := fmt.Sprintf("%.0f", cat["id"].(float64))

	// List
	w2 := s.doRequest("GET", catBase, nil, authHeaders(token))
	if w2.Code != http.StatusOK {
		t.Fatalf("list failed: %d", w2.Code)
	}

	// Get
	w3 := s.doRequest("GET", catBase+"/"+id, nil, authHeaders(token))
	if w3.Code != http.StatusOK {
		t.Fatalf("get failed: %d", w3.Code)
	}

	// Update
	w4 := s.doRequest("PUT", catBase+"/"+id, map[string]string{"name": "Vegetables"}, authHeaders(token))
	if w4.Code != http.StatusOK {
		t.Fatalf("update failed: %d", w4.Code)
	}

	// Delete
	w5 := s.doRequest("DELETE", catBase+"/"+id, nil, authHeaders(token))
	if w5.Code != http.StatusNoContent {
		t.Fatalf("delete failed: %d", w5.Code)
	}
}

// --- SHOPPING LIST TESTS ---

func setupHouseholdAndList(t *testing.T, s *testServer, email string) (token string, householdID, listID string) {
	t.Helper()
	token, _ = registerAndGetTokens(t, s, email)

	w := s.doRequest("POST", "/api/v1/households", map[string]string{"name": "Test HH"}, authHeaders(token))
	hh := parseJSON(t, w)
	householdID = fmt.Sprintf("%.0f", hh["id"].(float64))

	w2 := s.doRequest("POST", fmt.Sprintf("/api/v1/households/%s/lists", householdID), map[string]string{"name": "Groceries"}, authHeaders(token))
	list := parseJSON(t, w2)
	listID = fmt.Sprintf("%.0f", list["id"].(float64))
	return
}

func TestShoppingList_Create(t *testing.T) {
	s := setupServer(t)
	token, hhID, _ := setupHouseholdAndList(t, s, "sl@test.com")

	w := s.doRequest("POST", fmt.Sprintf("/api/v1/households/%s/lists", hhID), map[string]string{"name": "Weekly"}, authHeaders(token))

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestShoppingList_GetByID(t *testing.T) {
	s := setupServer(t)
	token, _, listID := setupHouseholdAndList(t, s, "slget@test.com")

	w := s.doRequest("GET", "/api/v1/lists/"+listID, nil, authHeaders(token))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestShoppingList_Delete(t *testing.T) {
	s := setupServer(t)
	token, _, listID := setupHouseholdAndList(t, s, "sldel@test.com")

	w := s.doRequest("DELETE", "/api/v1/lists/"+listID, nil, authHeaders(token))

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}
}

// --- SHOPPING ITEM TESTS ---

func TestShoppingItem_CRUD(t *testing.T) {
	s := setupServer(t)
	token, _, listID := setupHouseholdAndList(t, s, "si@test.com")

	// Create
	w := s.doRequest("POST", fmt.Sprintf("/api/v1/lists/%s/items", listID), map[string]interface{}{
		"name":     "Milk",
		"quantity": 2,
		"unit":     "liters",
	}, authHeaders(token))
	if w.Code != http.StatusCreated {
		t.Fatalf("create failed: %d %s", w.Code, w.Body.String())
	}
	item := parseJSON(t, w)
	itemID := fmt.Sprintf("%.0f", item["id"].(float64))

	// List
	w2 := s.doRequest("GET", fmt.Sprintf("/api/v1/lists/%s/items", listID), nil, authHeaders(token))
	if w2.Code != http.StatusOK {
		t.Fatalf("list failed: %d", w2.Code)
	}
	items := parseJSONArray(t, w2)
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}

	// Get
	w3 := s.doRequest("GET", "/api/v1/items/"+itemID, nil, authHeaders(token))
	if w3.Code != http.StatusOK {
		t.Fatalf("get failed: %d", w3.Code)
	}

	// Update
	w4 := s.doRequest("PUT", "/api/v1/items/"+itemID, map[string]interface{}{
		"name": "Whole Milk",
	}, authHeaders(token))
	if w4.Code != http.StatusOK {
		t.Fatalf("update failed: %d", w4.Code)
	}

	// Complete
	w5 := s.doRequest("PATCH", "/api/v1/items/"+itemID+"/completed", map[string]bool{
		"is_completed": true,
	}, authHeaders(token))
	if w5.Code != http.StatusOK {
		t.Fatalf("complete failed: %d %s", w5.Code, w5.Body.String())
	}

	// History should exist now
	w6 := s.doRequest("GET", fmt.Sprintf("/api/v1/lists/%s/history", listID), nil, authHeaders(token))
	if w6.Code != http.StatusOK {
		t.Fatalf("history failed: %d", w6.Code)
	}

	// Delete
	w7 := s.doRequest("DELETE", "/api/v1/items/"+itemID, nil, authHeaders(token))
	if w7.Code != http.StatusNoContent {
		t.Fatalf("delete failed: %d", w7.Code)
	}
}

// --- FULL LIFECYCLE TEST ---

func TestFullLifecycle(t *testing.T) {
	s := setupServer(t)
	token, _ := registerAndGetTokens(t, s, "lifecycle@test.com")

	// 1. Create household
	w := s.doRequest("POST", "/api/v1/households", map[string]string{"name": "Family"}, authHeaders(token))
	if w.Code != http.StatusCreated {
		t.Fatalf("create household: %d", w.Code)
	}
	hh := parseJSON(t, w)
	hhID := fmt.Sprintf("%.0f", hh["id"].(float64))

	// 2. Create category (scoped to household)
	w = s.doRequest("POST", fmt.Sprintf("/api/v1/households/%s/categories", hhID), map[string]string{"name": "Dairy"}, authHeaders(token))
	if w.Code != http.StatusCreated {
		t.Fatalf("create category: %d", w.Code)
	}
	cat := parseJSON(t, w)
	catID := fmt.Sprintf("%.0f", cat["id"].(float64))

	// 3. Create list
	w = s.doRequest("POST", fmt.Sprintf("/api/v1/households/%s/lists", hhID), map[string]string{"name": "Weekly"}, authHeaders(token))
	if w.Code != http.StatusCreated {
		t.Fatalf("create list: %d", w.Code)
	}
	list := parseJSON(t, w)
	listID := fmt.Sprintf("%.0f", list["id"].(float64))

	// 4. Add items
	for _, name := range []string{"Milk", "Eggs", "Bread"} {
		catUint := uint(0)
		fmt.Sscanf(catID, "%d", &catUint)
		w = s.doRequest("POST", fmt.Sprintf("/api/v1/lists/%s/items", listID), map[string]interface{}{
			"name":        name,
			"quantity":    1,
			"category_id": catUint,
		}, authHeaders(token))
		if w.Code != http.StatusCreated {
			t.Fatalf("create item %s: %d %s", name, w.Code, w.Body.String())
		}
	}

	// 5. List items
	w = s.doRequest("GET", fmt.Sprintf("/api/v1/lists/%s/items", listID), nil, authHeaders(token))
	items := parseJSONArray(t, w)
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}

	// 6. Complete first item
	milkID := fmt.Sprintf("%.0f", items[0]["id"].(float64))
	w = s.doRequest("PATCH", "/api/v1/items/"+milkID+"/completed", map[string]bool{"is_completed": true}, authHeaders(token))
	if w.Code != http.StatusOK {
		t.Fatalf("complete item: %d", w.Code)
	}

	// 7. Check history
	w = s.doRequest("GET", fmt.Sprintf("/api/v1/lists/%s/history", listID), nil, authHeaders(token))
	historyResp := parseJSONArray(t, w)
	if len(historyResp) != 1 {
		t.Fatalf("expected 1 history entry, got %d", len(historyResp))
	}

	// 8. Re-add from history
	histID := fmt.Sprintf("%.0f", historyResp[0]["id"].(float64))
	w = s.doRequest("POST", "/api/v1/history/"+histID+"/re-add", nil, authHeaders(token))
	if w.Code != http.StatusCreated {
		t.Fatalf("re-add: %d %s", w.Code, w.Body.String())
	}

	// 9. Verify household list
	w = s.doRequest("GET", "/api/v1/households", nil, authHeaders(token))
	hhs := parseJSONArray(t, w)
	if len(hhs) != 1 {
		t.Fatalf("expected 1 household, got %d", len(hhs))
	}

	// 10. Category list (scoped to household)
	w = s.doRequest("GET", fmt.Sprintf("/api/v1/households/%s/categories", hhID), nil, authHeaders(token))
	cats := parseJSONArray(t, w)
	if len(cats) != 1 {
		t.Fatalf("expected 1 category, got %d", len(cats))
	}
}
