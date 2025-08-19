package domain

import "testing"

func TestProtocol_String(t *testing.T) {
	tests := []struct {
		protocol Protocol
		expected string
	}{
		{ProtocolTCP, "tcp"},
		{ProtocolHTTP, "http"},
		{ProtocolHTTPS, "https"},
		{ProtocolGRPC, "grpc"},
		{ProtocolTLS, "tls"},
		{ProtocolWebSocket, "websocket"},
		{ProtocolUnknown, "unknown"},
		{Protocol(999), "unknown"}, // Invalid protocol
	}

	for _, tt := range tests {
		if got := tt.protocol.String(); got != tt.expected {
			t.Errorf("Protocol.String() = %v, want %v", got, tt.expected)
		}
	}
}

func TestParseProtocol(t *testing.T) {
	tests := []struct {
		input    string
		expected Protocol
		wantErr  bool
	}{
		{"tcp", ProtocolTCP, false},
		{"http", ProtocolHTTP, false},
		{"https", ProtocolHTTPS, false},
		{"grpc", ProtocolGRPC, false},
		{"tls", ProtocolTLS, false},
		{"websocket", ProtocolWebSocket, false},
		{"invalid", ProtocolUnknown, true},
		{"", ProtocolUnknown, true},
		{"HTTP", ProtocolUnknown, true}, // Case sensitive
	}

	for _, tt := range tests {
		got, err := ParseProtocol(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseProtocol(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}
		if got != tt.expected {
			t.Errorf("ParseProtocol(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestProtocol_IsValid(t *testing.T) {
	tests := []struct {
		protocol Protocol
		expected bool
	}{
		{ProtocolTCP, true},
		{ProtocolHTTP, true},
		{ProtocolHTTPS, true},
		{ProtocolGRPC, true},
		{ProtocolTLS, true},
		{ProtocolWebSocket, true},
		{ProtocolUnknown, false},
		{Protocol(999), false},
	}

	for _, tt := range tests {
		if got := tt.protocol.IsValid(); got != tt.expected {
			t.Errorf("Protocol.IsValid() = %v, want %v for protocol %v", got, tt.expected, tt.protocol)
		}
	}
}

func TestProtocol_IsSecure(t *testing.T) {
	tests := []struct {
		protocol Protocol
		expected bool
	}{
		{ProtocolTCP, false},
		{ProtocolHTTP, false},
		{ProtocolHTTPS, true},
		{ProtocolGRPC, false},
		{ProtocolTLS, true},
		{ProtocolWebSocket, false},
		{ProtocolUnknown, false},
	}

	for _, tt := range tests {
		if got := tt.protocol.IsSecure(); got != tt.expected {
			t.Errorf("Protocol.IsSecure() = %v, want %v for protocol %v", got, tt.expected, tt.protocol)
		}
	}
}

func TestProtocol_DefaultPort(t *testing.T) {
	tests := []struct {
		protocol Protocol
		expected int
	}{
		{ProtocolHTTP, 80},
		{ProtocolHTTPS, 443},
		{ProtocolGRPC, 9090},
		{ProtocolTCP, 0},
		{ProtocolTLS, 0},
		{ProtocolWebSocket, 0},
		{ProtocolUnknown, 0},
	}

	for _, tt := range tests {
		if got := tt.protocol.DefaultPort(); got != tt.expected {
			t.Errorf("Protocol.DefaultPort() = %v, want %v for protocol %v", got, tt.expected, tt.protocol)
		}
	}
}
