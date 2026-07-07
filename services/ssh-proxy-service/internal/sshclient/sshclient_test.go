package sshclient

import (
	"fmt"
	"io"
	"net"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

func TestAuthMethodFromPassword(t *testing.T) {
	am := AuthMethodFromPassword("secret123")
	require.NotNil(t, am)
	assert.NotNil(t, am)
}

func TestAuthMethodFromKey_Valid(t *testing.T) {
	// Generate a valid Ed25519 key for testing
	_, privPEM, err := generateTestKey()
	require.NoError(t, err)
	am, err := AuthMethodFromKey(privPEM)
	require.NoError(t, err)
	require.NotNil(t, am)
}

func TestAuthMethodFromKey_Invalid(t *testing.T) {
	am, err := AuthMethodFromKey("not-a-valid-key")
	require.Error(t, err)
	assert.Nil(t, am)
}

func TestAuthMethodFromAgent(t *testing.T) {
	am := AuthMethodFromAgent()
	// In CI environments SSH_AUTH_SOCK is usually unset, so nil is expected
	assert.Nil(t, am)
}

func TestSSHClient_ConnectAndClose(t *testing.T) {
	config := &ssh.ServerConfig{
		NoClientAuth: true,
	}
	pub, privPEM, err := generateTestKey()
	require.NoError(t, err)
	privateKey, err := ssh.ParseRawPrivateKey([]byte(privPEM))
	require.NoError(t, err)
	signer, err := ssh.NewSignerFromKey(privateKey)
	require.NoError(t, err)
	config.AddHostKey(signer)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	go func() {
		conn, _ := listener.Accept()
		if conn == nil {
			return
		}
		_, chans, reqs, err := ssh.NewServerConn(conn, config)
		if err != nil {
			return
		}
		go ssh.DiscardRequests(reqs)
		go func() {
			for newChannel := range chans {
				channel, requests, err := newChannel.Accept()
				if err != nil {
					return
				}
				go func(in <-chan *ssh.Request) {
					for req := range in {
						if req.WantReply {
							req.Reply(req.Type == "shell", nil)
						}
					}
				}(requests)
				go func() {
					io.Copy(channel, channel)
				}()
			}
		}()
	}()

	// Create a temporary known_hosts file with the test server key
	addr := listener.Addr().(*net.TCPAddr)
	hostPort := fmt.Sprintf("127.0.0.1:%d", addr.Port)
	knownHostsFile, err := os.CreateTemp("", "known_hosts")
	require.NoError(t, err)
	defer os.Remove(knownHostsFile.Name())

	line := knownhosts.Line([]string{hostPort}, pub)
	_, err = knownHostsFile.WriteString(line + "\n")
	require.NoError(t, err)
	knownHostsFile.Close()

	t.Setenv("SSH_KNOWN_HOSTS", knownHostsFile.Name())

	client, err := Connect("127.0.0.1", fmt.Sprintf("%d", addr.Port), "testuser", ssh.Password("any"))
	require.NoError(t, err)
	assert.True(t, client.IsConnected())

	session, err := client.OpenSession()
	require.NoError(t, err)
	require.NotNil(t, session)
	assert.NoError(t, session.Close())

	assert.NoError(t, client.Close())
	assert.False(t, client.IsConnected())
}
