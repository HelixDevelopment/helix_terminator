package model_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/helixdevelopment/config-service/internal/model"
)

func TestConfigToResponse(t *testing.T) {
	now := time.Now().UTC()
	scopeID := uuid.New()
	config := &model.Config{
		ID:          uuid.New(),
		Scope:       "org",
		ScopeID:     &scopeID,
		Key:         "test-key",
		Value:       "test-value",
		ValueType:   "string",
		Description: "A test config",
		IsSecret:    false,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	resp := config.ToResponse()
	assert.Equal(t, config.ID, resp.ID)
	assert.Equal(t, "org", resp.Scope)
	assert.Equal(t, scopeID, *resp.ScopeID)
	assert.Equal(t, "test-key", resp.Key)
	assert.Equal(t, "test-value", resp.Value)
	assert.Equal(t, "string", resp.ValueType)
	assert.Equal(t, "A test config", resp.Description)
	assert.Equal(t, false, resp.IsSecret)
	assert.Equal(t, now, resp.CreatedAt)
	assert.Equal(t, now, resp.UpdatedAt)
}

func TestConfigToResponse_NilScopeID(t *testing.T) {
	now := time.Now().UTC()
	config := &model.Config{
		ID:        uuid.New(),
		Scope:     "global",
		ScopeID:   nil,
		Key:       "global-key",
		Value:     "global-value",
		ValueType: "string",
		CreatedAt: now,
		UpdatedAt: now,
	}

	resp := config.ToResponse()
	assert.Nil(t, resp.ScopeID)
}
