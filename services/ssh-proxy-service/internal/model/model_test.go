package model

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestConnectionStatusConstants(t *testing.T) {
	assert.Equal(t, ConnectionStatus("connecting"), StatusConnecting)
	assert.Equal(t, ConnectionStatus("connected"), StatusConnected)
	assert.Equal(t, ConnectionStatus("disconnected"), StatusDisconnected)
	assert.Equal(t, ConnectionStatus("error"), StatusError)
}

func TestSSHSessionCreation(t *testing.T) {
	s := &SSHSession{
		ID:               uuid.New(),
		UserID:           uuid.New(),
		HostID:           uuid.New(),
		HostAddress:      "192.168.1.1:22",
		Username:         "root",
		AuthType:         "password",
		ConnectionStatus: StatusConnecting,
	}
	assert.NotEqual(t, uuid.Nil, s.ID)
	assert.Equal(t, "root", s.Username)
	assert.Equal(t, StatusConnecting, s.ConnectionStatus)
}

func TestSSHChannelCreation(t *testing.T) {
	localPort := 8080
	remotePort := 80
	ch := &SSHChannel{
		ID:          uuid.New(),
		SessionID:   uuid.New(),
		ChannelType: "session",
		LocalPort:   &localPort,
		RemotePort:  &remotePort,
	}
	assert.Equal(t, "session", ch.ChannelType)
	assert.Equal(t, 8080, *ch.LocalPort)
	assert.Equal(t, 80, *ch.RemotePort)
}

func TestTerminalResizeMessage(t *testing.T) {
	msg := TerminalResizeMessage{
		Type: "resize",
		Cols: 120,
		Rows: 40,
	}
	assert.Equal(t, "resize", msg.Type)
	assert.Equal(t, uint32(120), msg.Cols)
	assert.Equal(t, uint32(40), msg.Rows)
}
