package server_test

import (
	"testing"

	"github.com/helixdevelopment/org-service/internal/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	srv, err := server.New(nil)
	require.NoError(t, err)
	assert.NotNil(t, srv)
	assert.NotNil(t, srv.Router())
}
