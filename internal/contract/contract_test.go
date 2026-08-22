package contract

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
	"github.com/AliFnieer/needly-backend/internal/docs"
	"github.com/AliFnieer/needly-backend/internal/history"
	"github.com/AliFnieer/needly-backend/internal/household"
	"github.com/AliFnieer/needly-backend/internal/middleware"
	"github.com/AliFnieer/needly-backend/internal/shoppingitem"
	"github.com/AliFnieer/needly-backend/internal/shoppinglist"
	"github.com/AliFnieer/needly-backend/internal/testutil"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var openAPISpec map[string]interface{}

func init() {
	gin.SetMode(gin.TestMode)

	var spec map[string]interface{}
	if err := json.Unmarshal(docs.OpenAPISpec(), &spec); err != nil {
		panic("failed to parse openapi.json: " + err.Error())
	}
	openAPISpec = spec
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

	engine.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "database": "ok"})
	})
	engine.GET("/docs", gin.WrapH(docs.SwaggerUIHandler()))
	engine.GET("/docs/redoc", gin.WrapH(docs.RedocUIHandler()))
	engine.GET("/docs/openapi.json", gin.WrapH(http.HandlerFunc(docs.ServeOpenAPIHandler)))

	api := engine.Group("/api/v1")
	auth.RegisterRoutes(api, db, cfg, nil)
	household.RegisterRoutes(api, db, cfg, nil, nil)
	shoppinglist.RegisterRoutes(api, db, cfg, nil, nil)
	shoppingitem.RegisterRoutes(api, db, cfg, nil, nil)
	category.RegisterRoutes(api, db, cfg, nil)
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

func parseJSONArray(t *testing.T, w *httptest.ResponseRecorder) []interface{} {
	t.Helper()
	var result []interface{}
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

// --- Contract validation helpers ---

func specHasPath(path string) bool {
	paths, ok := openAPISpec["paths"].(map[string]interface{})
	if !ok {
		return false
	}
	_, ok = paths[path]
	return ok
}

func specOperationExists(path, method string) bool {
	paths, ok := openAPISpec["paths"].(map[string]interface{})
	if !ok {
		return false
	}
	pathItem, ok := paths[path].(map[string]interface{})
	if !ok {
		return false
	}
	_, ok = pathItem[method]
	return ok
}

func specHasResponseStatusCode(path, method string, statusCode int) bool {
	paths, ok := openAPISpec["paths"].(map[string]interface{})
	if !ok {
		return false
	}
	pathItem, ok := paths[path].(map[string]interface{})
	if !ok {
		return false
	}
	operation, ok := pathItem[method].(map[string]interface{})
	if !ok {
		return false
	}
	responses, ok := operation["responses"].(map[string]interface{})
	if !ok {
		return false
	}
	code := fmt.Sprintf("%d", statusCode)
	_, ok = responses[code]
	return ok
}

// --- OpenAPI Spec Contract Tests ---

func TestContract_OpenAPI_HasHealthEndpoint(t *testing.T) {
	if !specHasPath("/health") {
		t.Error("OpenAPI spec missing /health path")
	}
	if !specOperationExists("/health", "get") {
		t.Error("OpenAPI spec missing GET /health operation")
	}
}

func TestContract_OpenAPI_HasAuthEndpoints(t *testing.T) {
	if !specHasPath("/api/v1/auth/register") {
		t.Error("OpenAPI spec missing /api/v1/auth/register path")
	}
	if !specOperationExists("/api/v1/auth/register", "post") {
		t.Error("OpenAPI spec missing POST /api/v1/auth/register operation")
	}
	if !specHasPath("/api/v1/auth/login") {
		t.Error("OpenAPI spec missing /api/v1/auth/login path")
	}
	if !specOperationExists("/api/v1/auth/login", "post") {
		t.Error("OpenAPI spec missing POST /api/v1/auth/login operation")
	}
}

func TestContract_OpenAPI_HasHouseholdEndpoints(t *testing.T) {
	if !specHasPath("/api/v1/households") {
		t.Error("OpenAPI spec missing /api/v1/households path")
	}
	if !specOperationExists("/api/v1/households", "get") {
		t.Error("OpenAPI spec missing GET /api/v1/households operation")
	}
	if !specHasPath("/api/v1/households/{hid}/categories") {
		t.Error("OpenAPI spec missing /api/v1/households/{hid}/categories path")
	}
}

func TestContract_OpenAPI_HasPasswordResetAndVerificationEndpoints(t *testing.T) {
	for _, tc := range []struct {
		path, method string
	}{
		{"/api/v1/auth/forgot-password", "post"},
		{"/api/v1/auth/reset-password", "post"},
		{"/api/v1/auth/verify-email", "get"},
		{"/api/v1/auth/resend-verification", "post"},
	} {
		if !specHasPath(tc.path) {
			t.Errorf("OpenAPI spec missing %s path", tc.path)
			continue
		}
		if !specOperationExists(tc.path, tc.method) {
			t.Errorf("OpenAPI spec missing %s %s operation", tc.method, tc.path)
		}
	}
}

func TestContract_OpenAPI_StatusCodes(t *testing.T) {
	tests := []struct {
		path       string
		method     string
		statusCode int
		desc       string
	}{
		{"/health", "get", 200, "health check success"},
		{"/api/v1/auth/register", "post", 201, "register success"},
		{"/api/v1/auth/register", "post", 409, "register duplicate"},
		{"/api/v1/auth/login", "post", 200, "login success"},
		{"/api/v1/auth/login", "post", 401, "login bad credentials"},
		{"/api/v1/auth/forgot-password", "post", 200, "forgot password accepted"},
		{"/api/v1/auth/forgot-password", "post", 400, "forgot password invalid body"},
		{"/api/v1/auth/reset-password", "post", 200, "reset password success"},
		{"/api/v1/auth/reset-password", "post", 400, "reset password bad token"},
		{"/api/v1/auth/verify-email", "get", 200, "verify email success"},
		{"/api/v1/auth/verify-email", "get", 400, "verify email bad token"},
		{"/api/v1/households", "get", 200, "list households"},
		{"/api/v1/households/{hid}/categories", "get", 200, "list categories"},
		{"/api/v1/households/{hid}/categories", "post", 201, "create category"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if !specHasResponseStatusCode(tt.path, tt.method, tt.statusCode) {
				t.Errorf("OpenAPI spec missing %d response for %s %s", tt.statusCode, tt.method, tt.path)
			}
		})
	}
}

// --- Runtime Contract Tests ---

func TestContract_Health(t *testing.T) {
	s := setupServer(t)

	w := s.doRequest("GET", "/health", nil, nil)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /health: expected 200, got %d", w.Code)
	}

	resp := parseJSON(t, w)
	if _, ok := resp["status"]; !ok {
		t.Error("GET /health response missing 'status' field")
	}
}

func TestContract_Register_Success(t *testing.T) {
	s := setupServer(t)

	w := s.doRequest("POST", "/api/v1/auth/register", map[string]string{
		"first_name": "Contract",
		"last_name":  "Tester",
		"email":      "contract@test.com",
		"password":   "securepass123",
	}, nil)

	if w.Code != http.StatusCreated {
		t.Fatalf("POST /api/v1/auth/register: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseJSON(t, w)
	if resp["access_token"] == nil {
		t.Error("response missing 'access_token'")
	}
	if resp["refresh_token"] == nil {
		t.Error("response missing 'refresh_token'")
	}
}

func TestContract_Register_Duplicate(t *testing.T) {
	s := setupServer(t)

	s.doRequest("POST", "/api/v1/auth/register", map[string]string{
		"first_name": "Contract",
		"last_name":  "Tester",
		"email":      "dup-contract@test.com",
		"password":   "securepass123",
	}, nil)

	w := s.doRequest("POST", "/api/v1/auth/register", map[string]string{
		"first_name": "Contract",
		"last_name":  "Tester",
		"email":      "dup-contract@test.com",
		"password":   "securepass123",
	}, nil)

	if w.Code != http.StatusConflict {
		t.Fatalf("POST /api/v1/auth/register duplicate: expected 409, got %d: %s", w.Code, w.Body.String())
	}
}

func TestContract_Login_Success(t *testing.T) {
	s := setupServer(t)
	registerAndGetTokens(t, s, "login-contract@test.com")

	w := s.doRequest("POST", "/api/v1/auth/login", map[string]string{
		"email":    "login-contract@test.com",
		"password": "securepass123",
	}, nil)

	if w.Code != http.StatusOK {
		t.Fatalf("POST /api/v1/auth/login: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseJSON(t, w)
	if resp["access_token"] == nil {
		t.Error("response missing 'access_token'")
	}
	if resp["refresh_token"] == nil {
		t.Error("response missing 'refresh_token'")
	}
}

func TestContract_Login_BadCredentials(t *testing.T) {
	s := setupServer(t)
	registerAndGetTokens(t, s, "badcred-contract@test.com")

	w := s.doRequest("POST", "/api/v1/auth/login", map[string]string{
		"email":    "badcred-contract@test.com",
		"password": "wrongpassword",
	}, nil)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("POST /api/v1/auth/login bad creds: expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestContract_Households_List(t *testing.T) {
	s := setupServer(t)
	token, _ := registerAndGetTokens(t, s, "hh-contract@test.com")

	w := s.doRequest("GET", "/api/v1/households", nil, authHeaders(token))

	if w.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/households: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseJSONArray(t, w)
	if resp == nil {
		t.Fatal("GET /api/v1/households: response is not an array")
	}
}

func TestContract_HouseholdCategories_List(t *testing.T) {
	s := setupServer(t)
	token, _ := registerAndGetTokens(t, s, "cat-contract@test.com")

	w := s.doRequest("POST", "/api/v1/households", map[string]string{"name": "Cat Contract HH"}, authHeaders(token))
	if w.Code != http.StatusCreated {
		t.Fatalf("create household failed: %d", w.Code)
	}
	hh := parseJSON(t, w)
	hhID := fmt.Sprintf("%.0f", hh["id"].(float64))

	w2 := s.doRequest("GET", fmt.Sprintf("/api/v1/households/%s/categories", hhID), nil, authHeaders(token))

	if w2.Code != http.StatusOK {
		t.Fatalf("GET categories: expected 200, got %d: %s", w2.Code, w2.Body.String())
	}

	resp := parseJSONArray(t, w2)
	if resp == nil {
		t.Fatal("GET categories: response is not an array")
	}
}

func TestContract_HouseholdCategories_Create(t *testing.T) {
	s := setupServer(t)
	token, _ := registerAndGetTokens(t, s, "catcreate-contract@test.com")

	w := s.doRequest("POST", "/api/v1/households", map[string]string{"name": "CatCreate Contract HH"}, authHeaders(token))
	if w.Code != http.StatusCreated {
		t.Fatalf("create household failed: %d", w.Code)
	}
	hh := parseJSON(t, w)
	hhID := fmt.Sprintf("%.0f", hh["id"].(float64))

	w2 := s.doRequest("POST", fmt.Sprintf("/api/v1/households/%s/categories", hhID), map[string]string{"name": "Dairy"}, authHeaders(token))

	if w2.Code != http.StatusCreated {
		t.Fatalf("POST category: expected 201, got %d: %s", w2.Code, w2.Body.String())
	}

	resp := parseJSON(t, w2)
	if resp["id"] == nil {
		t.Error("response missing 'id'")
	}
	if resp["name"] != "Dairy" {
		t.Errorf("expected name 'Dairy', got %v", resp["name"])
	}
}

func TestContract_OpenAPI_IdempotencyKeyDocumented(t *testing.T) {
	protected := []string{
		"/api/v1/households",
		"/api/v1/households/{id}/members",
		"/api/v1/households/{id}/lists",
		"/api/v1/lists/{id}/items",
		"/api/v1/households/{hid}/categories",
		"/api/v1/history/{id}/re-add",
	}
	for _, p := range protected {
		pathItem, _ := openAPISpec["paths"].(map[string]interface{})[p].(map[string]interface{})
		if pathItem == nil {
			t.Fatalf("OpenAPI spec missing path %s", p)
		}
		op, _ := pathItem["post"].(map[string]interface{})
		if op == nil {
			t.Fatalf("OpenAPI spec missing POST %s", p)
		}
		params, _ := op["parameters"].([]interface{})
		found := false
		for _, raw := range params {
			param, _ := raw.(map[string]interface{})
			if ref, _ := param["$ref"].(string); ref == "#/components/parameters/IdempotencyKeyHeader" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("POST %s missing IdempotencyKeyHeader parameter reference", p)
		}
	}

	comps, _ := openAPISpec["components"].(map[string]interface{})
	parameters, _ := comps["parameters"].(map[string]interface{})
	hdr, _ := parameters["IdempotencyKeyHeader"].(map[string]interface{})
	if hdr == nil {
		t.Fatal("OpenAPI spec missing components.parameters.IdempotencyKeyHeader")
	}
	if name, _ := hdr["name"].(string); name != "Idempotency-Key" {
		t.Errorf("header parameter name = %q, want Idempotency-Key", name)
	}
}
