package repository_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/helixdevelopment/host-service/internal/repository"
)

func TestNewRepository(t *testing.T) {
	repo := repository.New(nil)
	assert.NotNil(t, repo)
}

func TestRepositoryPingWithoutDB(t *testing.T) {
	repo := repository.New(nil)
	err := repo.Ping(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database not connected")
}
