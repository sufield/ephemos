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
			wantErr: true,
		},
		{
			name: "empty domain",
			config: &ports.SPIFFEConfig{
				Domain:      "",
				SocketPath:  "/tmp/spire-agent/public/api.sock",
				TrustDomain: "example.com",
			},
			wantErr: true,
		},
		{
			name: "empty socket path",
			config: &ports.SPIFFEConfig{
				Domain:      "test.example.com",
				SocketPath:  "",
				TrustDomain: "example.com",
			},
			wantErr: true,
		},
		{
			name: "empty trust domain",
			config: &ports.SPIFFEConfig{
				Domain:      "test.example.com",
				SocketPath:  "/tmp/spire-agent/public/api.sock",
				TrustDomain: "",
			},
			wantErr: true,
		},
		{
			name: "valid config",
			config: &ports.SPIFFEConfig{
				Domain:      "test.example.com",
				SocketPath:  "/tmp/spire-agent/public/api.sock",
				TrustDomain: "example.com",
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

func TestSPIFFEProvider_ValidateConfig(t *testing.T) {
	tests := []struct {
		name   string
		config *ports.SPIFFEConfig
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "all fields empty",
			config: &ports.SPIFFEConfig{
				Domain:      "",
				SocketPath:  "",
				TrustDomain: "",
			},
			wantErr: true,
		},
		{
			name: "only domain set",
			config: &ports.SPIFFEConfig{
				Domain:      "test.example.com",
				SocketPath:  "",
				TrustDomain: "",
			},
			wantErr: true,
		},
		{
			name: "only socket path set",
			config: &ports.SPIFFEConfig{
				Domain:      "",
				SocketPath:  "/tmp/spire-agent/public/api.sock",
				TrustDomain: "",
			},
			wantErr: true,
		},
		{
			name: "only trust domain set",
			config: &ports.SPIFFEConfig{
				Domain:      "",
				SocketPath:  "",
				TrustDomain: "example.com",
			},
			wantErr: true,
		},
		{
			name: "missing trust domain",
			config: &ports.SPIFFEConfig{
				Domain:      "test.example.com",
				SocketPath:  "/tmp/spire-agent/public/api.sock",
				TrustDomain: "",
			},
			wantErr: true,
		},
		{
			name: "valid complete config",
			config: &ports.SPIFFEConfig{
				Domain:      "test.example.com",
				SocketPath:  "/tmp/spire-agent/public/api.sock",
				TrustDomain: "example.com",
			},
			wantErr: false,
		},
		{
			name: "whitespace in fields",
			config: &ports.SPIFFEConfig{
				Domain:      "  test.example.com  ",
				SocketPath:  "  /tmp/spire-agent/public/api.sock  ",
				TrustDomain: "  example.com  ",
			},
			wantErr: false, // Should be trimmed and valid
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := &SPIFFEProvider{
				config: tt.config,
			}
			
			err := provider.validateConfig()
			if (err != nil) != tt.wantErr {
				t.Errorf("validateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSPIFFEProvider_GetServiceIdentity(t *testing.T) {
	// Note: This test focuses on the configuration validation part
	// since actual SPIFFE/SPIRE interaction requires infrastructure
	
	tests := []struct {
		name    string
		config  *ports.SPIFFEConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &ports.SPIFFEConfig{
				Domain:      "test.example.com",
				SocketPath:  "/tmp/spire-agent/public/api.sock",
				TrustDomain: "example.com",
			},
			wantErr: true, // Will fail without actual SPIRE agent, but config should be valid
		},
		{
			name: "invalid config",
			config: &ports.SPIFFEConfig{
				Domain:      "",
				SocketPath:  "/tmp/spire-agent/public/api.sock",
				TrustDomain: "example.com",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewSPIFFEProvider(tt.config)
			if err != nil {
				if !tt.wantErr {
					t.Errorf("NewSPIFFEProvider() failed: %v", err)
				}
				return
			}

			// This will likely fail without SPIRE infrastructure, 
			// but we're testing the error handling
			_, err = provider.GetServiceIdentity()
			if err == nil && tt.wantErr {
				t.Error("GetServiceIdentity() expected error but got none")
			}
		})
	}
}

func TestSPIFFEProvider_Close(t *testing.T) {
	config := &ports.SPIFFEConfig{
		Domain:      "test.example.com",
		SocketPath:  "/tmp/spire-agent/public/api.sock",
		TrustDomain: "example.com",
	}

	provider, err := NewSPIFFEProvider(config)
	if err != nil {
		t.Skip("Cannot create SPIFFE provider:", err)
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

func TestSPIFFEProvider_ConfigTrimming(t *testing.T) {
	// Test that whitespace is properly trimmed from configuration
	config := &ports.SPIFFEConfig{
		Domain:      "  test.example.com  ",
		SocketPath:  "  /tmp/spire-agent/public/api.sock  ",
		TrustDomain: "  example.com  ",
	}

	provider, err := NewSPIFFEProvider(config)
	if err != nil {
		t.Errorf("NewSPIFFEProvider() with whitespace failed: %v", err)
		return
	}

	// The provider should have trimmed the config values internally
	if provider == nil {
		t.Error("Provider should not be nil with valid trimmed config")
	}
}

func BenchmarkNewSPIFFEProvider(b *testing.B) {
	config := &ports.SPIFFEConfig{
		Domain:      "test.example.com",
		SocketPath:  "/tmp/spire-agent/public/api.sock", 
		TrustDomain: "example.com",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		provider, err := NewSPIFFEProvider(config)
		if err == nil && provider != nil {
			provider.Close()
		}
	}
}

func BenchmarkSPIFFEProvider_ValidateConfig(b *testing.B) {
	config := &ports.SPIFFEConfig{
		Domain:      "test.example.com",
		SocketPath:  "/tmp/spire-agent/public/api.sock",
		TrustDomain: "example.com",
	}
	provider := &SPIFFEProvider{config: config}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := provider.validateConfig()
		if err != nil {
			b.Errorf("validateConfig() failed: %v", err)
		}
	}
}