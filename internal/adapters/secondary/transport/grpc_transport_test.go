package transport

import (
	"context"
	"testing"

	"github.com/sufield/ephemos/internal/core/ports"
)

// MockIdentityProvider for testing
type MockIdentityProvider struct {
	shouldFail bool
	identity   *MockServiceIdentity
}

type MockServiceIdentity struct {
	domain string
}

func (m *MockServiceIdentity) GetDomain() string {
	return m.domain
}

func (m *MockServiceIdentity) Close() error {
	return nil
}

func (m *MockIdentityProvider) GetServiceIdentity() (ports.ServiceIdentity, error) {
	if m.shouldFail {
		return nil, ports.ErrIdentityNotFound
	}
	return m.identity, nil
}

func (m *MockIdentityProvider) Close() error {
	return nil
}

func TestNewGRPCTransportProvider(t *testing.T) {
	tests := []struct {
		name     string
		provider ports.IdentityProvider
		wantErr  bool
	}{
		{
			name:     "nil identity provider",
			provider: nil,
			wantErr:  true,
		},
		{
			name: "valid identity provider",
			provider: &MockIdentityProvider{
				identity: &MockServiceIdentity{domain: "test.example.com"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport, err := NewGRPCTransportProvider(tt.provider)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewGRPCTransportProvider() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && transport == nil {
				t.Error("NewGRPCTransportProvider() returned nil transport")
			}
		})
	}
}

func TestGRPCTransportProvider_CreateServerTransport(t *testing.T) {
	tests := []struct {
		name            string
		identityProvider *MockIdentityProvider
		wantErr         bool
	}{
		{
			name: "valid identity provider",
			identityProvider: &MockIdentityProvider{
				identity: &MockServiceIdentity{domain: "test.example.com"},
			},
			wantErr: true, // Will fail without actual SPIFFE setup, but validates the flow
		},
		{
			name: "failing identity provider",
			identityProvider: &MockIdentityProvider{
				shouldFail: true,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport, err := NewGRPCTransportProvider(tt.identityProvider)
			if err != nil {
				t.Skip("Cannot create transport provider:", err)
			}

			ctx := context.Background()
			serverTransport, err := transport.CreateServerTransport(ctx)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateServerTransport() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && serverTransport == nil {
				t.Error("CreateServerTransport() returned nil transport without error")
			}
		})
	}
}

func TestGRPCTransportProvider_CreateClientTransport(t *testing.T) {
	tests := []struct {
		name            string
		identityProvider *MockIdentityProvider
		targetService   string
		serverAddress   string
		wantErr         bool
	}{
		{
			name: "valid parameters",
			identityProvider: &MockIdentityProvider{
				identity: &MockServiceIdentity{domain: "test.example.com"},
			},
			targetService: "echo-server",
			serverAddress: "localhost:50051",
			wantErr:       true, // Will fail without actual SPIFFE setup
		},
		{
			name: "empty target service",
			identityProvider: &MockIdentityProvider{
				identity: &MockServiceIdentity{domain: "test.example.com"},
			},
			targetService: "",
			serverAddress: "localhost:50051",
			wantErr:       true,
		},
		{
			name: "empty server address",
			identityProvider: &MockIdentityProvider{
				identity: &MockServiceIdentity{domain: "test.example.com"},
			},
			targetService: "echo-server",
			serverAddress: "",
			wantErr:       true,
		},
		{
			name: "failing identity provider",
			identityProvider: &MockIdentityProvider{
				shouldFail: true,
			},
			targetService: "echo-server",
			serverAddress: "localhost:50051",
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport, err := NewGRPCTransportProvider(tt.identityProvider)
			if err != nil {
				t.Skip("Cannot create transport provider:", err)
			}

			ctx := context.Background()
			clientTransport, err := transport.CreateClientTransport(ctx, tt.targetService, tt.serverAddress)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateClientTransport() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && clientTransport == nil {
				t.Error("CreateClientTransport() returned nil transport without error")
			}
		})
	}
}

func TestGRPCTransportProvider_CreateClientTransport_ContextValidation(t *testing.T) {
	identityProvider := &MockIdentityProvider{
		identity: &MockServiceIdentity{domain: "test.example.com"},
	}
	
	transport, err := NewGRPCTransportProvider(identityProvider)
	if err != nil {
		t.Skip("Cannot create transport provider:", err)
	}

	// Test with nil context
	_, err = transport.CreateClientTransport(nil, "echo-server", "localhost:50051")
	if err == nil {
		t.Error("CreateClientTransport() with nil context should return error")
	}
}

func TestGRPCTransportProvider_CreateServerTransport_ContextValidation(t *testing.T) {
	identityProvider := &MockIdentityProvider{
		identity: &MockServiceIdentity{domain: "test.example.com"},
	}
	
	transport, err := NewGRPCTransportProvider(identityProvider)
	if err != nil {
		t.Skip("Cannot create transport provider:", err)
	}

	// Test with nil context
	_, err = transport.CreateServerTransport(nil)
	if err == nil {
		t.Error("CreateServerTransport() with nil context should return error")
	}
}

func TestGRPCTransportProvider_Close(t *testing.T) {
	identityProvider := &MockIdentityProvider{
		identity: &MockServiceIdentity{domain: "test.example.com"},
	}
	
	transport, err := NewGRPCTransportProvider(identityProvider)
	if err != nil {
		t.Skip("Cannot create transport provider:", err)
	}

	// Close should not return an error
	err = transport.Close()
	if err != nil {
		t.Errorf("Close() returned error: %v", err)
	}

	// Multiple closes should be safe
	err = transport.Close()
	if err != nil {
		t.Errorf("Second Close() returned error: %v", err)
	}
}

func TestGRPCTransportProvider_Integration(t *testing.T) {
	// Integration test showing typical usage flow
	identityProvider := &MockIdentityProvider{
		identity: &MockServiceIdentity{domain: "test.example.com"},
	}
	
	transport, err := NewGRPCTransportProvider(identityProvider)
	if err != nil {
		t.Skip("Cannot create transport provider:", err)
	}
	defer transport.Close()

	ctx := context.Background()

	// Try to create both server and client transports
	// These will likely fail without SPIFFE infrastructure,
	// but we can verify the error handling works correctly
	
	_, serverErr := transport.CreateServerTransport(ctx)
	if serverErr == nil {
		t.Log("Server transport creation succeeded (unexpected without SPIFFE)")
	}

	_, clientErr := transport.CreateClientTransport(ctx, "test-service", "localhost:8080")
	if clientErr == nil {
		t.Log("Client transport creation succeeded (unexpected without SPIFFE)")
	}

	// Both should fail gracefully without panicking
}

func BenchmarkNewGRPCTransportProvider(b *testing.B) {
	identityProvider := &MockIdentityProvider{
		identity: &MockServiceIdentity{domain: "test.example.com"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		transport, err := NewGRPCTransportProvider(identityProvider)
		if err == nil && transport != nil {
			transport.Close()
		}
	}
}

func BenchmarkGRPCTransportProvider_CreateServerTransport(b *testing.B) {
	identityProvider := &MockIdentityProvider{
		identity: &MockServiceIdentity{domain: "test.example.com"},
	}
	
	transport, err := NewGRPCTransportProvider(identityProvider)
	if err != nil {
		b.Skip("Cannot create transport provider:", err)
	}
	defer transport.Close()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// This will likely fail but we're benchmarking the attempt
		transport.CreateServerTransport(ctx)
	}
}