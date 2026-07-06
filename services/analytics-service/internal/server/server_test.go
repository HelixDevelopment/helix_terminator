package server

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/helixdevelopment/analytics-service/internal/handler"
	"github.com/stretchr/testify/assert"
)

func TestNewServer(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := handler.New(nil)
	s := New(h)
	assert.NotNil(t, s)
}
