package model

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestSnippet_Model(t *testing.T) {
	s := Snippet{
		ID:          uuid.New(),
		CreatedBy:   uuid.New(),
		Name:        "test-snippet",
		Content:     "echo hello",
		Language:    "bash",
		Tags:        []string{"test", "bash"},
		Description: "A test snippet",
		IsPublic:    false,
		UsageCount:  0,
	}
	assert.NotEqual(t, uuid.Nil, s.ID)
	assert.Equal(t, "test-snippet", s.Name)
	assert.Equal(t, "bash", s.Language)
	assert.Equal(t, []string{"test", "bash"}, s.Tags)
	assert.False(t, s.IsPublic)
}
