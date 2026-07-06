package model

import "errors"

var (
	ErrInvalidHostID      = errors.New("invalid host id")
	ErrInvalidLocalPort   = errors.New("invalid local port")
	ErrInvalidRemotePort  = errors.New("invalid remote port")
	ErrInvalidRemoteHost  = errors.New("invalid remote host")
	ErrInvalidProtocol    = errors.New("invalid protocol")
	ErrForwardNotFound    = errors.New("forward not found")
	ErrInvalidForwardID   = errors.New("invalid forward id")
)
