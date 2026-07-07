package wshandler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"golang.org/x/crypto/ssh"

	"github.com/helixdevelopment/ssh-proxy-service/internal/model"
	"github.com/helixdevelopment/ssh-proxy-service/internal/sshclient"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// SessionManager tracks active SSH sessions.
type SessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*activeSession
}

// activeSession holds the runtime state for one WebSocket-to-SSH bridge.
type activeSession struct {
	sshClient   *sshclient.SSHClient
	sshSession  *ssh.Session
	wsConn      *websocket.Conn
	stdin       io.WriteCloser
	stdout      io.Reader
	stderr      io.Reader
	cancel      context.CancelFunc
	resizeCh    chan model.TerminalResizeMessage
	closeOnce   sync.Once
}

// NewSessionManager creates a new SessionManager.
func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*activeSession),
	}
}

// Register adds an active session to the manager.
func (m *SessionManager) Register(id string, s *activeSession) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[id] = s
}

// Unregister removes an active session and cleans up resources.
func (m *SessionManager) Unregister(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, ok := m.sessions[id]; ok {
		m.cleanup(s)
		delete(m.sessions, id)
	}
}

// Get retrieves an active session by ID.
func (m *SessionManager) Get(id string) (*activeSession, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[id]
	return s, ok
}

// CloseAll terminates every active session.
func (m *SessionManager) CloseAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, s := range m.sessions {
		m.cleanup(s)
		delete(m.sessions, id)
	}
}

func (m *SessionManager) cleanup(s *activeSession) {
	if s.cancel != nil {
		s.cancel()
	}
	if s.stdin != nil {
		_ = s.stdin.Close()
	}
	if s.sshSession != nil {
		_ = s.sshSession.Close()
	}
	if s.sshClient != nil {
		_ = s.sshClient.Close()
	}
	if s.wsConn != nil {
		_ = s.wsConn.Close()
	}
	s.closeOnce.Do(func() {
		close(s.resizeCh)
	})
}

// HandleWebSocket upgrades the HTTP connection to WebSocket and bridges to SSH.
func HandleWebSocket(c *gin.Context, sm *SessionManager, connectFunc func() (*sshclient.SSHClient, *ssh.Session, io.WriteCloser, io.Reader, io.Reader, error)) {
	wsConn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("websocket upgrade failed: %v", err)
		return
	}
	defer wsConn.Close()

	sshClient, sshSession, stdin, stdout, stderr, err := connectFunc()
	if err != nil {
		log.Printf("SSH connection failed: %v", err)
		_ = wsConn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("SSH connection failed: %v\r\n", err)))
		return
	}

	ctx, cancel := context.WithCancel(c.Request.Context())
	as := &activeSession{
		sshClient:  sshClient,
		sshSession: sshSession,
		wsConn:     wsConn,
		stdin:      stdin,
		stdout:     stdout,
		stderr:     stderr,
		cancel:     cancel,
		resizeCh:   make(chan model.TerminalResizeMessage, 8),
	}

	sessionID := c.Query("session_id")
	if sessionID == "" {
		sessionID = fmt.Sprintf("%d", time.Now().UnixNano())
	}
	sm.Register(sessionID, as)
	defer sm.Unregister(sessionID)

	// Set up PTY
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}
	if err := sshSession.RequestPty("xterm-256color", 80, 24, modes); err != nil {
		log.Printf("request pty failed: %v", err)
		return
	}
	if err := sshSession.Shell(); err != nil {
		log.Printf("start shell failed: %v", err)
		return
	}

	// Start goroutines
	go SSHToWSProxy(ctx, as)
	go WSToSSHProxy(ctx, as)
	go handleResize(ctx, as)

	// Keepalive ping
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := wsConn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(5*time.Second)); err != nil {
				log.Printf("ping failed: %v", err)
				cancel()
				return
			}
		}
	}
}

// SSHToWSProxy reads from SSH stdout/stderr and writes to WebSocket.
func SSHToWSProxy(ctx context.Context, as *activeSession) {
	buf := make([]byte, 4096)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		n, err := as.stdout.Read(buf)
		if n > 0 {
			if err := as.wsConn.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
				log.Printf("ws write error: %v", err)
				as.cancel()
				return
			}
		}
		if err != nil {
			if err != io.EOF {
				log.Printf("ssh stdout read error: %v", err)
			}
			as.cancel()
			return
		}
	}
}

// WSToSSHProxy reads from WebSocket and writes to SSH stdin.
func WSToSSHProxy(ctx context.Context, as *activeSession) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		msgType, data, err := as.wsConn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("ws read error: %v", err)
			}
			as.cancel()
			return
		}

		switch msgType {
		case websocket.BinaryMessage, websocket.TextMessage:
			// Try to parse as resize message first
			var resize model.TerminalResizeMessage
			if err := json.Unmarshal(data, &resize); err == nil && resize.Type == "resize" {
				select {
				case as.resizeCh <- resize:
				default:
				}
				continue
			}
			if _, err := as.stdin.Write(data); err != nil {
				log.Printf("ssh stdin write error: %v", err)
				as.cancel()
				return
			}
		case websocket.PingMessage:
			if err := as.wsConn.WriteControl(websocket.PongMessage, data, time.Now().Add(5*time.Second)); err != nil {
				log.Printf("pong write error: %v", err)
				as.cancel()
				return
			}
		case websocket.CloseMessage:
			as.cancel()
			return
		}
	}
}

func handleResize(ctx context.Context, as *activeSession) {
	for {
		select {
		case <-ctx.Done():
			return
		case resize, ok := <-as.resizeCh:
			if !ok {
				return
			}
			if err := as.sshSession.WindowChange(int(resize.Rows), int(resize.Cols)); err != nil {
				log.Printf("window change failed: %v", err)
			}
		}
	}
}
