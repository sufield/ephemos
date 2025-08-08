package spiffe

import (
	"testing"

	"github.com/sufield/ephemos/internal/core/ports"
)

func TestNewSPIFFEProvider(t *testing.T) {
	tests := []struct {
		name   string
		config *ports.SPIFFEConfig
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: false, // Uses default path
		},
		{
			name: "empty socket path",
			config: &ports.SPIFFEConfig{
				SocketPath: "",
			},
			wantErr: false, // Constructor doesn't validate, just sets the path
		},
		{
			name: "valid config",
			config: &ports.SPIFFEConfig{
				SocketPath: "/tmp/spire-agent/public/api.sock",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewSPIFFEProvider(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewSPIFFEProvider() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && provider == nil {
				t.Error("NewSPIFFEProvider() returned nil provider")
			}
		})
	}
}

func TestSPIFFEProvider_SocketPath(t *testing.T) {
	tests := []struct {
		name       string
		config     *ports.SPIFFEConfig
		expectPath string
	}{
		{
			name:       "nil config uses default",
			config:     nil,
			expectPath: "/tmp/spire-agent/public/api.sock",
		},
		{
			name: "custom socket path",
			config: &ports.SPIFFEConfig{
				SocketPath: "/custom/path/api.sock",
			},
			expectPath: "/custom/path/api.sock",
		},
		{
			name: "empty socket path",
			config: &ports.SPIFFEConfig{
				SocketPath: "",
			},
			expectPath: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewSPIFFEProvider(tt.config)
			if err != nil {
				t.Errorf("NewSPIFFEProvider() error = %v", err)
				return
			}
			
			if provider.socketPath != tt.expectPath {
				t.Errorf("socketPath = %v, want %v", provider.socketPath, tt.expectPath)
			}
		})
	}
}

func TestSPIFFEProvider_Close(t *testing.T) {
	// Test that Close doesn't panic when called on an uninitialized provider
	provider, err := NewSPIFFEProvider(&ports.SPIFFEConfig{
		SocketPath: "/tmp/spire-agent/public/api.sock",
	})
	if err != nil {
		t.Fatalf("NewSPIFFEProvider() failed: %v", err)
	}

	// Close should not panic and should be safe to call multiple times
	err = provider.Close()
	if err != nil {
		t.Errorf("Close() returned error: %v", err)
	}

	// Second close should also be safe
	err = provider.Close()
	if err != nil {
		t.Errorf("Second Close() returned error: %v", err)
	}
}

func TestSPIFFEProvider_GetX509Source(t *testing.T) {
	provider, err := NewSPIFFEProvider(&ports.SPIFFEConfig{
		SocketPath: "/tmp/spire-agent/public/api.sock",
	})
	if err != nil {
		t.Fatalf("NewSPIFFEProvider() failed: %v", err)
	}

	// Should return nil before any SPIRE operations
	source := provider.GetX509Source()
	if source != nil {
		t.Error("GetX509Source() should return nil before initialization")
	}
}

func TestSPIFFEProvider_SocketPathValidation(t *testing.T) {
	tests := []struct {
		name       string
		socketPath string
		wantErr    bool
	}{
		{
			name:       "absolute unix socket path",
			socketPath: "/tmp/spire-agent/public/api.sock",
			wantErr:    false,
		},
		{
			name:       "relative socket path",
			socketPath: "spire-agent/api.sock",
			wantErr:    false, // Constructor doesn't validate
		},
		{
			name:       "empty socket path",
			socketPath: "",
			wantErr:    false, // Constructor doesn't validate
		},
		{
			name:       "socket path with spaces",
			socketPath: "/tmp/spire agent/api.sock",
			wantErr:    false,
		},
		{
			name:       "root socket path",
			socketPath: "/api.sock",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &ports.SPIFFEConfig{
				SocketPath: tt.socketPath,
			}

			provider, err := NewSPIFFEProvider(config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewSPIFFEProvider() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && provider == nil {
				t.Error("NewSPIFFEProvider() returned nil provider for valid config")
			}
		})
	}
}

func BenchmarkNewSPIFFEProvider(b *testing.B) {
	config := &ports.SPIFFEConfig{
		SocketPath: "/tmp/spire-agent/public/api.sock", 
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		provider, err := NewSPIFFEProvider(config)
		if err == nil && provider != nil {
			provider.Close()
		}
	}
}

// Note: Tests that require actual SPIRE infrastructure are omitted
// as they would timeout in CI/testing environments without SPIRE running.
// These methods (GetServiceIdentity, GetCertificate, GetTrustBundle, GetTLSConfig)
// are integration-tested in environments with SPIRE running.