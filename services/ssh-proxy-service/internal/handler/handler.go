package handler

import (
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/ssh"

	"github.com/helixdevelopment/ssh-proxy-service/internal/model"
	"github.com/helixdevelopment/ssh-proxy-service/internal/repository"
	"github.com/helixdevelopment/ssh-proxy-service/internal/sshclient"
	"github.com/helixdevelopment/ssh-proxy-service/internal/wshandler"
)

// Handler holds service handlers.
type Handler struct {
	repo       repository.Repository
	sessionMgr *wshandler.SessionManager
}

// New returns a new Handler.
func New(repo repository.Repository, sm *wshandler.SessionManager) *Handler {
	return &Handler{
		repo:       repo,
		sessionMgr: sm,
	}
}

// HealthCheck returns service health status.
func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"service":   "ssh-proxy-service",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// ReadinessCheck returns readiness status.
func (h *Handler) ReadinessCheck(c *gin.Context) {
	ready := true
	if h.repo != nil {
		if err := h.repo.Ping(c.Request.Context()); err != nil {
			ready = false
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"ready":     ready,
		"service":   "ssh-proxy-service",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// ListSSHSessions returns paginated SSH sessions for a user.
func (h *Handler) ListSSHSessions(c *gin.Context) {
	var req model.ListSSHSessionsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	if req.Limit == 0 {
		req.Limit = 20
	}

	sessions, err := h.repo.ListSessions(c.Request.Context(), userID, req.Limit, req.Offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list sessions"})
		return
	}

	var resp []*model.SSHSessionResponse
	for _, s := range sessions {
		resp = append(resp, toResponse(s))
	}
	c.JSON(http.StatusOK, gin.H{"sessions": resp})
}

// GetSSHSession returns a single SSH session by ID.
func (h *Handler) GetSSHSession(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session id"})
		return
	}

	session, err := h.repo.GetSessionByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}

	c.JSON(http.StatusOK, toResponse(session))
}

// TerminateSSHSession marks a session as disconnected and closes the active connection.
func (h *Handler) TerminateSSHSession(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session id"})
		return
	}

	if err := h.repo.UpdateSessionStatus(c.Request.Context(), id, model.StatusDisconnected); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update session"})
		return
	}

	// Unregister from session manager if active
	if h.sessionMgr != nil {
		h.sessionMgr.Unregister(idStr)
	}

	c.Status(http.StatusNoContent)
}

// HandleWebSocket is the WebSocket endpoint for SSH connections.
func (h *Handler) HandleWebSocket(c *gin.Context) {
	var req model.CreateSSHSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.AuthType == "password" && req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "password is required for auth_type=password"})
		return
	}
	if req.AuthType == "key" && req.PrivateKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "private_key is required for auth_type=key"})
		return
	}

	hostID, err := uuid.Parse(req.HostID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid host_id"})
		return
	}
	_ = hostID

	userIDStr := c.GetHeader("X-User-ID")
	if userIDStr == "" {
		userIDStr = "00000000-0000-0000-0000-000000000000"
	}
	userID, _ := uuid.Parse(userIDStr)
	_ = userID

	port := c.Query("port")
	if port == "" {
		port = "22"
	}

	var authMethod ssh.AuthMethod
	switch req.AuthType {
	case "password":
		authMethod = sshclient.AuthMethodFromPassword(req.Password)
	case "key":
		am, err := sshclient.AuthMethodFromKey(req.PrivateKey)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid private key"})
			return
		}
		authMethod = am
	case "agent":
		authMethod = sshclient.AuthMethodFromAgent()
		if authMethod == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "SSH agent not available"})
			return
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported auth_type"})
		return
	}

	connectFunc := func() (*sshclient.SSHClient, *ssh.Session, io.WriteCloser, io.Reader, io.Reader, error) {
		client, err := sshclient.Connect(req.HostAddress, port, req.Username, authMethod)
		if err != nil {
			return nil, nil, nil, nil, nil, err
		}
		session, err := client.OpenSession()
		if err != nil {
			client.Close()
			return nil, nil, nil, nil, nil, err
		}
		stdin, err := session.StdinPipe()
		if err != nil {
			session.Close()
			client.Close()
			return nil, nil, nil, nil, nil, err
		}
		stdout, err := session.StdoutPipe()
		if err != nil {
			stdin.Close()
			session.Close()
			client.Close()
			return nil, nil, nil, nil, nil, err
		}
		stderr, err := session.StderrPipe()
		if err != nil {
			stdin.Close()
			session.Close()
			client.Close()
			return nil, nil, nil, nil, nil, err
		}
		return client, session, stdin, stdout, stderr, nil
	}

	wshandler.HandleWebSocket(c, h.sessionMgr, connectFunc)
}

func toResponse(s *model.SSHSession) *model.SSHSessionResponse {
	return &model.SSHSessionResponse{
		ID:               s.ID,
		UserID:           s.UserID,
		HostID:           s.HostID,
		HostAddress:      s.HostAddress,
		Username:         s.Username,
		AuthType:         s.AuthType,
		ConnectionStatus: s.ConnectionStatus,
		ConnectedAt:      s.ConnectedAt,
		DisconnectedAt:   s.DisconnectedAt,
		LastActivityAt:   s.LastActivityAt,
		CreatedAt:        s.CreatedAt,
	}
}
