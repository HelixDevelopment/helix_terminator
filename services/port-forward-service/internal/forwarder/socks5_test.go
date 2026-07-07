package forwarder

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSOCKS5Handshake_DomainConnect drives a REAL SOCKS5 CONNECT handshake
// over an in-memory pipe and asserts the negotiated target host:port is
// parsed correctly and the success reply is written back — proving the
// dynamic (-D) SOCKS5 parser genuinely speaks the protocol (RFC 1928),
// not a stub.
func TestSOCKS5Handshake_DomainConnect(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	type result struct {
		target string
		err    error
	}
	resCh := make(chan result, 1)
	go func() {
		target, err := socks5Handshake(server)
		resCh <- result{target, err}
	}()

	// Method negotiation: version 5, 1 method, NO-AUTH (0x00).
	_, err := client.Write([]byte{0x05, 0x01, 0x00})
	require.NoError(t, err)

	// Server must select NO-AUTH.
	sel := make([]byte, 2)
	_, err = readFull(t, client, sel)
	require.NoError(t, err)
	assert.Equal(t, []byte{0x05, 0x00}, sel)

	// CONNECT to example.com:443 (domain address type 0x03).
	host := "example.com"
	req := []byte{0x05, 0x01, 0x00, 0x03, byte(len(host))}
	req = append(req, []byte(host)...)
	req = append(req, 0x01, 0xBB) // port 443
	_, err = client.Write(req)
	require.NoError(t, err)

	// Server success reply.
	reply := make([]byte, 10)
	_, err = readFull(t, client, reply)
	require.NoError(t, err)
	assert.Equal(t, byte(0x05), reply[0])
	assert.Equal(t, byte(0x00), reply[1], "REP must be 0x00 (succeeded)")

	res := <-resCh
	require.NoError(t, res.err)
	assert.Equal(t, "example.com:443", res.target)
}

// TestSOCKS5Handshake_IPv4Connect proves the IPv4 (0x01) address form.
func TestSOCKS5Handshake_IPv4Connect(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	resCh := make(chan string, 1)
	errCh := make(chan error, 1)
	go func() {
		target, err := socks5Handshake(server)
		resCh <- target
		errCh <- err
	}()

	_, err := client.Write([]byte{0x05, 0x01, 0x00})
	require.NoError(t, err)
	sel := make([]byte, 2)
	_, err = readFull(t, client, sel)
	require.NoError(t, err)

	// CONNECT to 127.0.0.1:8080 (address type 0x01).
	_, err = client.Write([]byte{0x05, 0x01, 0x00, 0x01, 127, 0, 0, 1, 0x1F, 0x90})
	require.NoError(t, err)
	reply := make([]byte, 10)
	_, err = readFull(t, client, reply)
	require.NoError(t, err)

	assert.NoError(t, <-errCh)
	assert.Equal(t, "127.0.0.1:8080", <-resCh)
}

// TestSOCKS5Handshake_RejectsBadVersion proves a non-SOCKS5 greeting is
// rejected rather than mis-parsed.
func TestSOCKS5Handshake_RejectsBadVersion(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	errCh := make(chan error, 1)
	go func() {
		_, err := socks5Handshake(server)
		errCh <- err
	}()

	// Write exactly the 2 greeting bytes the handshake reads before the
	// version check — net.Pipe is unbuffered, so writing more than the peer
	// consumes would block once the handshake rejects and stops reading.
	_, err := client.Write([]byte{0x04, 0x01}) // SOCKS4 (bad) version
	require.NoError(t, err)

	select {
	case err := <-errCh:
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported protocol version")
	case <-time.After(2 * time.Second):
		t.Fatal("socks5Handshake did not reject a bad version in time")
	}
}

func readFull(t *testing.T, c net.Conn, buf []byte) (int, error) {
	t.Helper()
	_ = c.SetReadDeadline(time.Now().Add(2 * time.Second))
	total := 0
	for total < len(buf) {
		n, err := c.Read(buf[total:])
		total += n
		if err != nil {
			return total, err
		}
	}
	return total, nil
}
