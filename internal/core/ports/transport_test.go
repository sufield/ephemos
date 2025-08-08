package ports

import (
	"context"
	"errors"
	"testing"
)

// Mock implementation of ServerTransport for testing
type mockServerTransport struct {
	shouldFailServe bool
	serveCalled     bool
	closeCalled     bool
}

func (m *mockServerTransport) Serve(ctx context.Context, listener interface{}) error {
	m.serveCalled = true
	if m.shouldFailServe {
		return ErrTransportCreationFailed
	}
	return nil
}

func (m *mockServerTransport) Close() error {
	m.closeCalled = true
	return nil
}

// Mock implementation of ClientTransport for testing
type mockClientTransport struct {
	shouldFailConnect bool
	connectCalled     bool
	closeCalled       bool
}

func (m *mockClientTransport) Connect(ctx context.Context, serviceName, address string) (interface{}, error) {
	m.connectCalled = true
	if m.shouldFailConnect {
		return nil, ErrTransportCreationFailed
	}
	return &mockConnection{}, nil
}

func (m *mockClientTransport) Close() error {
	m.closeCalled = true
	return nil
}

// Mock connection for testing
type mockConnection struct{}

// Mock implementation of TransportProvider for testing
type mockTransportProvider struct {
	shouldFailServer bool
	shouldFailClient bool
	closeCalled      bool
}

func (m *mockTransportProvider) CreateServerTransport(ctx context.Context) (ServerTransport, error) {
	if m.shouldFailServer {
		return nil, ErrTransportCreationFailed
	}
	return &mockServerTransport{}, nil
}

func (m *mockTransportProvider) CreateClientTransport(ctx context.Context, targetService, serverAddress string) (ClientTransport, error) {
	if m.shouldFailClient {
		return nil, ErrTransportCreationFailed
	}
	return &mockClientTransport{}, nil
}

func (m *mockTransportProvider) Close() error {
	m.closeCalled = true
	return nil
}

func TestTransportProvider_Interface(t *testing.T) {
	// Test that mock implements the interface correctly
	var provider TransportProvider = &mockTransportProvider{}

	ctx := context.Background()

	// Test CreateServerTransport
	serverTransport, err := provider.CreateServerTransport(ctx)
	if err != nil {
		t.Errorf("CreateServerTransport() failed: %v", err)
	}

	if serverTransport == nil {
		t.Error("CreateServerTransport() returned nil")
	}

	// Test CreateClientTransport
	clientTransport, err := provider.CreateClientTransport(ctx, "test-service", "localhost:8080")
	if err != nil {
		t.Errorf("CreateClientTransport() failed: %v", err)
	}

	if clientTransport == nil {
		t.Error("CreateClientTransport() returned nil")
	}

	// Test Close
	err = provider.Close()
	if err != nil {
		t.Errorf("Close() failed: %v", err)
	}
}

func TestServerTransport_Interface(t *testing.T) {
	// Test that mock implements the interface correctly
	var transport ServerTransport = &mockServerTransport{}

	ctx := context.Background()

	// Test Serve
	err := transport.Serve(ctx, &mockConnection{})
	if err != nil {
		t.Errorf("Serve() failed: %v", err)
	}

	// Test Close
	err = transport.Close()
	if err != nil {
		t.Errorf("Close() failed: %v", err)
	}
}

func TestClientTransport_Interface(t *testing.T) {
	// Test that mock implements the interface correctly
	var transport ClientTransport = &mockClientTransport{}

	ctx := context.Background()

	// Test Connect
	conn, err := transport.Connect(ctx, "test-service", "localhost:8080")
	if err != nil {
		t.Errorf("Connect() failed: %v", err)
	}

	if conn == nil {
		t.Error("Connect() returned nil connection")
	}

	// Test Close
	err = transport.Close()
	if err != nil {
		t.Errorf("Close() failed: %v", err)
	}
}

func TestTransportProvider_ErrorHandling(t *testing.T) {
	tests := []struct {
		name     string
		provider *mockTransportProvider
		testFunc func(TransportProvider) error
		wantErr  bool
	}{
		{
			name:     "server transport creation success",
			provider: &mockTransportProvider{},
			testFunc: func(p TransportProvider) error {
				_, err := p.CreateServerTransport(context.Background())
				return err
			},
			wantErr: false,
		},
		{
			name:     "server transport creation failure",
			provider: &mockTransportProvider{shouldFailServer: true},
			testFunc: func(p TransportProvider) error {
				_, err := p.CreateServerTransport(context.Background())
				return err
			},
			wantErr: true,
		},
		{
			name:     "client transport creation success",
			provider: &mockTransportProvider{},
			testFunc: func(p TransportProvider) error {
				_, err := p.CreateClientTransport(context.Background(), "test-service", "localhost:8080")
				return err
			},
			wantErr: false,
		},
		{
			name:     "client transport creation failure",
			provider: &mockTransportProvider{shouldFailClient: true},
			testFunc: func(p TransportProvider) error {
				_, err := p.CreateClientTransport(context.Background(), "test-service", "localhost:8080")
				return err
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.testFunc(tt.provider)
			if (err != nil) != tt.wantErr {
				t.Errorf("Test function error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr && !errors.Is(err, ErrTransportCreationFailed) {
				t.Errorf("Expected ErrTransportCreationFailed, got %v", err)
			}
		})
	}
}

func TestServerTransport_Serve(t *testing.T) {
	tests := []struct {
		name      string
		transport *mockServerTransport
		ctx       context.Context
		listener  interface{}
		wantErr   bool
	}{
		{
			name:      "successful serve",
			transport: &mockServerTransport{},
			ctx:       context.Background(),
			listener:  &mockConnection{},
			wantErr:   false,
		},
		{
			name:      "serve failure",
			transport: &mockServerTransport{shouldFailServe: true},
			ctx:       context.Background(),
			listener:  &mockConnection{},
			wantErr:   true,
		},
		{
			name:      "nil context",
			transport: &mockServerTransport{},
			ctx:       nil,
			listener:  &mockConnection{},
			wantErr:   false, // Mock doesn't validate context
		},
		{
			name:      "nil listener",
			transport: &mockServerTransport{},
			ctx:       context.Background(),
			listener:  nil,
			wantErr:   false, // Mock doesn't validate listener
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.transport.Serve(tt.ctx, tt.listener)
			if (err != nil) != tt.wantErr {
				t.Errorf("Serve() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.transport.serveCalled {
				t.Error("Serve() was not called on transport")
			}
		})
	}
}

func TestClientTransport_Connect(t *testing.T) {
	tests := []struct {
		name        string
		transport   *mockClientTransport
		ctx         context.Context
		serviceName string
		address     string
		wantErr     bool
	}{
		{
			name:        "successful connect",
			transport:   &mockClientTransport{},
			ctx:         context.Background(),
			serviceName: "test-service",
			address:     "localhost:8080",
			wantErr:     false,
		},
		{
			name:        "connect failure",
			transport:   &mockClientTransport{shouldFailConnect: true},
			ctx:         context.Background(),
			serviceName: "test-service",
			address:     "localhost:8080",
			wantErr:     true,
		},
		{
			name:        "empty service name",
			transport:   &mockClientTransport{},
			ctx:         context.Background(),
			serviceName: "",
			address:     "localhost:8080",
			wantErr:     false, // Mock doesn't validate
		},
		{
			name:        "empty address",
			transport:   &mockClientTransport{},
			ctx:         context.Background(),
			serviceName: "test-service",
			address:     "",
			wantErr:     false, // Mock doesn't validate
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn, err := tt.transport.Connect(tt.ctx, tt.serviceName, tt.address)
			if (err != nil) != tt.wantErr {
				t.Errorf("Connect() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && conn == nil {
				t.Error("Connect() returned nil connection")
			}

			if !tt.transport.connectCalled {
				t.Error("Connect() was not called on transport")
			}
		})
	}
}

func TestTransportProvider_Lifecycle(t *testing.T) {
	// Test complete lifecycle
	provider := &mockTransportProvider{}
	ctx := context.Background()

	// Create server transport
	serverTransport, err := provider.CreateServerTransport(ctx)
	if err != nil {
		t.Errorf("CreateServerTransport() failed: %v", err)
	}

	// Use server transport
	err = serverTransport.Serve(ctx, &mockConnection{})
	if err != nil {
		t.Errorf("Serve() failed: %v", err)
	}

	// Close server transport
	err = serverTransport.Close()
	if err != nil {
		t.Errorf("ServerTransport Close() failed: %v", err)
	}

	// Create client transport
	clientTransport, err := provider.CreateClientTransport(ctx, "test-service", "localhost:8080")
	if err != nil {
		t.Errorf("CreateClientTransport() failed: %v", err)
	}

	// Use client transport
	conn, err := clientTransport.Connect(ctx, "test-service", "localhost:8080")
	if err != nil {
		t.Errorf("Connect() failed: %v", err)
	}

	if conn == nil {
		t.Error("Connect() returned nil connection")
	}

	// Close client transport
	err = clientTransport.Close()
	if err != nil {
		t.Errorf("ClientTransport Close() failed: %v", err)
	}

	// Close provider
	err = provider.Close()
	if err != nil {
		t.Errorf("Provider Close() failed: %v", err)
	}

	if !provider.closeCalled {
		t.Error("Close() was not called on provider")
	}
}

func TestErrTransportCreationFailed(t *testing.T) {
	// Test the standard error
	if ErrTransportCreationFailed == nil {
		t.Error("ErrTransportCreationFailed should not be nil")
	}

	expectedMsg := "transport creation failed"
	if ErrTransportCreationFailed.Error() != expectedMsg {
		t.Errorf("ErrTransportCreationFailed.Error() = %v, want %v", ErrTransportCreationFailed.Error(), expectedMsg)
	}

	// Test error comparison
	provider := &mockTransportProvider{shouldFailServer: true}
	_, err := provider.CreateServerTransport(context.Background())

	if !errors.Is(err, ErrTransportCreationFailed) {
		t.Error("Error should be ErrTransportCreationFailed")
	}
}

func TestTransportProvider_Concurrent(t *testing.T) {
	// Test concurrent access to transport provider
	provider := &mockTransportProvider{}
	ctx := context.Background()

	done := make(chan bool, 10)

	// Start multiple goroutines
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()

			for j := 0; j < 50; j++ {
				// Alternate between server and client creation
				if j%2 == 0 {
					transport, err := provider.CreateServerTransport(ctx)
					if err != nil {
						t.Errorf("CreateServerTransport() failed: %v", err)
						return
					}
					transport.Close()
				} else {
					transport, err := provider.CreateClientTransport(ctx, "test-service", "localhost:8080")
					if err != nil {
						t.Errorf("CreateClientTransport() failed: %v", err)
						return
					}
					transport.Close()
				}
			}
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

func BenchmarkTransportProvider_CreateServerTransport(b *testing.B) {
	provider := &mockTransportProvider{}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		transport, err := provider.CreateServerTransport(ctx)
		if err != nil {
			b.Errorf("CreateServerTransport() failed: %v", err)
		}
		if transport != nil {
			transport.Close()
		}
	}
}

func BenchmarkTransportProvider_CreateClientTransport(b *testing.B) {
	provider := &mockTransportProvider{}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		transport, err := provider.CreateClientTransport(ctx, "test-service", "localhost:8080")
		if err != nil {
			b.Errorf("CreateClientTransport() failed: %v", err)
		}
		if transport != nil {
			transport.Close()
		}
	}
}

func BenchmarkServerTransport_Serve(b *testing.B) {
	transport := &mockServerTransport{}
	ctx := context.Background()
	listener := &mockConnection{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := transport.Serve(ctx, listener)
		if err != nil {
			b.Errorf("Serve() failed: %v", err)
		}
	}
}

func BenchmarkClientTransport_Connect(b *testing.B) {
	transport := &mockClientTransport{}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		conn, err := transport.Connect(ctx, "test-service", "localhost:8080")
		if err != nil {
			b.Errorf("Connect() failed: %v", err)
		}
		if conn == nil {
			b.Error("Connect() returned nil")
		}
	}
}