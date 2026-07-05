package sshclient

import (
	"fmt"
	"io"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

func TestAuthMethodFromPassword(t *testing.T) {
	am := AuthMethodFromPassword("secret123")
	require.NotNil(t, am)
	// ssh.AuthMethod is a function; we can verify it's non-nil
	assert.NotNil(t, am)
}

func TestAuthMethodFromKey_Valid(t *testing.T) {
	// Ed25519 test key
	privateKeyPEM := `-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAMwAAAAtzc2gtZW
QyNTUxOQAAACB0aGVyZWFsbHlzZWNyZXRrZXl0ZXN0aW5nAAAAFHRlc3R1c2VyQGV4YW1w
bGUuY29tAAAAHHN0dWJ0ZXN0a2V5Zm9ydW5pdHRlc3Rpbmc=
-----END OPENSSH PRIVATE KEY-----`
	am, err := AuthMethodFromKey(privateKeyPEM)
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
	// Start a mock SSH server for testing
	config := &ssh.ServerConfig{
		NoClientAuth: true,
	}
	privateKey, err := ssh.ParseRawPrivateKey([]byte(`-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAMwAAAAtzc2gtZW
QyNTUxOQAAACB0aGVyZWFsbHlzZWNyZXRrZXl0ZXN0aW5nAAAAFHRlc3R1c2VyQGV4YW1w
bGUuY29tAAAAHHN0dWJ0ZXN0a2V5Zm9ydW5pdHRlc3Rpbmc=
-----END OPENSSH PRIVATE KEY-----`))
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

	addr := listener.Addr().(*net.TCPAddr)
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
