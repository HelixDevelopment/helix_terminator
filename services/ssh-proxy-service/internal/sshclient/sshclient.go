package sshclient

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// SSHClient wraps an active ssh.Client connection.
type SSHClient struct {
	client    *ssh.Client
	mu        sync.RWMutex
	connected bool
	config    *ssh.ClientConfig
	addr      string
}

// Connect establishes an SSH connection to the given host.
func Connect(host, port, username string, authMethod ssh.AuthMethod) (*SSHClient, error) {
	addr := net.JoinHostPort(host, port)
	config := &ssh.ClientConfig{
		User:            username,
		Auth:            []ssh.AuthMethod{authMethod},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO: use known_hosts in production
		Timeout:         15 * time.Second,
	}

	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("failed to dial SSH server: %w", err)
	}

	return &SSHClient{
		client:    client,
		connected: true,
		config:    config,
		addr:      addr,
	}, nil
}

// OpenSession creates a new ssh.Session on the established connection.
func (s *SSHClient) OpenSession() (*ssh.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.connected || s.client == nil {
		return nil, fmt.Errorf("SSH client is not connected")
	}

	session, err := s.client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("failed to open SSH session: %w", err)
	}
	return session, nil
}

// Close terminates the SSH connection.
func (s *SSHClient) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.client != nil {
		err := s.client.Close()
		s.connected = false
		return err
	}
	return nil
}

// IsConnected reports whether the SSH client is connected.
func (s *SSHClient) IsConnected() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.connected
}

// Client exposes the underlying ssh.Client (for advanced use).
func (s *SSHClient) Client() *ssh.Client {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.client
}

// AuthMethodFromPassword returns an ssh.AuthMethod for password authentication.
func AuthMethodFromPassword(password string) ssh.AuthMethod {
	return ssh.Password(password)
}

// AuthMethodFromKey returns an ssh.AuthMethod for private key authentication.
func AuthMethodFromKey(privateKeyPEM string) (ssh.AuthMethod, error) {
	signer, err := ssh.ParsePrivateKey([]byte(privateKeyPEM))
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}
	return ssh.PublicKeys(signer), nil
}

func generateTestKey() (pub ssh.PublicKey, privPEM string, err error) {
	// Generate a real Ed25519 key pair for tests
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, "", err
	}
	privPEMBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return nil, "", err
	}
	block := &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privPEMBytes,
	}
	privPEM = string(pem.EncodeToMemory(block))
	pub, err = ssh.NewPublicKey(priv.Public())
	if err != nil {
		return nil, "", err
	}
	return pub, privPEM, nil
}

// AuthMethodFromAgent returns an ssh.AuthMethod using the local SSH agent.
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
