package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/helixdevelopment/ssh-proxy-service/internal/model"
)

func TestInMemoryRepository(t *testing.T) {
	ctx := context.Background()
	repo := &InMemoryRepository{}

	// Create session
	session := &model.SSHSession{
		ID:               uuid.New(),
		UserID:           uuid.New(),
		HostID:           uuid.New(),
		HostAddress:      "192.168.1.1:22",
		Username:         "root",
		AuthType:         "password",
		ConnectionStatus: model.StatusConnecting,
		CreatedAt:        time.Now().UTC(),
	}

	err := repo.CreateSession(ctx, session)
	require.NoError(t, err)

	// Get session
	got, err := repo.GetSessionByID(ctx, session.ID)
	require.NoError(t, err)
	assert.Equal(t, session.ID, got.ID)

	// Update status
	err = repo.UpdateSessionStatus(ctx, session.ID, model.StatusConnected)
	require.NoError(t, err)
	got, err = repo.GetSessionByID(ctx, session.ID)
	require.NoError(t, err)
	assert.Equal(t, model.StatusConnected, got.ConnectionStatus)
	assert.NotNil(t, got.ConnectedAt)

	// List sessions
	sessions, err := repo.ListSessions(ctx, session.UserID, 10, 0)
	require.NoError(t, err)
	assert.Len(t, sessions, 1)

	// Create channel
	localPort := 8080
	channel := &model.SSHChannel{
		ID:          uuid.New(),
		SessionID:   session.ID,
		ChannelType: "session",
		LocalPort:   &localPort,
		CreatedAt:   time.Now().UTC(),
	}
	err = repo.CreateChannel(ctx, channel)
	require.NoError(t, err)

	// List channels
	channels, err := repo.ListChannels(ctx, session.ID)
	require.NoError(t, err)
	assert.Len(t, channels, 1)

	// Ping
	assert.NoError(t, repo.Ping(ctx))
}

func TestInMemoryRepository_GetSessionNotFound(t *testing.T) {
	ctx := context.Background()
	repo := &InMemoryRepository{}
	_, err := repo.GetSessionByID(ctx, uuid.New())
	assert.Error(t, err)
}
