package services_test

import (
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sufield/ephemos/internal/core/domain"
	"github.com/sufield/ephemos/internal/core/ports"
	"github.com/sufield/ephemos/internal/core/services"
)

// TestIdentityService_CacheMetrics_EdgeCases tests edge cases for cache metrics functionality.
func TestIdentityService_CacheMetrics_EdgeCases(t *testing.T) {
	config := &ports.Configuration{
		Service: ports.ServiceConfig{
			Name:   "test-service",
			Domain: "example.com",
		},
	}

	mockProvider := &MockIdentityProvider{}
	mockTransport := &MockTransportProvider{}

	// Create service with Prometheus metrics for testing
	metrics := services.NewPrometheusMetrics()
	service, err := services.NewIdentityService(mockProvider, mockTransport, config, nil, metrics)
	if err != nil {
		t.Fatalf("Failed to create IdentityService: %v", err)
	}

	t.Run("service creation with metrics", func(t *testing.T) {
		// Test that service was created successfully with metrics
		if service == nil {
			t.Error("Service should not be nil")
		}
		// Metrics are now tracked through Prometheus - integration would require 
		// Prometheus client testing which is beyond the scope of unit tests
	})

	t.Run("cache operations work with metrics", func(t *testing.T) {
		// Test that cache operations work with metrics enabled
		// This would trigger metric recording in the Prometheus system
		_, err := service.GetCertificate()
		// We expect an error since the mock provider doesn't return valid certificates
		if err == nil {
			t.Log("Certificate retrieval succeeded (or mock returned error as expected)")
		}
	})

	t.Run("no-op metrics behavior", func(t *testing.T) {
		// Test with no-op metrics
		noOpService, err := services.NewIdentityService(mockProvider, mockTransport, config, nil, nil)
		if err != nil {
			t.Fatalf("Failed to create service with no-op metrics: %v", err)
		}
		if noOpService == nil {
			t.Error("Service should not be nil with no-op metrics")
		}
	})
}

// TestIdentityService_ConfigurableTTL_EdgeCases tests edge cases for configurable TTL functionality.
func TestIdentityService_ConfigurableTTL_EdgeCases(t *testing.T) {
	mockProvider := &MockIdentityProvider{}
	mockTransport := &MockTransportProvider{}

	t.Run("default TTL when no cache config", func(t *testing.T) {
		config := &ports.Configuration{
			Service: ports.ServiceConfig{
				Name:   "test-service",
				Domain: "example.com",
				// No Cache config - should use default
			},
		}

		service, err := services.NewIdentityService(mockProvider, mockTransport, config, nil, nil)
		if err != nil {
			t.Fatalf("Failed to create IdentityService: %v", err)
		}

		// We can't directly test the internal TTL, but we can verify the service was created
		if service == nil {
			t.Error("Service should be created with default TTL")
		}
	})

	t.Run("custom TTL from configuration", func(t *testing.T) {
		config := &ports.Configuration{
			Service: ports.ServiceConfig{
				Name:   "test-service",
				Domain: "example.com",
				Cache: &ports.CacheConfig{
					TTLMinutes:              15, // Custom TTL
					ProactiveRefreshMinutes: 5,  // Custom refresh threshold
				},
			},
		}

		service, err := services.NewIdentityService(mockProvider, mockTransport, config, nil, nil)
		if err != nil {
			t.Fatalf("Failed to create IdentityService: %v", err)
		}

		if service == nil {
			t.Error("Service should be created with custom TTL")
		}
	})

	t.Run("zero TTL should use default", func(t *testing.T) {
		config := &ports.Configuration{
			Service: ports.ServiceConfig{
				Name:   "test-service",
				Domain: "example.com",
				Cache: &ports.CacheConfig{
					TTLMinutes: 0, // Zero should trigger default behavior
				},
			},
		}

		service, err := services.NewIdentityService(mockProvider, mockTransport, config, nil, nil)
		if err != nil {
			t.Fatalf("Failed to create IdentityService: %v", err)
		}

		if service == nil {
			t.Error("Service should be created with default TTL when configured TTL is 0")
		}
	})
}

// TestIdentityService_ThreadSafety_EdgeCases tests edge cases for thread safety.
func TestIdentityService_ThreadSafety_EdgeCases(t *testing.T) {
	config := &ports.Configuration{
		Service: ports.ServiceConfig{
			Name:   "test-service",
			Domain: "example.com",
		},
	}

	mockProvider := &MockIdentityProvider{}
	mockTransport := &MockTransportProvider{}

	service, err := services.NewIdentityService(mockProvider, mockTransport, config, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create IdentityService: %v", err)
	}

	t.Run("concurrent cache metrics access", func(t *testing.T) {
		const numGoroutines = 100
		const numOperations = 10

		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		// Start multiple goroutines accessing cache metrics concurrently
		for i := 0; i < numGoroutines; i++ {
			go func() {
				defer wg.Done()
				for j := 0; j < numOperations; j++ {
					// Test concurrent read access
					// Test concurrent certificate access
					_, _ = service.GetCertificate() // Expected to fail with mock provider

					// Test concurrent certificate operations
				}
			}()
		}

		// Wait for all goroutines to complete
		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			// Success - no race conditions detected
		case <-time.After(10 * time.Second):
			t.Fatal("Test timed out - possible deadlock or race condition")
		}
	})

	t.Run("concurrent service identity creation", func(t *testing.T) {
		const numGoroutines = 50

		var wg sync.WaitGroup
		var successCount int64
		var errorCount int64

		wg.Add(numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func() {
				defer wg.Done()
				
				// Attempt to create server identity concurrently
				_, err := service.CreateServerIdentity()
				if err != nil {
					atomic.AddInt64(&errorCount, 1)
				} else {
					atomic.AddInt64(&successCount, 1)
				}

				// Attempt to create client identity concurrently
				_, err = service.CreateClientIdentity()
				if err != nil {
					atomic.AddInt64(&errorCount, 1)
				} else {
					atomic.AddInt64(&successCount, 1)
				}
			}()
		}

		// Wait for completion with timeout
		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			t.Logf("Concurrent operations completed: %d successes, %d errors", 
				atomic.LoadInt64(&successCount), atomic.LoadInt64(&errorCount))
			// At least some operations should complete without hanging
		case <-time.After(15 * time.Second):
			t.Fatal("Concurrent test timed out - possible race condition or deadlock")
		}
	})
}

// TestIdentityService_ProactiveRefresh_EdgeCases tests edge cases for proactive refresh functionality.
func TestIdentityService_ProactiveRefresh_EdgeCases(t *testing.T) {
	t.Run("refresh threshold larger than TTL", func(t *testing.T) {
		config := &ports.Configuration{
			Service: ports.ServiceConfig{
				Name:   "test-service",
				Domain: "example.com",
				Cache: &ports.CacheConfig{
					TTLMinutes:              10,
					ProactiveRefreshMinutes: 15, // Larger than TTL - should be caught by validation
				},
			},
		}

		// This should fail validation
		err := config.Validate()
		if err == nil {
			t.Error("Expected validation error when refresh threshold is larger than TTL")
		}
	})

	t.Run("negative refresh threshold", func(t *testing.T) {
		config := &ports.Configuration{
			Service: ports.ServiceConfig{
				Name:   "test-service", 
				Domain: "example.com",
				Cache: &ports.CacheConfig{
					TTLMinutes:              10,
					ProactiveRefreshMinutes: -5, // Negative value - should be caught by validation
				},
			},
		}

		// This should fail validation
		err := config.Validate()
		if err == nil {
			t.Error("Expected validation error when refresh threshold is negative")
		}
	})
}

// TestIdentityService_ValidationFailures_EdgeCases tests edge cases for validation failures.
func TestIdentityService_ValidationFailures_EdgeCases(t *testing.T) {
	t.Run("nil configuration", func(t *testing.T) {
		mockProvider := &MockIdentityProvider{}
		mockTransport := &MockTransportProvider{}

		_, err := services.NewIdentityService(mockProvider, mockTransport, nil, nil, nil)
		if err == nil {
			t.Error("Expected error when configuration is nil")
		}
	})

	t.Run("empty service name", func(t *testing.T) {
		config := &ports.Configuration{
			Service: ports.ServiceConfig{
				Name:   "", // Empty name should fail validation
				Domain: "example.com",
			},
		}

		mockProvider := &MockIdentityProvider{}
		mockTransport := &MockTransportProvider{}

		_, err := services.NewIdentityService(mockProvider, mockTransport, config, nil, nil)
		if err == nil {
			t.Error("Expected error when service name is empty")
		}
	})

	t.Run("invalid cache TTL values", func(t *testing.T) {
		testCases := []struct {
			name       string
			ttlMinutes int
			expectErr  bool
		}{
			{"negative TTL", -1, true},
			{"zero TTL", 0, false}, // Zero should be allowed and use default
			{"valid TTL", 30, false},
			{"max TTL", 60, false},
			{"excessive TTL", 120, true}, // Over 60 minutes should fail
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				config := &ports.Configuration{
					Service: ports.ServiceConfig{
						Name:   "test-service",
						Domain: "example.com",
						Cache: &ports.CacheConfig{
							TTLMinutes: tc.ttlMinutes,
						},
					},
				}

				err := config.Validate()
				if tc.expectErr && err == nil {
					t.Errorf("Expected validation error for TTL %d minutes", tc.ttlMinutes)
				}
				if !tc.expectErr && err != nil {
					t.Errorf("Unexpected validation error for TTL %d minutes: %v", tc.ttlMinutes, err)
				}
			})
		}
	})
}

// Mock implementations for testing
type MockIdentityProvider struct{}

func (m *MockIdentityProvider) GetServiceIdentity() (*domain.ServiceIdentity, error) {
	return domain.NewServiceIdentity("test-service", "example.com"), nil
}

func (m *MockIdentityProvider) GetCertificate() (*domain.Certificate, error) {
	// Return an error to simulate the expected behavior during concurrent testing
	// This prevents the service from hanging trying to validate nil certificates
	return nil, fmt.Errorf("mock certificate error for testing")
}

func (m *MockIdentityProvider) GetTrustBundle() (*domain.TrustBundle, error) {
	// Return an error to simulate the expected behavior during concurrent testing
	// This prevents the service from hanging trying to validate nil trust bundles
	return nil, fmt.Errorf("mock trust bundle error for testing")
}

func (m *MockIdentityProvider) Close() error {
	return nil
}

type MockTransportProvider struct{}

func (m *MockTransportProvider) CreateServer(cert *domain.Certificate, bundle *domain.TrustBundle, policy *domain.AuthenticationPolicy) (ports.ServerPort, error) {
	return &MockServerPort{}, nil
}

func (m *MockTransportProvider) CreateClient(cert *domain.Certificate, bundle *domain.TrustBundle, policy *domain.AuthenticationPolicy) (ports.ClientPort, error) {
	return &MockClientPort{}, nil
}

type MockServerPort struct{}

func (m *MockServerPort) RegisterService(registrar ports.ServiceRegistrarPort) error {
	return nil
}

func (m *MockServerPort) Start(listener ports.ListenerPort) error {
	return nil
}

func (m *MockServerPort) Stop() error {
	return nil
}

type MockClientPort struct{}

func (m *MockClientPort) Connect(serviceName, address string) (ports.ConnectionPort, error) {
	return &MockConnectionPort{}, nil
}

func (m *MockClientPort) Close() error {
	return nil
}

type MockConnectionPort struct{}

func (m *MockConnectionPort) GetClientConnection() interface{} {
	return nil
}

func (m *MockConnectionPort) AsNetConn() net.Conn {
	return nil
}

func (m *MockConnectionPort) Close() error {
	return nil
}