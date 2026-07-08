package model_test

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/helixdevelopment/host-service/internal/model"
)

func TestAuthTypeConstants(t *testing.T) {
	assert.Equal(t, model.AuthType("password"), model.AuthTypePassword)
	assert.Equal(t, model.AuthType("key"), model.AuthTypeKey)
	assert.Equal(t, model.AuthType("agent"), model.AuthTypeAgent)
	assert.Equal(t, model.AuthType("vault_key"), model.AuthTypeVaultKey)
}

func TestConnectionStatusConstants(t *testing.T) {
	assert.Equal(t, model.ConnectionStatus("unknown"), model.StatusUnknown)
	assert.Equal(t, model.ConnectionStatus("online"), model.StatusOnline)
	assert.Equal(t, model.ConnectionStatus("offline"), model.StatusOffline)
	assert.Equal(t, model.ConnectionStatus("error"), model.StatusError)
}

func TestHostJSONSerialization(t *testing.T) {
	id := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	userID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	orgID := uuid.MustParse("33333333-3333-3333-3333-333333333333")

	host := model.Host{
		ID:               id,
		UserID:           userID,
		OrgID:            orgID,
		Name:             "test-host",
		Hostname:         "192.168.1.1",
		Port:             22,
		Username:         "admin",
		AuthType:         model.AuthTypePassword,
		Tags:             []string{"prod"},
		ConnectionStatus: model.StatusOnline,
	}

	data, err := json.Marshal(host)
	assert.NoError(t, err)

	var decoded model.Host
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, host.ID, decoded.ID)
	assert.Equal(t, host.Name, decoded.Name)
	assert.Equal(t, host.Port, decoded.Port)
	assert.Equal(t, host.AuthType, decoded.AuthType)
	assert.Equal(t, host.ConnectionStatus, decoded.ConnectionStatus)
}

func TestCreateHostRequestValidationTags(t *testing.T) {
	req := model.CreateHostRequest{
		Name:     "test",
		Hostname: "host.example.com",
		Port:     22,
		Username: "user",
		AuthType: model.AuthTypeKey,
	}

	data, err := json.Marshal(req)
	assert.NoError(t, err)
	assert.Contains(t, string(data), "name")
	assert.Contains(t, string(data), "hostname")
	assert.Contains(t, string(data), "auth_type")
}
