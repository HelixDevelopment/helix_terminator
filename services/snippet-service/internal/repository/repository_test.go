package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	r := New(nil)
	assert.NotNil(t, r)
	err := r.checkPool()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database not connected")
}
