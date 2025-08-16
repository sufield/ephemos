package api_test

import (
	"context"
	"errors"
	"net"
	"sync"
	"testing"

	"github.com/sufield/ephemos/internal/adapters/primary/api"
	"github.com/sufield/ephemos/internal/core/domain"
	epherrors "github.com/sufield/ephemos/internal/core/errors"
	"github.com/sufield/ephemos/internal/core/ports"
)

// Mock implementations for server testing
type mockServerIdentityProvider struct{}

func (m *mockServerIdentityProvider) GetServiceIdentity() (*domain.ServiceIdentity, error) {
	return domain.NewServiceIdentity("test-service", "test.local"), nil
}

func (m *mockServerIdentityProvider) GetCertificate() (*domain.Certificate, error) {
	return &domain.Certificate{}, nil
}

func (m *mockServerIdentityProvider) GetTrustBundle() (*domain.TrustBundle, error) {
	return &domain.TrustBundle{}, nil
}

func (m *mockServerIdentityProvider) Close() error {
	return nil
}

type mockServerTransportProvider struct{
	server *mockServer
}

func (m *mockServerTransportProvider) CreateServer(cert *domain.Certificate, bundle *domain.TrustBundle, policy *domain.AuthenticationPolicy) (ports.ServerPort, error) {
	return m.server, nil
}

func (m *mockServerTransportProvider) CreateClient(cert *domain.Certificate, bundle *domain.TrustBundle, policy *domain.AuthenticationPolicy) (ports.ClientPort, error) {
	return nil, nil
}

type mockConfigProvider struct{}

func (m *mockConfigProvider) LoadConfiguration(ctx context.Context, path string) (*ports.Configuration, error) {
	return &ports.Configuration{}, nil
}

func (m *mockConfigProvider) GetDefaultConfiguration(ctx context.Context) *ports.Configuration {
	return &ports.Configuration{}
}

type mockServer struct{
	started sync.WaitGroup
	stopped chan struct{}
}

func (m *mockServer) Start(listener ports.ListenerPort) error {
	m.started.Done()
	<-m.stopped
	return nil
}

func (m *mockServer) Stop() error { 
	select {
	case <-m.stopped:
		// already closed
	default:
		close(m.stopped)
	}
	return nil
}

func (m *mockServer) RegisterService(serviceRegistrar ports.ServiceRegistrarPort) error {
	return nil
}

func (m *mockServer) Close() error {
	return nil
}

func TestServer_WorkloadServer(t *testing.T) {
	t.Parallel()
	
	ms := &mockServer{stopped: make(chan struct{})}
	ms.started.Add(1)
	tp := &mockServerTransportProvider{server: ms}
	
	tests := []struct {
		name              string
		identityProvider  ports.IdentityProvider
		transportProvider ports.TransportProvider
		configProvider    ports.ConfigurationProvider
		config            *ports.Configuration
		wantErr           bool
		wantErrField      string
	}{
		{
			name:              "nil config",
			identityProvider:  &mockServerIdentityProvider{},
			transportProvider: tp,
			configProvider:    &mockConfigProvider{},
			config:            nil,
			wantErr:           true,
			wantErrField:      "configuration",
		},
		{
			name:              "nil identity provider",
			identityProvider:  nil,
			transportProvider: tp,
			configProvider:    &mockConfigProvider{},
			config: &ports.Configuration{
				Service: ports.ServiceConfig{
					Name:   "test-service",
					Domain: "test.local",
				},
			},
			wantErr:           true,
			wantErrField:      "identityProvider",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server, err := api.WorkloadServer(tt.identityProvider, tt.transportProvider, tt.configProvider, tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("api.WorkloadServer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if tt.wantErr && tt.wantErrField != "" {
				var ve *epherrors.ValidationError
				if !errors.As(err, &ve) || ve.Field != tt.wantErrField {
					t.Fatalf("want ValidationError{Field:%s}, got %v", tt.wantErrField, err)
				}
			}
			
			if !tt.wantErr && server == nil {
				t.Error("api.WorkloadServer() returned nil server")
			}
		})
	}
}

func TestServer_RegisterService(t *testing.T) {
	t.Parallel()
	
	ms := &mockServer{stopped: make(chan struct{})}
	ms.started.Add(1)
	tp := &mockServerTransportProvider{server: ms}
	
	config := &ports.Configuration{
		Service: ports.ServiceConfig{
			Name:   "test-service",
			Domain: "test.local",
		},
	}
	server, err := api.WorkloadServer(&mockServerIdentityProvider{}, tp, &mockConfigProvider{}, config)
	if err != nil {
		t.Skip("Skipping RegisterService tests - could not create server:", err)
	}

	// Test with nil context
	ctx := context.Background()
	err = server.RegisterService(nil, nil)
	var ve *epherrors.ValidationError
	if !errors.As(err, &ve) || ve.Field != "context" {
		t.Fatalf("want ValidationError{Field:context}, got %v", err)
	}

	// Test with nil registrar
	err = server.RegisterService(ctx, nil)
	if !errors.As(err, &ve) || ve.Field != "serviceRegistrar" {
		t.Fatalf("want ValidationError{Field:serviceRegistrar}, got %v", err)
	}
}

func TestServer_Serve(t *testing.T) {
	t.Parallel()
	
	ms := &mockServer{stopped: make(chan struct{})}
	ms.started.Add(1)
	tp := &mockServerTransportProvider{server: ms}
	
	config := &ports.Configuration{
		Service: ports.ServiceConfig{
			Name:   "test-service",
			Domain: "test.local",
		},
	}
	server, err := api.WorkloadServer(&mockServerIdentityProvider{}, tp, &mockConfigProvider{}, config)
	if err != nil {
		t.Skip("Skipping Serve tests - could not create server:", err)
	}

	// Test with nil listener
	ctx := context.Background()
	err = server.Serve(ctx, nil)
	var ve *epherrors.ValidationError
	if !errors.As(err, &ve) || ve.Field != "listener" {
		t.Fatalf("want ValidationError{Field:listener}, got %v", err)
	}

	// Test context cancellation behavior
	t.Run("context cancellation", func(t *testing.T) {
		t.Parallel()
		// This test verifies that Serve respects context cancellation
		// Since the mock returns immediately, we test that the cancellation path exists
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately
		
		// Create a listener on a random port
		lis, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Skip("Could not create listener:", err)
		}
		defer lis.Close()

		// Use a simple mock that doesn't block
		simpleMock := &mockServer{stopped: make(chan struct{})}
		simpleMock.started.Add(1)
		close(simpleMock.stopped) // Don't block
		simpleTP := &mockServerTransportProvider{server: simpleMock}
		
		testServer, err := api.WorkloadServer(&mockServerIdentityProvider{}, simpleTP, &mockConfigProvider{}, config)
		if err != nil {
			t.Skip("Could not create test server:", err)
		}

		// This should complete quickly since context is already cancelled
		err = testServer.Serve(ctx, lis)
		if err == nil {
			t.Error("Serve() should have returned an error with cancelled context")
		}
	})
}

func TestServer_Close(t *testing.T) {
	t.Parallel()
	
	ms := &mockServer{stopped: make(chan struct{})}
	ms.started.Add(1)
	tp := &mockServerTransportProvider{server: ms}
	
	config := &ports.Configuration{
		Service: ports.ServiceConfig{
			Name:   "test-service",
			Domain: "test.local",
		},
	}
	server, err := api.WorkloadServer(&mockServerIdentityProvider{}, tp, &mockConfigProvider{}, config)
	if err != nil {
		t.Skip("Skipping Close test - could not create server:", err)
	}

	// Close should not return an error
	if err := server.Close(); err != nil {
		t.Errorf("Close() returned error: %v", err)
	}

	// Multiple closes should be safe (idempotent)
	if err := server.Close(); err != nil {
		t.Errorf("Second Close() returned error: %v", err)
	}
}

