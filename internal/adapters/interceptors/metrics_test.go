package interceptors

import (
	"context"
	"testing"

	"google.golang.org/grpc"
)

// mockAuthMetricsCollector for testing authentication metrics.
type mockAuthMetricsCollector struct {
	authenticationTotal int
}

func (m *mockAuthMetricsCollector) IncAuthenticationTotal(_, _ string) {
	m.authenticationTotal++
}

func TestNewAuthMetricsInterceptor(t *testing.T) {
	collector := &mockAuthMetricsCollector{}
	config := &AuthMetricsConfig{
		AuthMetricsCollector: collector,
		ServiceName:          "test-service",
	}

	interceptor := NewAuthMetricsInterceptor(config)

	if interceptor == nil {
		t.Fatal("NewAuthMetricsInterceptor returned nil")
	}
	if interceptor.config != config {
		t.Error("Config not properly set")
	}
}

func TestNewAuthMetricsInterceptor_WithNilCollector(t *testing.T) {
	config := &AuthMetricsConfig{
		AuthMetricsCollector: nil,
		ServiceName:          "test-service",
	}

	interceptor := NewAuthMetricsInterceptor(config)

	if interceptor.config.AuthMetricsCollector == nil {
		t.Error("AuthMetricsCollector should be set to default when nil provided")
	}
}

func TestAuthMetricsInterceptor_UnaryServerInterceptor_WithIdentity(t *testing.T) {
	collector := &mockAuthMetricsCollector{}
	config := &AuthMetricsConfig{
		AuthMetricsCollector: collector,
		ServiceName:          "test-service",
	}
	interceptor := NewAuthMetricsInterceptor(config)

	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		return "success", nil
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/TestMethod",
	}

	// Create context with identity
	identity := &AuthenticatedIdentity{
		SPIFFEID:    "spiffe://example.org/test",
		ServiceName: "test-service",
	}
	ctx := context.WithValue(t.Context(), IdentityContextKey{}, identity)

	_, err := interceptor.UnaryServerInterceptor()(ctx, "request", info, handler)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Check authentication metrics were recorded
	if collector.authenticationTotal != 1 {
		t.Errorf("Expected 1 authentication total, got: %d", collector.authenticationTotal)
	}
}

func TestAuthMetricsInterceptor_UnaryServerInterceptor_WithoutIdentity(t *testing.T) {
	collector := &mockAuthMetricsCollector{}
	config := &AuthMetricsConfig{
		AuthMetricsCollector: collector,
		ServiceName:          "test-service",
	}
	interceptor := NewAuthMetricsInterceptor(config)

	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		return "success", nil
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/TestMethod",
	}

	// No identity in context
	_, err := interceptor.UnaryServerInterceptor()(t.Context(), "request", info, handler)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Check no authentication metrics were recorded
	if collector.authenticationTotal != 0 {
		t.Errorf("Expected 0 authentication total, got: %d", collector.authenticationTotal)
	}
}

func TestDefaultAuthMetricsConfig(t *testing.T) {
	serviceName := "test-service"
	config := DefaultAuthMetricsConfig(serviceName)

	if config.ServiceName != serviceName {
		t.Errorf("Expected service name %s, got: %s", serviceName, config.ServiceName)
	}
	if config.AuthMetricsCollector == nil {
		t.Error("Expected default auth metrics collector")
	}
}

func TestDefaultAuthMetricsCollector(_ *testing.T) {
	collector := &DefaultAuthMetricsCollector{}

	// Should be no-op and not panic
	collector.IncAuthenticationTotal("service", "result")
}