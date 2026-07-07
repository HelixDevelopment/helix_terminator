package server_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/helixdevelopment/pki-service/internal/server"
)

func TestNewServer(t *testing.T) {
	t.Setenv("PKI_ENCRYPTION_KEY", "test-encryption-key-32-bytes-long!!")
	srv, err := server.New(nil)
	require.NoError(t, err)
	assert.NotNil(t, srv)
	assert.NotNil(t, srv.Router())
}
