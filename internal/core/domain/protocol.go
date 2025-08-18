package domain

import "fmt"

// Protocol is an enum for transport protocols.
type Protocol int

const (
	ProtocolUnknown Protocol = iota
	ProtocolTCP
	ProtocolHTTP
	ProtocolHTTPS
	ProtocolGRPC
	ProtocolTLS
	ProtocolWebSocket
)

var protocolStrings = map[Protocol]string{
	ProtocolTCP:       "tcp",
	ProtocolHTTP:      "http",
	ProtocolHTTPS:     "https",
	ProtocolGRPC:      "grpc",
	ProtocolTLS:       "tls",
	ProtocolWebSocket: "websocket",
}

var stringToProtocol = map[string]Protocol{
	"tcp":       ProtocolTCP,
	"http":      ProtocolHTTP,
	"https":     ProtocolHTTPS,
	"grpc":      ProtocolGRPC,
	"tls":       ProtocolTLS,
	"websocket": ProtocolWebSocket,
}

// String returns the string representation.
func (p Protocol) String() string {
	if s, ok := protocolStrings[p]; ok {
		return s
	}
	return "unknown"
}

// ParseProtocol parses a string to Protocol.
func ParseProtocol(s string) (Protocol, error) {
	if proto, ok := stringToProtocol[s]; ok {
		return proto, nil
	}
	return ProtocolUnknown, fmt.Errorf("invalid protocol: %s", s)
}

// IsValid returns true if the protocol is known/valid.
func (p Protocol) IsValid() bool {
	_, ok := protocolStrings[p]
	return ok
}

// IsSecure returns true if the protocol provides encryption/security.
func (p Protocol) IsSecure() bool {
	switch p {
	case ProtocolHTTPS, ProtocolTLS:
		return true
	default:
		return false
	}
}

// DefaultPort returns the default port for the protocol.
func (p Protocol) DefaultPort() int {
	switch p {
	case ProtocolHTTP:
		return 80
	case ProtocolHTTPS:
		return 443
	case ProtocolGRPC:
		return 9090 // Common gRPC port
	default:
		return 0 // No default port
	}
}