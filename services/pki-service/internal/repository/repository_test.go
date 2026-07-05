package repository_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/helixdevelopment/pki-service/internal/repository"
)

func TestPostgresRepositoryCheckPool(t *testing.T) {
	repo := repository.NewPostgresRepository(nil)
	assert.NotNil(t, repo)

	err := repo.Ping(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database connection not available")
}
