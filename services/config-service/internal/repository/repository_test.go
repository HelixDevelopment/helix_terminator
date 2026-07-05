package repository_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/helixdevelopment/config-service/internal/model"
	"github.com/helixdevelopment/config-service/internal/repository"
)

func TestRepositoryCheckPool(t *testing.T) {
	repo := repository.New(nil)

	err := repo.Ping(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database not connected")

	_, err = repo.GetConfigByID(context.Background(), [16]byte{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database not connected")

	_, err = repo.GetConfigByKey(context.Background(), "global", nil, "key")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database not connected")

	_, _, err = repo.ListConfigs(context.Background(), "global", nil, "", 10, 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database not connected")

	err = repo.UpdateConfig(context.Background(), [16]byte{}, map[string]interface{}{"value": "v"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database not connected")

	err = repo.DeleteConfig(context.Background(), [16]byte{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database not connected")

	err = repo.BulkCreateConfigs(context.Background(), nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database not connected")

	_, err = repo.CountConfigs(context.Background(), "global", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database not connected")
}

func TestRepositoryUpdateConfigNoUpdates(t *testing.T) {
	repo := repository.New(nil)
	err := repo.UpdateConfig(context.Background(), [16]byte{}, map[string]interface{}{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no updates")
}

func TestRepositoryBulkCreateEmpty(t *testing.T) {
	repo := repository.New(nil)
	err := repo.BulkCreateConfigs(context.Background(), []*model.Config{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database not connected")
}
