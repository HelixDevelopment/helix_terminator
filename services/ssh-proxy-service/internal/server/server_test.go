package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestServerCreation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	srv, err := New(nil)
	assert.NoError(t, err)
	assert.NotNil(t, srv)
	assert.NotNil(t, srv.Router())
	assert.NotNil(t, srv.SessionManager())
}

func TestRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	srv, err := New(nil)
	assert.NoError(t, err)

	router := srv.Router().(*gin.Engine)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/healthz", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/healthz/ready", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}
