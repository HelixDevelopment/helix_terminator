package model_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/helixdevelopment/vault-service/internal/model"
)

func TestValidSecretTypes(t *testing.T) {
	types := model.ValidSecretTypes()
	assert.Len(t, types, 5)
	assert.Contains(t, types, "ssh_key")
	assert.Contains(t, types, "api_token")
	assert.Contains(t, types, "password")
	assert.Contains(t, types, "certificate")
	assert.Contains(t, types, "env_var")
}

func TestToSecretResponse(t *testing.T) {
	secret := &model.Secret{
		ID:        uuid.MustParse("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"),
		UserID:    uuid.MustParse("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12"),
		OrgID:     uuid.MustParse("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13"),
		Name:      "test-secret",
		Type:      model.SecretTypeAPIToken,
		Metadata:  map[string]interface{}{"key": "value"},
		Tags:      []string{"prod"},
		CreatedAt: secret.CreatedAt,
		UpdatedAt: secret.UpdatedAt,
	}
	resp := model.ToSecretResponse(secret)
	assert.Equal(t, secret.ID, resp.ID)
	assert.Equal(t, secret.Name, resp.Name)
	assert.Equal(t, "api_token", resp.Type)
	assert.Equal(t, secret.Metadata, resp.Metadata)
	assert.Equal(t, secret.Tags, resp.Tags)
}
