package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/helixdevelopment/billing-service/internal/handler"
	"github.com/helixdevelopment/billing-service/internal/model"
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

func TestReadinessCheck(t *testing.T) {
	r := setupTestRouter()
	h := handler.New(nil)
	r.GET("/healthz/ready", h.ReadinessCheck)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/healthz/ready", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", w.Code)
	}
}

func TestCreateSubscriptionValidation(t *testing.T) {
	r := setupTestRouter()
	h := handler.New(nil)
	r.POST("/api/v1/subscriptions", h.CreateSubscription)

	body := model.CreateSubscriptionRequest{
		OrgID:  "not-a-uuid",
		PlanID: "not-a-uuid",
	}
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/subscriptions", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code == http.StatusOK || w.Code == http.StatusCreated {
		t.Fatal("expected non-2xx for invalid input")
	}
}
