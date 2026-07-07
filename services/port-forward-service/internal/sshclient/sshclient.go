// Package sshclient wraps golang.org/x/crypto/ssh client connections with
// mandatory host-key verification. This mirrors the proven pattern already
// shipped by services/ssh-proxy-service/internal/sshclient (Constitution
// §11.4.74 — extend/reuse a proven pattern, do not reimplement SSH from
// scratch). It is duplicated rather than imported because each HelixTerminator
// microservice is an independently-versioned, independently-deployable Go
// module (Constitution §11.4.28 — services stay decoupled, no cross-service
// nested imports).
package sshclient

import (
	"fmt"
	"net"
	"os"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"
)

// SSHClient wraps an active ssh.Client connection.
type SSHClient struct {
	client    *ssh.Client
	connected bool
}

// Connect establishes a REAL SSH connection to the given host with mandatory
// host-key verification (SSH_KNOWN_HOSTS is required — there is no
// InsecureIgnoreHostKey escape hatch here, by design).
func Connect(host, port, username string, authMethod ssh.AuthMethod) (*SSHClient, error) {
	if authMethod == nil {
		return nil, fmt.Errorf("ssh auth method is required")
	}
	addr := net.JoinHostPort(host, port)

	knownHostsPath := os.Getenv("SSH_KNOWN_HOSTS")
	if knownHostsPath == "" {
		return nil, fmt.Errorf("SSH_KNOWN_HOSTS environment variable is required for host key verification")
	}
	hostKeyCallback, err := knownhosts.New(knownHostsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load known_hosts file %q: %w", knownHostsPath, err)
	}

	config := &ssh.ClientConfig{
		User:            username,
		Auth:            []ssh.AuthMethod{authMethod},
		HostKeyCallback: hostKeyCallback,
		Timeout:         15 * time.Second,
	}

	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("failed to dial SSH server: %w", err)
	}

	return &SSHClient{
		client:    client,
		connected: true,
	}, nil
}

// Client exposes the underlying *ssh.Client for dialing/listening through the
// established connection (direct-tcpip dials for -L, tcpip-forward listens
// for -R).
func (s *SSHClient) Client() *ssh.Client {
	return s.client
}

// Close terminates the SSH connection.
func (s *SSHClient) Close() error {
	if s.client == nil {
		return nil
	}
	s.connected = false
	return s.client.Close()
}

// IsConnected reports whether the SSH client is connected.
func (s *SSHClient) IsConnected() bool {
	return s.connected
}

// AuthMethodFromPassword returns an ssh.AuthMethod for password authentication.
func AuthMethodFromPassword(password string) ssh.AuthMethod {
	return ssh.Password(password)
}

// AuthMethodFromKey returns an ssh.AuthMethod for private-key authentication.
func AuthMethodFromKey(privateKeyPEM string) (ssh.AuthMethod, error) {
	signer, err := ssh.ParsePrivateKey([]byte(privateKeyPEM))
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}
	return ssh.PublicKeys(signer), nil
}

// AuthMethodFromAgent returns an ssh.AuthMethod using the local SSH agent
// (SSH_AUTH_SOCK). Returns nil when no agent is available.
func AuthMethodFromAgent() ssh.AuthMethod {
	sock := os.Getenv("SSH_AUTH_SOCK")
	if sock == "" {
		return nil
	}
	conn, err := net.Dial("unix", sock)
	if err != nil {
		return nil
	}
	ag := agent.NewClient(conn)
	return ssh.PublicKeysCallback(ag.Signers)
}
