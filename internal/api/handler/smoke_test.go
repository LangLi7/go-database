package handler_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"go-database/internal/api/response"
	"go-database/internal/api/router"
	"go-database/internal/auth"
	"go-database/internal/connection"
	"go-database/internal/internaldb"

	_ "go-database/plugins/sqlite"
)

var (
	testStore  *internaldb.Store
	testMgr    *connection.Manager
	testJWT    *auth.JWTService
	testEngine *gin.Engine
	testToken  string
)

func setupTestEnv(t *testing.T) {
	if testEngine != nil {
		return
	}

	gin.SetMode(gin.TestMode)

	var err error
	testStore, err = internaldb.Open(context.Background(), ":memory:")
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	testMgr = connection.NewManager()
	testJWT = auth.NewJWTService("test-secret", 60)

	testEngine = gin.New()
	testEngine.Use(gin.Recovery())
	router.SetupRoutes(testEngine, testStore, testMgr, testJWT)

	// Login as admin to get token
	token, err := loginAsAdmin()
	if err != nil {
		t.Fatalf("failed to login as admin: %v", err)
	}
	testToken = token
}

func loginAsAdmin() (string, error) {
	body := `{"username":"admin","password":"admin"}`
	req := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	testEngine.ServeHTTP(w, req)

	if w.Code != 200 {
		return "", fmt.Errorf("login returned %d", w.Code)
	}

	var res response.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		return "", err
	}
	data := res.Data.(map[string]any)
	token := data["token"].(string)
	return token, nil
}

func authRequest(method, path, body string) *http.Request {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+testToken)
	return req
}

func checkStatus(t *testing.T, w *httptest.ResponseRecorder, expected int) {
	if w.Code != expected {
		t.Errorf("expected status %d, got %d. Body: %s", expected, w.Code, w.Body.String())
	}
}

// ---- TESTS ----

func TestHealth(t *testing.T) {
	setupTestEnv(t)
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	testEngine.ServeHTTP(w, req)
	checkStatus(t, w, 200)
}

func TestLogin(t *testing.T) {
	setupTestEnv(t)
	body := `{"username":"admin","password":"admin"}`
	req := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	testEngine.ServeHTTP(w, req)
	checkStatus(t, w, 200)

	var res response.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatal(err)
	}
	if !res.Success {
		t.Fatal("login failed")
	}
}

func TestLoginInvalid(t *testing.T) {
	setupTestEnv(t)
	body := `{"username":"admin","password":"wrong"}`
	req := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	testEngine.ServeHTTP(w, req)
	checkStatus(t, w, 401)
}

func TestAuthRequired(t *testing.T) {
	setupTestEnv(t)
	req := httptest.NewRequest("GET", "/api/v1/connections", nil)
	w := httptest.NewRecorder()
	testEngine.ServeHTTP(w, req)
	checkStatus(t, w, 401)
}

func TestListConnections(t *testing.T) {
	setupTestEnv(t)
	req := authRequest("GET", "/api/v1/connections", "")
	w := httptest.NewRecorder()
	testEngine.ServeHTTP(w, req)
	checkStatus(t, w, 200)
}

func TestCreateConnection(t *testing.T) {
	setupTestEnv(t)
	body := `{"name":"test-sqlite","type":"sqlite","source":"file","filepath":":memory:","tags":["test"]}`
	req := authRequest("POST", "/api/v1/connections", body)
	w := httptest.NewRecorder()
	testEngine.ServeHTTP(w, req)
	checkStatus(t, w, 201)
}

func TestAdminDesign(t *testing.T) {
	setupTestEnv(t)
	req := authRequest("GET", "/api/v1/admin/design", "")
	w := httptest.NewRecorder()
	testEngine.ServeHTTP(w, req)
	checkStatus(t, w, 200)
}

func TestAdminStats(t *testing.T) {
	setupTestEnv(t)
	req := authRequest("GET", "/api/v1/admin/stats", "")
	w := httptest.NewRecorder()
	testEngine.ServeHTTP(w, req)
	checkStatus(t, w, 200)
}

func TestListUsers(t *testing.T) {
	setupTestEnv(t)
	req := authRequest("GET", "/api/v1/admin/users", "")
	w := httptest.NewRecorder()
	testEngine.ServeHTTP(w, req)
	checkStatus(t, w, 200)
}

func TestCreateUser(t *testing.T) {
	setupTestEnv(t)
	body := `{"username":"testuser","password":"test123","role":"developer"}`
	req := authRequest("POST", "/api/v1/admin/users", body)
	w := httptest.NewRecorder()
	testEngine.ServeHTTP(w, req)
	checkStatus(t, w, 201)
}

func TestListRoles(t *testing.T) {
	setupTestEnv(t)
	req := authRequest("GET", "/api/v1/admin/roles", "")
	w := httptest.NewRecorder()
	testEngine.ServeHTTP(w, req)
	checkStatus(t, w, 200)
}

func TestListAPIKeys(t *testing.T) {
	setupTestEnv(t)
	req := authRequest("GET", "/api/v1/apikeys", "")
	w := httptest.NewRecorder()
	testEngine.ServeHTTP(w, req)
	checkStatus(t, w, 200)
}

func TestCreateAPIKey(t *testing.T) {
	setupTestEnv(t)
	body := `{"name":"test-key","permissions":["connections:list"]}`
	req := authRequest("POST", "/api/v1/apikeys", body)
	w := httptest.NewRecorder()
	testEngine.ServeHTTP(w, req)
	checkStatus(t, w, 201)
}

func TestTrafficStats(t *testing.T) {
	setupTestEnv(t)
	req := authRequest("GET", "/api/v1/traffic/stats", "")
	w := httptest.NewRecorder()
	testEngine.ServeHTTP(w, req)
	checkStatus(t, w, 200)
}

func TestActivity(t *testing.T) {
	setupTestEnv(t)
	req := authRequest("GET", "/api/v1/admin/activity", "")
	w := httptest.NewRecorder()
	testEngine.ServeHTTP(w, req)
	checkStatus(t, w, 200)
}

func TestRefreshToken(t *testing.T) {
	setupTestEnv(t)
	body := fmt.Sprintf(`{"token":"%s"}`, testToken)
	req := httptest.NewRequest("POST", "/api/v1/auth/refresh", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+testToken)
	w := httptest.NewRecorder()
	testEngine.ServeHTTP(w, req)
	checkStatus(t, w, 200)
}
