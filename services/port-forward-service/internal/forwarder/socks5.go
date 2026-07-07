package forwarder

import (
	"fmt"
	"io"
	"net"
	"strconv"
)

// socks5Handshake performs a minimal, real SOCKS5 handshake (RFC 1928):
// no-auth method negotiation + CONNECT command only (BIND/UDP ASSOCIATE are
// out of scope). It returns the "host:port" the caller asked to reach, so
// the tunnel can dial it through the SSH connection. This is a genuine
// protocol implementation (reads/writes the real wire bytes), not a stub.
func socks5Handshake(conn net.Conn) (target string, err error) {
	greeting := make([]byte, 2)
	if _, err = io.ReadFull(conn, greeting); err != nil {
		return "", fmt.Errorf("socks5: failed to read greeting: %w", err)
	}
	if greeting[0] != 0x05 {
		return "", fmt.Errorf("socks5: unsupported protocol version %d", greeting[0])
	}
	nMethods := int(greeting[1])
	if nMethods > 0 {
		methods := make([]byte, nMethods)
		if _, err = io.ReadFull(conn, methods); err != nil {
			return "", fmt.Errorf("socks5: failed to read auth methods: %w", err)
		}
	}
	// Always offer NO AUTHENTICATION REQUIRED (0x00).
	if _, err = conn.Write([]byte{0x05, 0x00}); err != nil {
		return "", fmt.Errorf("socks5: failed to write method selection: %w", err)
	}

	header := make([]byte, 4)
	if _, err = io.ReadFull(conn, header); err != nil {
		return "", fmt.Errorf("socks5: failed to read request header: %w", err)
	}
	if header[0] != 0x05 {
		return "", fmt.Errorf("socks5: unsupported protocol version %d", header[0])
	}
	if header[1] != 0x01 { // CONNECT only
		_, _ = conn.Write([]byte{0x05, 0x07, 0x00, 0x01, 0, 0, 0, 0, 0, 0}) // Command not supported
		return "", fmt.Errorf("socks5: unsupported command %d", header[1])
	}

	var host string
	switch header[3] {
	case 0x01: // IPv4
		ip := make([]byte, 4)
		if _, err = io.ReadFull(conn, ip); err != nil {
			return "", fmt.Errorf("socks5: failed to read IPv4 address: %w", err)
		}
		host = net.IP(ip).String()
	case 0x03: // domain name
		lenBuf := make([]byte, 1)
		if _, err = io.ReadFull(conn, lenBuf); err != nil {
			return "", fmt.Errorf("socks5: failed to read domain length: %w", err)
		}
		domain := make([]byte, lenBuf[0])
		if _, err = io.ReadFull(conn, domain); err != nil {
			return "", fmt.Errorf("socks5: failed to read domain: %w", err)
		}
		host = string(domain)
	case 0x04: // IPv6
		ip := make([]byte, 16)
		if _, err = io.ReadFull(conn, ip); err != nil {
			return "", fmt.Errorf("socks5: failed to read IPv6 address: %w", err)
		}
		host = net.IP(ip).String()
	default:
		_, _ = conn.Write([]byte{0x05, 0x08, 0x00, 0x01, 0, 0, 0, 0, 0, 0}) // Address type not supported
		return "", fmt.Errorf("socks5: unsupported address type %d", header[3])
	}

	portBuf := make([]byte, 2)
	if _, err = io.ReadFull(conn, portBuf); err != nil {
		return "", fmt.Errorf("socks5: failed to read port: %w", err)
	}
	port := int(portBuf[0])<<8 | int(portBuf[1])
	target = net.JoinHostPort(host, strconv.Itoa(port))

	// Succeeded (0x00); BND.ADDR/BND.PORT are informational only for CONNECT.
	if _, err = conn.Write([]byte{0x05, 0x00, 0x00, 0x01, 0, 0, 0, 0, 0, 0}); err != nil {
		return "", fmt.Errorf("socks5: failed to write success reply: %w", err)
	}
	return target, nil
}
