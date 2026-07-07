// Package forwarder establishes and tears down REAL SSH port-forwarding
// tunnels (local "-L", remote "-R", dynamic "-D"/SOCKS5) on top of a real
// golang.org/x/crypto/ssh client connection (services/port-forward-service/
// internal/sshclient — the proven pattern already shipped by
// ssh-proxy-service, reused rather than reimplemented, Constitution
// §11.4.74). Status is never fabricated: a Tunnel only exists once its
// listener is really bound AND the SSH connection is really established
// (Constitution §11.4 anti-bluff covenant / §11.4.108 runtime-signature).
package forwarder

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/google/uuid"
	"golang.org/x/crypto/ssh"

	"github.com/helixdevelopment/port-forward-service/internal/sshclient"
)

// ErrTunnelNotRunning is returned when Stop/Metrics is called for a forward
// that has no live in-memory tunnel (never started, already stopped, or the
// manager was restarted).
var ErrTunnelNotRunning = errors.New("tunnel is not running")

// ErrAlreadyRunning is returned when Start is called for a forward that
// already has a live tunnel.
var ErrAlreadyRunning = errors.New("tunnel is already running")

// Config describes one forward to establish.
type Config struct {
	ID          uuid.UUID
	ForwardType string // local | remote | dynamic
	BindAddress string // local bind address for the listener side
	LocalPort   int    // 0 == let the OS choose an ephemeral port
	RemoteHost  string // local: target reached THROUGH ssh; remote: local target reached FROM the inbound ssh forward
	RemotePort  int

	SSHHost     string
	SSHPort     int
	SSHUsername string
	AuthMethod  ssh.AuthMethod
}

// Metrics is a point-in-time snapshot of a running tunnel's REAL activity —
// never fabricated, always read from live atomic counters.
type Metrics struct {
	ConnectionsTotal  int64 `json:"connectionsTotal"`
	ActiveConnections int64 `json:"activeConnections"`
	BytesSent         int64 `json:"bytesSent"`
	BytesReceived     int64 `json:"bytesReceived"`
}

// Tunnel is one running, real forward: a bound listener plus (for local and
// dynamic forwards) a live SSH client connection.
type Tunnel struct {
	cfg       Config
	sshClient *sshclient.SSHClient
	listener  net.Listener

	stopCh    chan struct{}
	closeOnce sync.Once
	wg        sync.WaitGroup

	connectionsTotal  int64
	activeConnections int64
	bytesSent         int64
	bytesReceived     int64
}

// Addr returns the real, resolved local address of the tunnel's listener
// (useful when Config.LocalPort was 0 and the OS picked an ephemeral port).
func (t *Tunnel) Addr() net.Addr {
	return t.listener.Addr()
}

// Metrics returns a live snapshot of this tunnel's real traffic counters.
func (t *Tunnel) Metrics() Metrics {
	return Metrics{
		ConnectionsTotal:  atomic.LoadInt64(&t.connectionsTotal),
		ActiveConnections: atomic.LoadInt64(&t.activeConnections),
		BytesSent:         atomic.LoadInt64(&t.bytesSent),
		BytesReceived:     atomic.LoadInt64(&t.bytesReceived),
	}
}

// Close really tears the tunnel down: the listener is closed (so subsequent
// connects to it fail with connection-refused), every in-flight connection
// handler is drained, and the SSH connection is closed last. Safe to call
// more than once.
func (t *Tunnel) Close() error {
	t.closeOnce.Do(func() {
		close(t.stopCh)
		if t.listener != nil {
			_ = t.listener.Close()
		}
		t.wg.Wait()
		if t.sshClient != nil {
			_ = t.sshClient.Close()
		}
	})
	return nil
}

// Manager owns the set of currently-running tunnels, keyed by forward ID.
// It holds NO database state — it is the pure, real-runtime half of the
// forward's lifecycle; the HTTP handler layer is responsible for persisting
// the corresponding Status.
type Manager struct {
	mu      sync.Mutex
	tunnels map[uuid.UUID]*Tunnel
}

// NewManager creates an empty tunnel manager.
func NewManager() *Manager {
	return &Manager{tunnels: make(map[uuid.UUID]*Tunnel)}
}

// Start establishes a REAL tunnel for cfg: dials a real SSH connection,
// binds a real listener (or asks the SSH server to), and begins serving
// real traffic. It returns only after the tunnel is genuinely up — there is
// no code path that reports success without a bound listener AND (for
// local/dynamic) a connected SSH client.
func (m *Manager) Start(ctx context.Context, cfg Config) (*Tunnel, error) {
	m.mu.Lock()
	if _, exists := m.tunnels[cfg.ID]; exists {
		m.mu.Unlock()
		return nil, ErrAlreadyRunning
	}
	m.mu.Unlock()

	t, err := newTunnel(cfg)
	if err != nil {
		return nil, err
	}

	m.mu.Lock()
	m.tunnels[cfg.ID] = t
	m.mu.Unlock()
	return t, nil
}

// Stop really tears down the tunnel for id (closes the listener + SSH
// connection) and removes it from the manager. Returns ErrTunnelNotRunning
// if there is no live tunnel for id.
func (m *Manager) Stop(id uuid.UUID) error {
	m.mu.Lock()
	t, ok := m.tunnels[id]
	if ok {
		delete(m.tunnels, id)
	}
	m.mu.Unlock()
	if !ok {
		return ErrTunnelNotRunning
	}
	return t.Close()
}

// Get returns the live tunnel for id, if any.
func (m *Manager) Get(id uuid.UUID) (*Tunnel, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.tunnels[id]
	return t, ok
}

func newTunnel(cfg Config) (*Tunnel, error) {
	switch cfg.ForwardType {
	case ForwardTypeLocal, "":
		return newLocalTunnel(cfg)
	case ForwardTypeRemote:
		return newRemoteTunnel(cfg)
	case ForwardTypeDynamic:
		return newDynamicTunnel(cfg)
	default:
		return nil, ErrUnsupportedForwardType
	}
}

// newLocalTunnel implements "-L": bind a REAL local listener; for every
// accepted connection, dial the target THROUGH the real SSH connection and
// copy bytes both ways.
func newLocalTunnel(cfg Config) (*Tunnel, error) {
	sshConn, err := dial(cfg)
	if err != nil {
		return nil, err
	}

	bindAddr := net.JoinHostPort(orDefault(cfg.BindAddress, "127.0.0.1"), strconv.Itoa(cfg.LocalPort))
	listener, err := net.Listen("tcp", bindAddr)
	if err != nil {
		sshConn.Close()
		return nil, fmt.Errorf("failed to bind local listener on %s: %w", bindAddr, err)
	}

	t := &Tunnel{cfg: cfg, sshClient: sshConn, listener: listener, stopCh: make(chan struct{})}
	t.wg.Add(1)
	go t.acceptLoop(func(conn net.Conn) {
		target := net.JoinHostPort(cfg.RemoteHost, strconv.Itoa(cfg.RemotePort))
		remote, err := sshConn.Client().Dial("tcp", target)
		if err != nil {
			conn.Close()
			return
		}
		t.serveConn(conn, remote)
	})
	return t, nil
}

// newRemoteTunnel implements "-R": ask the SSH server to listen on ITS side
// (a real "tcpip-forward" global request via ssh.Client.Listen); for every
// connection the server forwards back to us, dial OUR local target and copy
// bytes both ways.
func newRemoteTunnel(cfg Config) (*Tunnel, error) {
	sshConn, err := dial(cfg)
	if err != nil {
		return nil, err
	}

	bindAddr := net.JoinHostPort(orDefault(cfg.BindAddress, "0.0.0.0"), strconv.Itoa(cfg.LocalPort))
	listener, err := sshConn.Client().Listen("tcp", bindAddr)
	if err != nil {
		sshConn.Close()
		return nil, fmt.Errorf("failed to request remote listen on %s: %w", bindAddr, err)
	}

	t := &Tunnel{cfg: cfg, sshClient: sshConn, listener: listener, stopCh: make(chan struct{})}
	t.wg.Add(1)
	go t.acceptLoop(func(conn net.Conn) {
		target := net.JoinHostPort(cfg.RemoteHost, strconv.Itoa(cfg.RemotePort))
		local, err := net.Dial("tcp", target)
		if err != nil {
			conn.Close()
			return
		}
		t.serveConn(conn, local)
	})
	return t, nil
}

// newDynamicTunnel implements "-D": bind a REAL local SOCKS5 listener; for
// every accepted connection, perform a genuine (CONNECT-only) SOCKS5
// handshake and dial the negotiated target THROUGH the SSH connection.
func newDynamicTunnel(cfg Config) (*Tunnel, error) {
	sshConn, err := dial(cfg)
	if err != nil {
		return nil, err
	}

	bindAddr := net.JoinHostPort(orDefault(cfg.BindAddress, "127.0.0.1"), strconv.Itoa(cfg.LocalPort))
	listener, err := net.Listen("tcp", bindAddr)
	if err != nil {
		sshConn.Close()
		return nil, fmt.Errorf("failed to bind local SOCKS5 listener on %s: %w", bindAddr, err)
	}

	t := &Tunnel{cfg: cfg, sshClient: sshConn, listener: listener, stopCh: make(chan struct{})}
	t.wg.Add(1)
	go t.acceptLoop(func(conn net.Conn) {
		target, err := socks5Handshake(conn)
		if err != nil {
			conn.Close()
			return
		}
		remote, err := sshConn.Client().Dial("tcp", target)
		if err != nil {
			conn.Close()
			return
		}
		t.serveConn(conn, remote)
	})
	return t, nil
}

func dial(cfg Config) (*sshclient.SSHClient, error) {
	return sshclient.Connect(cfg.SSHHost, strconv.Itoa(orDefaultInt(cfg.SSHPort, 22)), cfg.SSHUsername, cfg.AuthMethod)
}

func (t *Tunnel) acceptLoop(handle func(net.Conn)) {
	defer t.wg.Done()
	for {
		conn, err := t.listener.Accept()
		if err != nil {
			select {
			case <-t.stopCh:
				return
			default:
				return
			}
		}
		t.wg.Add(1)
		atomic.AddInt64(&t.connectionsTotal, 1)
		atomic.AddInt64(&t.activeConnections, 1)
		go func() {
			defer t.wg.Done()
			defer atomic.AddInt64(&t.activeConnections, -1)
			handle(conn)
		}()
	}
}

// serveConn really copies bytes both ways between the two ends of the
// tunnel until both directions are drained, then closes both. Byte counters
// update in REAL TIME as data flows (via countingWriter) rather than only
// after a copy completes, so /metrics reflects live traffic on an active
// tunnel, not just after teardown. A half-close (CloseWrite) on one side is
// propagated so an echo-style / request-response peer observes EOF and can
// shut its side down (this is what a real sshd does for a forwarded conn).
func (t *Tunnel) serveConn(a, b io.ReadWriteCloser) {
	defer a.Close()
	defer b.Close()

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, _ = io.Copy(countingWriter{w: b, n: &t.bytesSent}, a)
		halfClose(b)
	}()
	go func() {
		defer wg.Done()
		_, _ = io.Copy(countingWriter{w: a, n: &t.bytesReceived}, b)
		halfClose(a)
	}()
	wg.Wait()
}

// countingWriter wraps an io.Writer and atomically accumulates the number of
// bytes written through it, so tunnel traffic counters advance as bytes
// actually flow.
type countingWriter struct {
	w io.Writer
	n *int64
}

func (c countingWriter) Write(p []byte) (int, error) {
	written, err := c.w.Write(p)
	if written > 0 {
		atomic.AddInt64(c.n, int64(written))
	}
	return written, err
}

type halfCloseWriter interface {
	CloseWrite() error
}

func halfClose(w io.Writer) {
	if hc, ok := w.(halfCloseWriter); ok {
		_ = hc.CloseWrite()
	}
}

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

func orDefaultInt(v, def int) int {
	if v == 0 {
		return def
	}
	return v
}
