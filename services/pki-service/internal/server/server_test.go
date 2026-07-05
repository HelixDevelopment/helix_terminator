package server_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/helixdevelopment/pki-service/internal/server"
)

func TestNewServer(t *testing.T) {
	srv, err := server.New(nil)
	require.NoError(t, err)
	assert.NotNil(t, srv)
	assert.NotNil(t, srv.Router())
}
