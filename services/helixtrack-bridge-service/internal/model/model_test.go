package model

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestHelixTrackBridgeStatusConstants(t *testing.T) {
	assert.Equal(t, "active", HelixTrackBridgeStatusActive)
	assert.Equal(t, "inactive", HelixTrackBridgeStatusInactive)
	assert.Equal(t, "error", HelixTrackBridgeStatusError)
}

func TestHelixTrackBridge_Model(t *testing.T) {
	b := HelixTrackBridge{
		ID:            uuid.New(),
		IntegrationID: "integration-123",
		OrgID:         uuid.New(),
		Name:          "test-integration",
		Status:        HelixTrackBridgeStatusActive,
		Config:        []byte(`{"key":"value"}`),
	}
	assert.NotEqual(t, uuid.Nil, b.ID)
	assert.Equal(t, "integration-123", b.IntegrationID)
	assert.Equal(t, "test-integration", b.Name)
	assert.Equal(t, HelixTrackBridgeStatusActive, b.Status)
}
