package ephemos

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/sufield/ephemos/internal/core/ports"
)

// Compile-time interface conformance checks
var (
	_ Client                    = (*clientWrapper)(nil)
	_ Server                    = (*serverWrapper)(nil)
	_ ports.Dialer              = (*mockDialer)(nil)
	_ ports.Conn                = (*mockConn)(nil)
	_ ports.AuthenticatedServer = (*mockServerPort)(nil)
	_ ConfigLoader              = (*mockConfigLoader)(nil)
)

// Mock implementations for testing

type mockDialer struct {
	connectFunc func(ctx context.Context, serviceName, address string) (ports.Conn, error)
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

type mockServerPort struct {
	serveFunc func(ctx context.Context, lis net.Listener) error
	closeFunc func() error
	addrFunc  func() net.Addr
}

func (m *mockServerPort) Serve(ctx context.Context, lis net.Listener) error {
	if m.serveFunc != nil {
		return m.serveFunc(ctx, lis)
	}
	return nil
}

func (m *mockServerPort) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

func (m *mockServerPort) Addr() net.Addr {
	if m.addrFunc != nil {
		return m.addrFunc()
	}
	return nil
}

type mockConfigLoader struct {
	loadFunc func(source string) (*ports.Configuration, error)
}

func (m *mockConfigLoader) LoadConfiguration(source string) (*ports.Configuration, error) {
	if m.loadFunc != nil {
		return m.loadFunc(source)
	}
	return &ports.Configuration{
		Service: ports.ServiceConfig{
			Name:   "test-service",
			Domain: "test.local",
		},
	}, nil
}

// Test interface conformance and basic functionality

func TestClientConformance(t *testing.T) {
	ctx := context.Background()

	// Test with mock dialer
	mockDialer := &mockDialer{}
	client, err := IdentityClient(ctx, WithDialer(mockDialer))
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	// Test Connect
	conn, err := client.Connect(ctx, "localhost:8080")
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	// Test HTTPClient
	httpClient, err := conn.HTTPClient()
	if err != nil {
		t.Fatalf("failed to get HTTP client: %v", err)
	}
	if httpClient == nil {
		t.Fatal("HTTP client should not be nil")
	}
}

func TestServerConformance(t *testing.T) {
	ctx := context.Background()

	// Create a test listener
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("failed to create listener: %v", err)
	}
	defer listener.Close()

	// Test with mock server implementation
	mockServerPort := &mockServerPort{}
	server, err := IdentityServer(ctx, WithServerImpl(mockServerPort), WithListener(listener))
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	defer server.Close()

	// Test Addr
	addr := server.Addr()
	if addr != nil {
		t.Logf("server address: %v", addr)
	}
}

func TestClientClosureIdempotency(t *testing.T) {
	ctx := context.Background()

	mockDialer := &mockDialer{}
	client, err := IdentityClient(ctx, WithDialer(mockDialer))
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	// Test multiple Close calls
	if err := client.Close(); err != nil {
		t.Fatalf("first close failed: %v", err)
	}

	if err := client.Close(); err != nil {
		t.Fatalf("second close failed: %v", err)
	}
}

func TestConnectionClosureIdempotency(t *testing.T) {
	ctx := context.Background()

	mockDialer := &mockDialer{}
	client, err := IdentityClient(ctx, WithDialer(mockDialer))
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	conn, err := client.Connect(ctx, "localhost:8080")
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}

	// Test multiple Close calls
	if err := conn.Close(); err != nil {
		t.Fatalf("first close failed: %v", err)
	}

	if err := conn.Close(); err != nil {
		t.Fatalf("second close failed: %v", err)
	}
}

func TestServerClosureIdempotency(t *testing.T) {
	ctx := context.Background()

	mockServerPort := &mockServerPort{}
	server, err := IdentityServer(ctx, WithServerImpl(mockServerPort), WithAddress("localhost:0"))
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// Test multiple Close calls
	if err := server.Close(); err != nil {
		t.Fatalf("first close failed: %v", err)
	}

	if err := server.Close(); err != nil {
		t.Fatalf("second close failed: %v", err)
	}
}

func TestTimeoutConfiguration(t *testing.T) {
	ctx := context.Background()

	timeout := 5 * time.Second
	mockDialer := &mockDialer{}
	client, err := IdentityClient(ctx, WithDialer(mockDialer), WithClientTimeout(timeout))
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	// The timeout is applied during Connect
	conn, err := client.Connect(ctx, "localhost:8080", WithDialTimeout(2*time.Second))
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()
}

func TestErrorSentinels(t *testing.T) {
	// Test that sentinel errors are defined
	sentinelErrors := []error{
		ErrNoAuth,
		ErrNoSPIFFEAuth,
		ErrInvalidIdentity,
		ErrConfigInvalid,
		ErrConnectionFailed,
		ErrServerClosed,
		ErrInvalidAddress,
		ErrTimeout,
	}

	for _, err := range sentinelErrors {
		if err == nil {
			t.Error("sentinel error should not be nil")
		}
		if err.Error() == "" {
			t.Error("sentinel error should have a message")
		}
	}
}
