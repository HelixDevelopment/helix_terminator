package model

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestContainerBridgeStatusConstants(t *testing.T) {
	assert.Equal(t, "active", ContainerBridgeStatusActive)
	assert.Equal(t, "inactive", ContainerBridgeStatusInactive)
	assert.Equal(t, "error", ContainerBridgeStatusError)
}

func TestContainerBridge_Model(t *testing.T) {
	b := ContainerBridge{
		ID:          uuid.New(),
		HostID:      uuid.New(),
		ContainerID: "abc123",
		Name:        "test-container",
		Image:       "nginx:latest",
		Status:      ContainerBridgeStatusActive,
		Ports:       []string{"80:8080"},
	}
	assert.NotEqual(t, uuid.Nil, b.ID)
	assert.Equal(t, "abc123", b.ContainerID)
	assert.Equal(t, "nginx:latest", b.Image)
	assert.Equal(t, ContainerBridgeStatusActive, b.Status)
}
