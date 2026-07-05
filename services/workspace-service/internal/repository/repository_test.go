package repository_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/helixdevelopment/workspace-service/internal/repository"
)

func TestRepositoryCheckPool(t *testing.T) {
	repo := repository.New(nil)
	err := repo.Ping(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database not connected")
}
