package repository_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/helixdevelopment/org-service/internal/repository"
)

func TestNew(t *testing.T) {
	repo := repository.New(nil)
	assert.NotNil(t, repo)
}
