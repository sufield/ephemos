// Package net provides internal network utilities and interfaces.
package net

import "net"

// Listener represents a network listener interface used internally.
type Listener interface {
	// Accept waits for and returns the next connection
	Accept() (net.Conn, error)
	// Close closes the listener
	Close() error
	// Addr returns the listener's network address
	Addr() net.Addr
}