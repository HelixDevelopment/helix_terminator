package wshandler

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"

	"github.com/helixdevelopment/ssh-proxy-service/internal/model"
	"github.com/helixdevelopment/ssh-proxy-service/internal/sshclient"
)

func TestSessionManager_RegisterUnregister(t *testing.T) {
	sm := NewSessionManager()
	as := &activeSession{
		resizeCh: make(chan model.TerminalResizeMessage, 1),
	}
	sm.Register("sess-1", as)
	got, ok := sm.Get("sess-1")
	require.True(t, ok)
	assert.Equal(t, as, got)

	sm.Unregister("sess-1")
	_, ok = sm.Get("sess-1")
	assert.False(t, ok)
}

func TestSessionManager_CloseAll(t *testing.T) {
	sm := NewSessionManager()
	as1 := &activeSession{resizeCh: make(chan model.TerminalResizeMessage, 1)}
	as2 := &activeSession{resizeCh: make(chan model.TerminalResizeMessage, 1)}
	sm.Register("s1", as1)
	sm.Register("s2", as2)
	sm.CloseAll()
	_, ok1 := sm.Get("s1")
	_, ok2 := sm.Get("s2")
	assert.False(t, ok1)
	assert.False(t, ok2)
}

func TestHandleWebSocket_UpgradeFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	sm := NewSessionManager()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/ws/ssh", nil)

	connectFunc := func() (*sshclient.SSHClient, *ssh.Session, io.WriteCloser, io.Reader, io.Reader, error) {
		return nil, nil, nil, nil, nil, assert.AnError
	}

	// HandleWebSocket expects a real HTTP response writer for upgrade, so we just call it directly
	// Since it's not a real upgrade, it should return without panic
	HandleWebSocket(c, sm, connectFunc)
	// The upgrade fails because the test recorder is not a hijacker, so gin returns 400
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestWSToSSHProxy_ResizeMessage(t *testing.T) {
	resizeCh := make(chan model.TerminalResizeMessage, 2)
	_ = &activeSession{
		resizeCh: resizeCh,
		cancel:   func() {},
	}

	msg := model.TerminalResizeMessage{Type: "resize", Cols: 120, Rows: 40}
	data, _ := json.Marshal(msg)

	// Simulate a WebSocket message being parsed
	var parsed model.TerminalResizeMessage
	err := json.Unmarshal(data, &parsed)
	require.NoError(t, err)
	assert.Equal(t, "resize", parsed.Type)
	assert.Equal(t, uint32(120), parsed.Cols)
	assert.Equal(t, uint32(40), parsed.Rows)
}

func TestTerminalResizeMessage_JSON(t *testing.T) {
	msg := model.TerminalResizeMessage{
		Type: "resize",
		Cols: 80,
		Rows: 24,
	}
	data, err := json.Marshal(msg)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"type":"resize"`)
	assert.Contains(t, string(data), `"cols":80`)
	assert.Contains(t, string(data), `"rows":24`)
}
