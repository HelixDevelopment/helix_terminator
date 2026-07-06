package model

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestSessionStatusConstants(t *testing.T) {
	assert.Equal(t, "active", string(SessionStatusActive))
	assert.Equal(t, "ended", string(SessionStatusEnded))
}

func TestCollaborationSession_Model(t *testing.T) {
	s := CollaborationSession{
		ID:     uuid.New(),
		HostID: uuid.New(),
		Name:   "test-session",
		Status: SessionStatusActive,
	}
	assert.NotEqual(t, uuid.Nil, s.ID)
	assert.NotEqual(t, uuid.Nil, s.HostID)
	assert.Equal(t, "test-session", s.Name)
	assert.Equal(t, SessionStatusActive, s.Status)
}
