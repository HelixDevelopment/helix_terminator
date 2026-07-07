package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/helixdevelopment/user-service/internal/handler"
	"github.com/helixdevelopment/user-service/internal/model"
)

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	return r
}

func TestHealthCheck(t *testing.T) {
	r := setupTestRouter()
	h := handler.New(nil)
	r.GET("/healthz", h.HealthCheck)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/healthz", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp["status"] != "healthy" {
		t.Fatalf("expected status healthy, got %v", resp["status"])
	}
}

// TestReadinessCheck_NoRepoConfigured asserts the honest failure mode
// when the handler has no repository at all: readiness MUST report
// 503 + a non-ready status, never a fabricated "status":"ready"
// (T8-6). A real, reachable-vs-closed-DB proof lives in the
// "integration"-tagged handler_readiness_integration_test.go, which
// exercises a real PostgreSQL instance end-to-end - this unit test
// only covers the nil-repo edge case that doesn't need a real
// database.
func TestReadinessCheck_NoRepoConfigured(t *testing.T) {
	r := setupTestRouter()
	h := handler.New(nil)
	r.GET("/healthz/ready", h.ReadinessCheck)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/healthz/ready", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503 (no repo configured -> not ready), got %d, body=%s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp["status"] == "ready" {
		t.Fatalf("expected non-ready status, got %v", resp["status"])
	}
}

func TestCreateUserValidation(t *testing.T) {
	r := setupTestRouter()
	h := handler.New(nil)
	r.POST("/api/v1/users", h.CreateUser)

	body := model.CreateUserRequest{
		Email:       "not-an-email",
		DisplayName: "",
		Role:        "invalid-role",
	}
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/users", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code == http.StatusOK || w.Code == http.StatusCreated {
		t.Fatal("expected non-2xx for invalid input")
	}
}
