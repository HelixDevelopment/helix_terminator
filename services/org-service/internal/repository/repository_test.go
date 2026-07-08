package repository_test

import (
	"testing"

	"github.com/helixdevelopment/org-service/internal/repository"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	repo := repository.New(nil)
	assert.NotNil(t, repo)
}
