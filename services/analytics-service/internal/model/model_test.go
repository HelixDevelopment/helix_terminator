package model

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestAnalyticsEventTypeConstants(t *testing.T) {
	assert.Equal(t, "session", AnalyticsEventTypeSession)
	assert.Equal(t, "command", AnalyticsEventTypeCommand)
	assert.Equal(t, "transfer", AnalyticsEventTypeTransfer)
	assert.Equal(t, "login", AnalyticsEventTypeLogin)
	assert.Equal(t, "error", AnalyticsEventTypeError)
}

func TestAnalyticsEvent_Model(t *testing.T) {
	e := AnalyticsEvent{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		EventType: AnalyticsEventTypeSession,
		Payload:   []byte(`{"action":"login"}`),
	}
	assert.NotEqual(t, uuid.Nil, e.ID)
	assert.Equal(t, AnalyticsEventTypeSession, e.EventType)
	assert.NotNil(t, e.Payload)
}
