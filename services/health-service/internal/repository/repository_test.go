package repository_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/helixdevelopment/health-service/internal/repository"
)

func TestPostgresRepositoryPing(t *testing.T) {
	repo := repository.NewPostgresRepository()
	err := repo.Ping(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not implemented")
}
