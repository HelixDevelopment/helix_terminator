//go:build integration

// Real, anti-bluff proof (Constitution §11.4.123 / §11.4.107) that
// port-forward-service establishes a GENUINE SSH local port-forward: a
// real in-process SSH server (golang.org/x/crypto/ssh ServerConn, handling
// real "direct-tcpip" channel opens) plus a real TCP echo target — bytes
// sent to the service's bound local listener are asserted to have really
// traversed the tunnel and come back from the target. STOP is asserted to
// really tear the tunnel down (post-stop connect fails). The SOCKS5/remote
// blast-radius gate is asserted to return 403 when unauthorized, and to
// allow through once explicitly configured.
package handler_test

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"

	"github.com/helixdevelopment/port-forward-service/internal/handler"
	"github.com/helixdevelopment/port-forward-service/internal/model"
)

// --- fake in-memory repository (no live Postgres required for this proof;
// the real Postgres-backed repository.Repository is exercised separately by
// the repository package's own tests) ---

type fakeRepo struct {
	mu   sync.Mutex
	data map[uuid.UUID]*model.PortForward
}

var _ handler.ForwardRepository = (*fakeRepo)(nil)

func newFakeRepo() *fakeRepo {
	return &fakeRepo{data: make(map[uuid.UUID]*model.PortForward)}
}

func (f *fakeRepo) Ping(context.Context) error { return nil }

func (f *fakeRepo) CreateForward(_ context.Context, fw *model.PortForward) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	cp := *fw
	f.data[fw.ID] = &cp
	return nil
}

func (f *fakeRepo) GetForwardByID(_ context.Context, id uuid.UUID) (*model.PortForward, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	fw, ok := f.data[id]
	if !ok {
		return nil, fmt.Errorf("forward not found")
	}
	cp := *fw
	return &cp, nil
}

func (f *fakeRepo) ListForwards(context.Context, uuid.UUID, int, int) ([]*model.PortForward, int, error) {
	return nil, 0, nil
}

func (f *fakeRepo) UpdateForward(context.Context, uuid.UUID, map[string]interface{}) error {
	return nil
}

func (f *fakeRepo) UpdateStatus(_ context.Context, id uuid.UUID, status string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	fw, ok := f.data[id]
	if !ok {
		return fmt.Errorf("forward not found")
	}
	fw.Status = status
	return nil
}

func (f *fakeRepo) DeleteForward(_ context.Context, id uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.data, id)
	return nil
}

// --- real in-process SSH server (server-side "direct-tcpip" support) ---

func generateTestHostKey(t *testing.T) (ssh.PublicKey, ssh.Signer) {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	privPEMBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	require.NoError(t, err)
	block := &pem.Block{Type: "PRIVATE KEY", Bytes: privPEMBytes}
	privPEM := pem.EncodeToMemory(block)
	rawKey, err := ssh.ParseRawPrivateKey(privPEM)
	require.NoError(t, err)
	signer, err := ssh.NewSignerFromKey(rawKey)
	require.NoError(t, err)
	pub, err := ssh.NewPublicKey(priv.Public())
	require.NoError(t, err)
	return pub, signer
}

// startTestSSHServer stands up a REAL SSH server (real protocol handshake,
// real auth negotiation, real "direct-tcpip" channel handling — genuinely
// dials the requested destination and pipes real bytes). It accepts any
// password (throwaway ephemeral test credential, not a production auth
// posture) so the test does not depend on external key material.
func startTestSSHServer(t *testing.T) (addr string, hostPub ssh.PublicKey) {
	t.Helper()
	config := &ssh.ServerConfig{
		PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			return nil, nil
		},
	}
	pub, signer := generateTestHostKey(t)
	config.AddHostKey(signer)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { listener.Close() })

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go handleTestSSHConn(conn, config)
		}
	}()

	return listener.Addr().String(), pub
}

func handleTestSSHConn(nConn net.Conn, config *ssh.ServerConfig) {
	sshConn, chans, reqs, err := ssh.NewServerConn(nConn, config)
	if err != nil {
		return
	}
	defer sshConn.Close()
	go ssh.DiscardRequests(reqs)
	for newChannel := range chans {
		if newChannel.ChannelType() != "direct-tcpip" {
			_ = newChannel.Reject(ssh.UnknownChannelType, "unsupported channel type")
			continue
		}
		var payload struct {
			DestAddr   string
			DestPort   uint32
			OriginAddr string
			OriginPort uint32
		}
		if err := ssh.Unmarshal(newChannel.ExtraData(), &payload); err != nil {
			_ = newChannel.Reject(ssh.ConnectionFailed, "bad direct-tcpip payload")
			continue
		}
		channel, requests, err := newChannel.Accept()
		if err != nil {
			continue
		}
		go ssh.DiscardRequests(requests)
		target, err := net.Dial("tcp", net.JoinHostPort(payload.DestAddr, strconv.Itoa(int(payload.DestPort))))
		if err != nil {
			_ = channel.Close()
			continue
		}
		go proxyPair(channel, target)
	}
}

// proxyPair mirrors what a REAL sshd does for a direct-tcpip forward: it
// copies bytes both ways AND propagates a half-close (FIN) from one side to
// the other. A real SSH server, when the client closes its write side,
// closes the target's write side too — that is what lets an echo-style
// target observe EOF and shut down. The earlier version omitted the
// half-close propagation, which deadlocked the echo target (it never saw
// EOF) — a harness-fidelity bug, not a defect in the production tunnel.
func proxyPair(a, b io.ReadWriteCloser) {
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); _, _ = io.Copy(b, a); halfCloseWrite(b) }()
	go func() { defer wg.Done(); _, _ = io.Copy(a, b); halfCloseWrite(a) }()
	wg.Wait()
	a.Close()
	b.Close()
}

// halfCloseWrite closes only the write half of w when it supports it
// (net.TCPConn and ssh.Channel both do), sending a FIN/EOF to the peer
// without tearing the whole connection down mid-stream.
func halfCloseWrite(w interface{}) {
	if hc, ok := w.(interface{ CloseWrite() error }); ok {
		_ = hc.CloseWrite()
	}
}

// startEchoServer stands up a REAL TCP target: it echoes back exactly the
// bytes it receives, so a successful round trip through the tunnel is
// mechanically provable (not merely "a connection succeeded").
func startEchoServer(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { listener.Close() })
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				_, _ = io.Copy(c, c)
			}(conn)
		}
	}()
	return listener.Addr().String()
}

func newRouter(h *handler.Handler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/v1/forwards", h.CreateForward)
	router.POST("/api/v1/forwards/:id/start", h.StartForward)
	router.POST("/api/v1/forwards/:id/stop", h.StopForward)
	router.GET("/api/v1/forwards/:id/metrics", h.GetForwardMetrics)
	return router
}

func doJSON(t *testing.T, router *gin.Engine, method, path string, body map[string]interface{}) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, json.NewEncoder(&buf).Encode(body))
	req, err := http.NewRequest(method, path, &buf)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// TestIntegration_LocalForward_RealTraffic is the mandatory anti-bluff
// proof: create -> start (REAL SSH dial + REAL local listener bind) ->
// connect to the bound address -> real bytes traverse client -> local
// listener -> SSH tunnel -> SSH server's direct-tcpip handler -> echo
// target -> back. Then stop really tears the tunnel down (post-stop
// connect fails with connection-refused).
func TestIntegration_LocalForward_RealTraffic(t *testing.T) {
	sshAddr, hostPub := startTestSSHServer(t)
	sshHost, sshPortStr, err := net.SplitHostPort(sshAddr)
	require.NoError(t, err)
	sshPort, err := strconv.Atoi(sshPortStr)
	require.NoError(t, err)

	targetAddr := startEchoServer(t)
	targetHost, targetPortStr, err := net.SplitHostPort(targetAddr)
	require.NoError(t, err)
	targetPort, err := strconv.Atoi(targetPortStr)
	require.NoError(t, err)

	knownHostsPath := writeKnownHosts(t, sshAddr, hostPub)
	t.Setenv("SSH_KNOWN_HOSTS", knownHostsPath)
	t.Setenv("PORT_FORWARD_ALLOW_REMOTE", "")
	t.Setenv("PORT_FORWARD_ALLOW_DYNAMIC", "")

	repo := newFakeRepo()
	h := handler.New(repo)
	router := newRouter(h)

	createResp := doJSON(t, router, "POST", "/api/v1/forwards", map[string]interface{}{
		"hostId":      uuid.New().String(),
		"forwardType": "local",
		"localPort":   0, // let the OS choose an ephemeral port
		"remoteHost":  targetHost,
		"remotePort":  targetPort,
		"protocol":    "tcp",
		"sshHost":     sshHost,
		"sshPort":     sshPort,
		"sshUsername": "testuser",
		"authType":    "password",
	})
	require.Equal(t, http.StatusCreated, createResp.Code, createResp.Body.String())

	var created model.PortForward
	require.NoError(t, json.Unmarshal(createResp.Body.Bytes(), &created))
	assert.Equal(t, model.PortForwardStatusPending, created.Status, "Create must NEVER report Active with zero backing")

	startResp := doJSON(t, router, "POST", "/api/v1/forwards/"+created.ID.String()+"/start", map[string]interface{}{
		"password": "any-password-accepted-by-the-test-server",
	})
	require.Equal(t, http.StatusOK, startResp.Code, startResp.Body.String())

	var started model.StartForwardResponse
	require.NoError(t, json.Unmarshal(startResp.Body.Bytes(), &started))
	assert.Equal(t, model.PortForwardStatusActive, started.Status)
	require.NotEmpty(t, started.BoundAddress)

	// Confirm the persisted catalog record's status reflects the REAL
	// runtime state (Constitution §11.4.108 runtime-signature-as-
	// definition-of-done) — not a value fabricated at Create time.
	stored, err := repo.GetForwardByID(context.Background(), created.ID)
	require.NoError(t, err)
	assert.Equal(t, model.PortForwardStatusActive, stored.Status)

	// --- REAL traffic proof: dial the bound local listener and send bytes
	// that must really traverse: our conn -> local listener -> ssh tunnel
	// -> ssh server's direct-tcpip handler -> echo target -> back. ---
	conn, err := net.Dial("tcp", started.BoundAddress)
	require.NoError(t, err)
	payload := []byte("helix-terminator-real-ssh-tunnel-proof-20260707")
	_, err = conn.Write(payload)
	require.NoError(t, err)
	got := make([]byte, len(payload))
	_, err = io.ReadFull(conn, got)
	require.NoError(t, err)
	assert.Equal(t, payload, got, "bytes must really have traversed the SSH tunnel to the target and back")

	// Metrics must reflect the REAL connection just served. Checked while
	// the tunnel is still UP (byte counters advance in real time), so the
	// assertion is deterministic — the echo round-trip above already proved
	// len(payload) bytes flowed each way.
	metricsResp := httptest.NewRecorder()
	mreq, _ := http.NewRequest("GET", "/api/v1/forwards/"+created.ID.String()+"/metrics", nil)
	router.ServeHTTP(metricsResp, mreq)
	require.Equal(t, http.StatusOK, metricsResp.Code)
	var metrics map[string]int64
	require.NoError(t, json.Unmarshal(metricsResp.Body.Bytes(), &metrics))
	assert.GreaterOrEqual(t, metrics["connectionsTotal"], int64(1))
	assert.GreaterOrEqual(t, metrics["bytesSent"], int64(len(payload)))
	assert.GreaterOrEqual(t, metrics["bytesReceived"], int64(len(payload)))

	require.NoError(t, conn.Close())

	// --- STOP must really tear the tunnel down ---
	stopResp := doJSON(t, router, "POST", "/api/v1/forwards/"+created.ID.String()+"/stop", map[string]interface{}{})
	require.Equal(t, http.StatusOK, stopResp.Code, stopResp.Body.String())

	stoppedRecord, err := repo.GetForwardByID(context.Background(), created.ID)
	require.NoError(t, err)
	assert.Equal(t, model.PortForwardStatusStopped, stoppedRecord.Status)

	_, err = net.Dial("tcp", started.BoundAddress)
	require.Error(t, err, "the local listener must really be closed after Stop — a subsequent connect must fail")
}

// TestIntegration_DynamicForward_GateDenied proves the SOCKS5 (-D)
// blast-radius gate is default-deny: with no PORT_FORWARD_ALLOW_DYNAMIC
// configuration, creating a dynamic forward is refused with a real 403 —
// no catalog entry, no listener, nothing established.
func TestIntegration_DynamicForward_GateDenied(t *testing.T) {
	t.Setenv("PORT_FORWARD_ALLOW_DYNAMIC", "")
	t.Setenv("PORT_FORWARD_ALLOW_REMOTE", "")

	repo := newFakeRepo()
	h := handler.New(repo)
	router := newRouter(h)

	resp := doJSON(t, router, "POST", "/api/v1/forwards", map[string]interface{}{
		"hostId":      uuid.New().String(),
		"forwardType": "dynamic",
		"localPort":   1080,
		"protocol":    "tcp",
		"sshHost":     "unauthorized.example.invalid",
		"sshUsername": "testuser",
		"authType":    "password",
	})
	assert.Equal(t, http.StatusForbidden, resp.Code, resp.Body.String())
	assert.Empty(t, repo.data, "an unauthorized dynamic forward must never reach the catalog")
}

// TestIntegration_RemoteForward_GateDenied mirrors the dynamic-forward gate
// proof for "-R" remote forwarding.
func TestIntegration_RemoteForward_GateDenied(t *testing.T) {
	t.Setenv("PORT_FORWARD_ALLOW_DYNAMIC", "")
	t.Setenv("PORT_FORWARD_ALLOW_REMOTE", "")

	repo := newFakeRepo()
	h := handler.New(repo)
	router := newRouter(h)

	resp := doJSON(t, router, "POST", "/api/v1/forwards", map[string]interface{}{
		"hostId":      uuid.New().String(),
		"forwardType": "remote",
		"localPort":   9000,
		"remoteHost":  "127.0.0.1",
		"remotePort":  80,
		"protocol":    "tcp",
		"sshHost":     "unauthorized.example.invalid",
		"sshUsername": "testuser",
		"authType":    "password",
	})
	assert.Equal(t, http.StatusForbidden, resp.Code, resp.Body.String())
	assert.Empty(t, repo.data, "an unauthorized remote forward must never reach the catalog")
}

// TestIntegration_RemoteForward_GateAllowed_WhenConfigured proves the gate
// is a REAL two-way switch (not permanently closed): once the operator
// explicitly authorizes remote forwarding via config, creation succeeds.
func TestIntegration_RemoteForward_GateAllowed_WhenConfigured(t *testing.T) {
	t.Setenv("PORT_FORWARD_ALLOW_REMOTE", "true")
	t.Setenv("PORT_FORWARD_ALLOW_DYNAMIC", "")

	repo := newFakeRepo()
	h := handler.New(repo)
	router := newRouter(h)

	resp := doJSON(t, router, "POST", "/api/v1/forwards", map[string]interface{}{
		"hostId":      uuid.New().String(),
		"forwardType": "remote",
		"localPort":   9000,
		"remoteHost":  "127.0.0.1",
		"remotePort":  80,
		"protocol":    "tcp",
		"sshHost":     "authorized.example.invalid",
		"sshUsername": "testuser",
		"authType":    "password",
	})
	assert.Equal(t, http.StatusCreated, resp.Code, resp.Body.String())
}

func writeKnownHosts(t *testing.T, addr string, pub ssh.PublicKey) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "known_hosts")
	require.NoError(t, err)
	line := knownhosts.Line([]string{addr}, pub)
	_, err = f.WriteString(line + "\n")
	require.NoError(t, err)
	require.NoError(t, f.Close())
	return f.Name()
}
