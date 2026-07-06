package model_test

import (
	"testing"

	"github.com/helixdevelopment/notification-service/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestNotificationTypes(t *testing.T) {
	// Verify the model package compiles and types are accessible
	_ = model.Notification{Type: "info"}
	_ = model.NotificationPreference{Channel: "email"}
	assert.True(t, true)
}
