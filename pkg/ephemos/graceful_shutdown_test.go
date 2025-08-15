package ephemos

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	// Internal imports removed for public API compliance
)

// Mock implementations for testing

type mockSPIFFEProvider struct {
	closeCalled atomic.Bool
	closeError  error
}

func (m *mockSPIFFEProvider) GetTLSConfig(ctx context.Context) (*tls.Config, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockSPIFFEProvider) GetServiceIdentity() (ServiceIdentity, error) {
	return &mockServiceIdentity{}, nil
}

func (m *mockSPIFFEProvider) Close() error {
	m.closeCalled.Store(true)
	return m.closeError
}

// mockServiceIdentity implements ServiceIdentity interface
type mockServiceIdentity struct{}

func (m *mockServiceIdentity) GetName() string {
	return "test-service"
}

func (m *mockServiceIdentity) GetDomain() string {
	return "test.domain"
}

func (m *mockServiceIdentity) GetSPIFFEID() string {
	return "spiffe://test.domain/test-service"
}

func (m *mockServiceIdentity) Validate() error {
	return nil
}

type mockServer struct {
	closeCalled atomic.Bool
	closeDelay  time.Duration
	closeError  error
}

func (m *mockServer) Close() error {
	m.closeCalled.Store(true)
	if m.closeDelay > 0 {
		time.Sleep(m.closeDelay)
	}
	return m.closeError
}

type mockClient struct {
	closeCalled atomic.Bool
	closeDelay  time.Duration
	closeError  error
}

func (m *mockClient) Connect(ctx context.Context, serviceName, address string) (*ClientConnection, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockClient) Close() error {
	m.closeCalled.Store(true)
	if m.closeDelay > 0 {
		time.Sleep(m.closeDelay)
	}
	return m.closeError
}

type mockListener struct {
	closeCalled atomic.Bool
	closeError  error
	address     string
}

func (m *mockListener) Accept() (net.Conn, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockListener) Close() error {
	m.closeCalled.Store(true)
	return m.closeError
}

func (m *mockListener) Addr() net.Addr {
	return &mockAddr{address: m.address}
}

// mockAddr implements net.Addr interface
type mockAddr struct {
	address string
}

func (m *mockAddr) Network() string {
	return "tcp"
}

func (m *mockAddr) String() string {
	return m.address
}

// Tests

func TestShutdownCoordinator_BasicShutdown(t *testing.T) {
	config := &ShutdownConfig{
		GracePeriod:  2 * time.Second,
		DrainTimeout: 1 * time.Second,
		ForceTimeout: 3 * time.Second,
	}

	coordinator := NewShutdownCoordinator(config)

	// Register mock resources
	server := &mockServer{}
	client := &mockClient{}
	listener := &mockListener{address: "test:1234"}

	coordinator.RegisterServer(server)
	coordinator.RegisterClient(client)
	coordinator.RegisterListener(listener)

	// Perform shutdown
	ctx := t.Context()
	err := coordinator.Shutdown(ctx)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Verify all resources were closed
	if !server.closeCalled.Load() {
		t.Error("Server close was not called")
	}
	if !client.closeCalled.Load() {
		t.Error("Client close was not called")
	}
	if !listener.closeCalled.Load() {
		t.Error("Listener close was not called")
	}
}

func TestShutdownCoordinator_ShutdownWithErrors(t *testing.T) {
	config := &ShutdownConfig{
		GracePeriod:  1 * time.Second,
		DrainTimeout: 500 * time.Millisecond,
		ForceTimeout: 2 * time.Second,
	}

	coordinator := NewShutdownCoordinator(config)

	// Register resources that will error
	server := &mockServer{closeError: errors.New("server close failed")}
	client := &mockClient{closeError: errors.New("client close failed")}
	listener := &mockListener{closeError: errors.New("listener close failed")}

	coordinator.RegisterServer(server)
	coordinator.RegisterClient(client)
	coordinator.RegisterListener(listener)

	// Perform shutdown
	ctx := t.Context()
	err := coordinator.Shutdown(ctx)

	// Should return an error containing all failures
	if err == nil {
		t.Error("Expected error from shutdown with failures")
	}

	// Verify all resources were still attempted to close
	if !server.closeCalled.Load() {
		t.Error("Server close was not attempted")
	}
	if !client.closeCalled.Load() {
		t.Error("Client close was not attempted")
	}
	if !listener.closeCalled.Load() {
		t.Error("Listener close was not attempted")
	}
}

func TestShutdownCoordinator_ShutdownWithTimeout(t *testing.T) {
	config := &ShutdownConfig{
		GracePeriod:  50 * time.Millisecond, // Very short grace period
		DrainTimeout: 25 * time.Millisecond,
		ForceTimeout: 100 * time.Millisecond,
	}

	coordinator := NewShutdownCoordinator(config)

	// Register a slow server that will timeout
	server := &mockServer{closeDelay: 200 * time.Millisecond} // Takes longer than grace period
	coordinator.RegisterServer(server)

	// Perform shutdown
	ctx := t.Context()

	start := time.Now()
	err := coordinator.Shutdown(ctx)
	elapsed := time.Since(start)

	// Should complete but with timeout error for the server
	if err == nil {
		t.Error("Expected timeout error for slow server")
	}

	// Should take about the grace period time (not wait for full server delay)
	if elapsed > 150*time.Millisecond {
		t.Errorf("Shutdown took too long: %v", elapsed)
	}
	if elapsed < 40*time.Millisecond {
		t.Errorf("Shutdown completed too quickly: %v", elapsed)
	}
}

func TestShutdownCoordinator_CustomCleanupFuncs(t *testing.T) {
	config := DefaultShutdownConfig()
	coordinator := NewShutdownCoordinator(config)

	var cleanupCalled atomic.Bool
	var cleanupOrder []string
	var mu sync.Mutex

	// Register cleanup functions
	coordinator.RegisterCleanupFunc(func() error {
		mu.Lock()
		cleanupOrder = append(cleanupOrder, "cleanup1")
		mu.Unlock()
		cleanupCalled.Store(true)
		return nil
	})

	coordinator.RegisterCleanupFunc(func() error {
		mu.Lock()
		cleanupOrder = append(cleanupOrder, "cleanup2")
		mu.Unlock()
		return errors.New("cleanup2 failed")
	})

	// Perform shutdown
	ctx := t.Context()
	err := coordinator.Shutdown(ctx)

	// Should have error from cleanup2
	if err == nil {
		t.Error("Expected error from failed cleanup function")
	}

	// Verify cleanup was called
	if !cleanupCalled.Load() {
		t.Error("Cleanup function was not called")
	}

	// Verify both cleanups ran
	mu.Lock()
	if len(cleanupOrder) != 2 {
		t.Errorf("Expected 2 cleanup calls, got %d", len(cleanupOrder))
	}
	mu.Unlock()
}

func TestShutdownCoordinator_Callbacks(t *testing.T) {
	var startCalled, completeCalled atomic.Bool
	var completeErr error

	config := &ShutdownConfig{
		GracePeriod:  100 * time.Millisecond,
		DrainTimeout: 50 * time.Millisecond,
		ForceTimeout: 200 * time.Millisecond,
		OnShutdownStart: func() {
			startCalled.Store(true)
		},
		OnShutdownComplete: func(err error) {
			completeCalled.Store(true)
			completeErr = err
		},
	}

	coordinator := NewShutdownCoordinator(config)

	// Add a server that will fail
	server := &mockServer{closeError: errors.New("close failed")}
	coordinator.RegisterServer(server)

	// Perform shutdown
	ctx := t.Context()
	err := coordinator.Shutdown(ctx)

	// Verify callbacks were called
	if !startCalled.Load() {
		t.Error("OnShutdownStart was not called")
	}
	if !completeCalled.Load() {
		t.Error("OnShutdownComplete was not called")
	}

	// Verify error was passed to complete callback
	if completeErr == nil {
		t.Error("Expected error in OnShutdownComplete")
	}
	if err == nil {
		t.Error("Expected error from Shutdown")
	}
}

func TestShutdownCoordinator_MultipleShutdownCalls(t *testing.T) {
	config := DefaultShutdownConfig()
	coordinator := NewShutdownCoordinator(config)

	server := &mockServer{}
	coordinator.RegisterServer(server)

	// Call shutdown multiple times concurrently
	var wg sync.WaitGroup
	errors := make([]error, 5)

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			ctx := t.Context()
			errors[idx] = coordinator.Shutdown(ctx)
		}(i)
	}

	wg.Wait()

	// Server should only be closed once
	if !server.closeCalled.Load() {
		t.Error("Server was not closed")
	}

	// All shutdown calls should complete without panic
	for i, err := range errors {
		if err != nil {
			t.Logf("Shutdown %d returned error: %v", i, err)
		}
	}
}

func TestShutdownCoordinator_SPIFFEProviderCleanup(t *testing.T) {
	config := DefaultShutdownConfig()
	coordinator := NewShutdownCoordinator(config)

	// Create a mock SPIFFE provider
	provider := &mockSPIFFEProvider{}
	coordinator.RegisterSPIFFEProvider(provider)

	// Perform shutdown
	ctx := t.Context()
	err := coordinator.Shutdown(ctx)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Verify the provider was closed
	if !provider.closeCalled.Load() {
		t.Error("SPIFFEProvider close was not called")
	}
}

func TestIdentityOrchestrator_ServeWithShutdown(t *testing.T) {
	// Create a test listener
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	// Note: This test would need proper mocking of the server
	// For now, we're testing the shutdown coordinator independently
	t.Skip("Requires proper mocking of server implementation")
}

func TestShutdownConfig_Defaults(t *testing.T) {
	config := DefaultShutdownConfig()

	if config.GracePeriod != 30*time.Second {
		t.Errorf("Expected GracePeriod of 30s, got %v", config.GracePeriod)
	}
	if config.DrainTimeout != 20*time.Second {
		t.Errorf("Expected DrainTimeout of 20s, got %v", config.DrainTimeout)
	}
	if config.ForceTimeout != 45*time.Second {
		t.Errorf("Expected ForceTimeout of 45s, got %v", config.ForceTimeout)
	}
}

// Benchmark tests

func BenchmarkShutdown_NoResources(b *testing.B) {
	for i := 0; i < b.N; i++ {
		config := DefaultShutdownConfig()
		coordinator := NewShutdownCoordinator(config)
		ctx := b.Context()
		_ = coordinator.Shutdown(ctx)
	}
}

func BenchmarkShutdown_WithResources(b *testing.B) {
	for i := 0; i < b.N; i++ {
		config := &ShutdownConfig{
			GracePeriod:  10 * time.Millisecond,
			DrainTimeout: 5 * time.Millisecond,
			ForceTimeout: 20 * time.Millisecond,
		}
		coordinator := NewShutdownCoordinator(config)

		// Register multiple resources
		for j := 0; j < 10; j++ {
			coordinator.RegisterServer(&mockServer{})
			coordinator.RegisterClient(&mockClient{})
			coordinator.RegisterListener(&mockListener{})
		}

		ctx := b.Context()
		_ = coordinator.Shutdown(ctx)
	}
}
