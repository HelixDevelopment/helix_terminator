package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/helixdevelopment/ssh-proxy-service/internal/repository"
	"github.com/helixdevelopment/ssh-proxy-service/internal/wshandler"
)

func setupTestHandler() (*Handler, *gin.Engine) {
	gin.SetMode(gin.TestMode)
	repo := &repository.InMemoryRepository{}
	sm := wshandler.NewSessionManager()
	h := New(repo, sm)
	return h, gin.New()
}

func TestHealthCheck(t *testing.T) {
	h, _ := setupTestHandler()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	h.HealthCheck(c)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "healthy")
	assert.Contains(t, w.Body.String(), "ssh-proxy-service")
}

func TestReadinessCheck(t *testing.T) {
	h, _ := setupTestHandler()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	// gin test context doesn't have a request; set one
	c.Request, _ = http.NewRequest("GET", "/healthz/ready", nil)
	h.ReadinessCheck(c)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "ready")
}

func TestGetSSHSession_NotFound(t *testing.T) {
	h, r := setupTestHandler()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/ssh/sessions/00000000-0000-0000-0000-000000000000", nil)
	r.GET("/api/v1/ssh/sessions/:id", h.GetSSHSession)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}
