package handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/helixdevelopment/vault-service/internal/handler"
)

func TestHealthCheck(t *testing.T) {
	t.Skip("TODO: implement real health check test")
	gin.SetMode(gin.TestMode)
	h := handler.New()
	r := gin.New()
	r.GET("/health", h.HealthCheck)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/health", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
