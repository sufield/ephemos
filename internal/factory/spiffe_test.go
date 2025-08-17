package factory

import (
	"context"
	"net"
	"net/http"
	"testing"

	"github.com/sufield/ephemos/internal/core/ports"
)

// mockIdentityProvider implements a test identity provider
type mockIdentityProvider struct {
	closeFunc func() error
}

func (m *mockIdentityProvider) GetDomain() string {
	return "test.domain"
}

func (m *mockIdentityProvider) GetName() string {
	return "test-service"
}

func (m *mockIdentityProvider) Validate() error {
	return nil
}

func (m *mockIdentityProvider) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

// mockConn implements a test connection
type mockConn struct {
	httpClientFunc func() (*http.Client, error)
	closeFunc      func() error
}

func (m *mockConn) HTTPClient() (*http.Client, error) {
	if m.httpClientFunc != nil {
		return m.httpClientFunc()
	}
	return &http.Client{}, nil
}

func (m *mockConn) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

// mockDialer implements a test dialer
type mockDialer struct {
	connectFunc func(context.Context, string, string) (ports.Conn, error)
	closeFunc   func() error
}

func (m *mockDialer) Connect(ctx context.Context, serviceName, address string) (ports.Conn, error) {
	if m.connectFunc != nil {
		return m.connectFunc(ctx, serviceName, address)
	}
	return &mockConn{}, nil
}

func (m *mockDialer) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

// mockServer implements a test server
type mockServer struct {
	serveFunc func(context.Context, net.Listener) error
	closeFunc func() error
	addrFunc  func() net.Addr
}

func (m *mockServer) Serve(ctx context.Context, lis net.Listener) error {
	if m.serveFunc != nil {
		return m.serveFunc(ctx, lis)
	}
	return nil
}

func (m *mockServer) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

func (m *mockServer) Addr() net.Addr {
	if m.addrFunc != nil {
		return m.addrFunc()
	}
	return nil
}

func TestSPIFFEDialer(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		config  *ports.Configuration
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil configuration",
			config:  nil,
			wantErr: true,
			errMsg:  "configuration cannot be nil",
		},
		{
			name: "invalid configuration",
			config: &ports.Configuration{
				Service: ports.ServiceConfig{
					Name:   "", // Invalid: empty name
					Domain: "test.domain",
				},
			},
			wantErr: true,
			errMsg:  "invalid configuration",
		},
		{
			name: "valid configuration",
			config: &ports.Configuration{
				Service: ports.ServiceConfig{
					Name:   "test-service",
					Domain: "test.domain",
				},
				Agent: &ports.AgentConfig{
					SocketPath: "/tmp/test.sock",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dialer, err := SPIFFEDialer(ctx, tt.config)

			if tt.wantErr {
				if err == nil {
					t.Errorf("SPIFFEDialer() expected error containing %q, got nil", tt.errMsg)
				} else if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("SPIFFEDialer() error = %v, want error containing %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("SPIFFEDialer() unexpected error = %v", err)
				}
				if dialer == nil {
					t.Error("SPIFFEDialer() returned nil dialer without error")
				}
			}
		})
	}
}

func TestSPIFFEServer(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		config  *ports.Configuration
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil configuration",
			config:  nil,
			wantErr: true,
			errMsg:  "configuration cannot be nil",
		},
		{
			name: "invalid configuration",
			config: &ports.Configuration{
				Service: ports.ServiceConfig{
					Name:   "test-service",
					Domain: "", // Invalid: empty domain
				},
			},
			wantErr: true,
			errMsg:  "failed to create",
		},
		{
			name: "valid configuration",
			config: &ports.Configuration{
				Service: ports.ServiceConfig{
					Name:   "test-service",
					Domain: "test.domain",
				},
				Agent: &ports.AgentConfig{
					SocketPath: "/tmp/test.sock",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, err := SPIFFEServer(ctx, tt.config)

			if tt.wantErr {
				if err == nil {
					t.Errorf("SPIFFEServer() expected error containing %q, got nil", tt.errMsg)
				} else if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("SPIFFEServer() error = %v, want error containing %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("SPIFFEServer() unexpected error = %v", err)
				}
				if server == nil {
					t.Error("SPIFFEServer() returned nil server without error")
				}
			}
		})
	}
}

func TestSpiffeDialerAdapter(t *testing.T) {
	t.Run("Connect delegates to internal client", func(t *testing.T) {
		// This test verifies that the adapter properly delegates
		// Since we can't easily mock the internal API client,
		// we'll skip this for now and rely on integration tests
		t.Skip("Integration test required")
	})

	t.Run("Close delegates to internal client", func(t *testing.T) {
		// This test verifies that the adapter properly delegates
		// Since we can't easily mock the internal API client,
		// we'll skip this for now and rely on integration tests
		t.Skip("Integration test required")
	})
}

func TestSpiffeConnAdapter(t *testing.T) {
	t.Run("HTTPClient delegates to internal connection", func(t *testing.T) {
		// This test verifies that the adapter properly delegates
		// Since we can't easily mock the internal API connection,
		// we'll skip this for now and rely on integration tests
		t.Skip("Integration test required")
	})

	t.Run("Close delegates to internal connection", func(t *testing.T) {
		// This test verifies that the adapter properly delegates
		// Since we can't easily mock the internal API connection,
		// we'll skip this for now and rely on integration tests
		t.Skip("Integration test required")
	})
}

func TestSpiffeServerAdapter(t *testing.T) {
	t.Run("Serve delegates to internal server", func(t *testing.T) {
		// This test verifies that the adapter properly delegates
		// Since we can't easily mock the internal API server,
		// we'll skip this for now and rely on integration tests
		t.Skip("Integration test required")
	})

	t.Run("Close delegates to internal server", func(t *testing.T) {
		// This test verifies that the adapter properly delegates
		// Since we can't easily mock the internal API server,
		// we'll skip this for now and rely on integration tests
		t.Skip("Integration test required")
	})

	t.Run("Addr returns nil", func(t *testing.T) {
		adapter := &spiffeServerAdapter{}
		if addr := adapter.Addr(); addr != nil {
			t.Errorf("Addr() = %v, want nil", addr)
		}
	})
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && len(substr) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
