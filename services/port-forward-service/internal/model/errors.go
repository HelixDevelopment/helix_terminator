package model

import "errors"

var (
	ErrInvalidHostID     = errors.New("invalid host id")
	ErrInvalidLocalPort  = errors.New("invalid local port")
	ErrInvalidRemotePort = errors.New("invalid remote port")
	ErrInvalidRemoteHost = errors.New("invalid remote host")
	ErrInvalidProtocol   = errors.New("invalid protocol")
	ErrForwardNotFound   = errors.New("forward not found")
	ErrInvalidForwardID  = errors.New("invalid forward id")
	// ErrMissingTarget is returned when a local/remote forward is missing
	// the required remoteHost/remotePort target.
	ErrMissingTarget = errors.New("remoteHost and remotePort are required for local/remote forwards")
	// ErrMissingCredential is returned when StartForward is called without
	// the secret material required by the forward's persisted auth type
	// (password, private key, or a reachable SSH agent).
	ErrMissingCredential = errors.New("missing SSH credential for the configured auth type")
)
