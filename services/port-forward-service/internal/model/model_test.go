package model

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestPortForwardStatusConstants(t *testing.T) {
	assert.Equal(t, "active", PortForwardStatusActive)
	assert.Equal(t, "inactive", PortForwardStatusInactive)
	assert.Equal(t, "deleted", PortForwardStatusDeleted)
}

func TestPortForward_Model(t *testing.T) {
	pf := PortForward{
		ID:         uuid.New(),
		HostID:     uuid.New(),
		LocalPort:  8080,
		RemotePort: 80,
		RemoteHost: "localhost",
		Protocol:   "tcp",
		Status:     PortForwardStatusActive,
	}
	assert.NotEqual(t, uuid.Nil, pf.ID)
	assert.NotEqual(t, uuid.Nil, pf.HostID)
	assert.Equal(t, 8080, pf.LocalPort)
	assert.Equal(t, 80, pf.RemotePort)
	assert.Equal(t, "localhost", pf.RemoteHost)
	assert.Equal(t, "tcp", pf.Protocol)
	assert.Equal(t, PortForwardStatusActive, pf.Status)
}
