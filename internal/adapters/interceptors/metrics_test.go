package interceptors

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type mockMetricsCollector struct {
	requestsTotal       int
	requestDuration     time.Duration
	activeRequests      int
	streamMessagesTotal int
	authenticationTotal int
	payloadSizes        []int
}

func (m *mockMetricsCollector) IncRequestsTotal(_, _, _ string) {
	m.requestsTotal++
}

func (m *mockMetricsCollector) ObserveRequestDuration(_, _, _ string, duration time.Duration) {
	m.requestDuration = duration
}

func (m *mockMetricsCollector) IncActiveRequests(_, _ string) {
	m.activeRequests++
}

func (m *mockMetricsCollector) DecActiveRequests(_, _ string) {
	m.activeRequests--
}

func (m *mockMetricsCollector) IncStreamMessagesTotal(_, _, _ string) {
	m.streamMessagesTotal++
}

func (m *mockMetricsCollector) IncAuthenticationTotal(_, _ string) {
	m.authenticationTotal++
}

func (m *mockMetricsCollector) ObservePayloadSize(_, _, _ string, size int) {
	m.payloadSizes = append(m.payloadSizes, size)
}

func TestNewMetricsInterceptor(t *testing.T) {
	collector := &mockMetricsCollector{}
	config := &MetricsConfig{
		MetricsCollector: collector,
		ServiceName:      "test-service",
	}

	interceptor := NewMetricsInterceptor(config)

	if interceptor == nil {
		t.Fatal("NewMetricsInterceptor returned nil")
	}
	if interceptor.config != config {
		t.Error("Config not properly set")
	}
}

func TestNewMetricsInterceptor_WithNilCollector(t *testing.T) {
	config := &MetricsConfig{
		MetricsCollector: nil,
		ServiceName:      "test-service",
	}

	interceptor := NewMetricsInterceptor(config)

	if interceptor.config.MetricsCollector == nil {
		t.Error("MetricsCollector should be set to default when nil provided")
	}
}

func TestMetricsInterceptor_UnaryServerInterceptor_Success(t *testing.T) {
	collector := &mockMetricsCollector{}
	config := &MetricsConfig{
		MetricsCollector:     collector,
		ServiceName:          "test-service",
		EnableActiveRequests: true,
		EnablePayloadSize:    true,
	}
	interceptor := NewMetricsInterceptor(config)

	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		time.Sleep(10 * time.Millisecond) // Simulate processing time
		return testResponse, nil
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/TestMethod",
	}

	result, err := interceptor.UnaryServerInterceptor()(
		t.Context(), "request", info, handler)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result != testResponse {
		t.Errorf("Expected 'response', got: %v", result)
	}

	// Check metrics were collected
	if collector.requestsTotal != 1 {
		t.Errorf("Expected 1 total request, got: %d", collector.requestsTotal)
	}
	if collector.requestDuration == 0 {
		t.Error("Expected request duration to be recorded")
	}
	if collector.activeRequests != 0 {
		t.Errorf("Expected active requests to be 0 after completion, got: %d", collector.activeRequests)
	}
	if len(collector.payloadSizes) == 0 {
		t.Error("Expected payload sizes to be recorded")
	}
}

func TestMetricsInterceptor_UnaryServerInterceptor_Error(t *testing.T) {
	collector := &mockMetricsCollector{}
	config := &MetricsConfig{
		MetricsCollector: collector,
		ServiceName:      "test-service",
	}
	interceptor := NewMetricsInterceptor(config)

	expectedError := status.Error(codes.Internal, "test error")
	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		return nil, fmt.Errorf("handler error: %w", expectedError)
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/TestMethod",
	}

	_, err := interceptor.UnaryServerInterceptor()(
		t.Context(), "request", info, handler)

	if !errors.Is(err, expectedError) {
		t.Errorf("Expected error %v, got: %v", expectedError, err)
	}

	// Check metrics were collected even for error
	if collector.requestsTotal != 1 {
		t.Errorf("Expected 1 total request, got: %d", collector.requestsTotal)
	}
}

func TestMetricsInterceptor_UnaryServerInterceptor_WithIdentity(t *testing.T) {
	collector := &mockMetricsCollector{}
	config := &MetricsConfig{
		MetricsCollector: collector,
		ServiceName:      "test-service",
	}
	interceptor := NewMetricsInterceptor(config)

	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		return testResponse, nil
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

func TestMetricsInterceptor_UnaryClientInterceptor(t *testing.T) {
	collector := &mockMetricsCollector{}
	config := &MetricsConfig{
		MetricsCollector:  collector,
		ServiceName:       "test-client",
		EnablePayloadSize: true,
	}
	interceptor := NewMetricsInterceptor(config)

	invoker := func(_ context.Context, _ string, _, _ interface{}, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
		time.Sleep(5 * time.Millisecond)
		return nil
	}

	err := interceptor.UnaryClientInterceptor()(
		t.Context(), "/test.Service/TestMethod",
		"request", "reply", nil, invoker)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Check metrics were collected
	if collector.requestsTotal != 1 {
		t.Errorf("Expected 1 total request, got: %d", collector.requestsTotal)
	}
	if collector.requestDuration == 0 {
		t.Error("Expected request duration to be recorded")
	}
}

func TestDefaultMetricsConfig(t *testing.T) {
	serviceName := "test-service"
	config := DefaultMetricsConfig(serviceName)

	if config.ServiceName != serviceName {
		t.Errorf("Expected service name %s, got: %s", serviceName, config.ServiceName)
	}
	if config.MetricsCollector == nil {
		t.Error("Expected default metrics collector")
	}
	if config.EnablePayloadSize {
		t.Error("Expected payload size to be disabled by default")
	}
	if !config.EnableActiveRequests {
		t.Error("Expected active requests to be enabled by default")
	}
}

func TestDefaultMetricsCollector(_ *testing.T) {
	collector := &DefaultMetricsCollector{}

	// All methods should be no-ops and not panic
	collector.IncRequestsTotal("method", "service", "code")
	collector.ObserveRequestDuration("method", "service", "code", time.Second)
	collector.IncActiveRequests("method", "service")
	collector.DecActiveRequests("method", "service")
	collector.IncStreamMessagesTotal("method", "service", "direction")
	collector.IncAuthenticationTotal("service", "result")
	collector.ObservePayloadSize("method", "service", "direction", 100)
}

func TestEstimatePayloadSize(t *testing.T) {
	tests := []struct {
		name     string
		payload  interface{}
		expected int
	}{
		{
			name:     "nil_payload",
			payload:  nil,
			expected: 0,
		},
		{
			name:     "string_payload",
			payload:  "hello world",
			expected: 11,
		},
		{
			name:     "byte_slice_payload",
			payload:  []byte("test data"),
			expected: 9,
		},
		{
			name:     "other_payload",
			payload:  struct{ name string }{name: "test"},
			expected: 64, // Default estimate
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			size := estimatePayloadSize(tt.payload)
			if size != tt.expected {
				t.Errorf("Expected size %d, got: %d", tt.expected, size)
			}
		})
	}
}

// Mock implementations for stream testing

type mockServerStream struct {
	grpc.ServerStream
	sendErr     error
	recvErr     error
	contextFunc func() context.Context
}

func (m *mockServerStream) Context() context.Context {
	if m.contextFunc != nil {
		return m.contextFunc()
	}
	return context.Background()
}

func (m *mockServerStream) SendMsg(_ interface{}) error {
	return m.sendErr
}

func (m *mockServerStream) RecvMsg(_ interface{}) error {
	return m.recvErr
}

func TestMetricsServerStream_SendMsg(t *testing.T) {
	collector := &mockMetricsCollector{}
	stream := &metricsServerStream{
		ServerStream: &mockServerStream{sendErr: nil},
		method:       "/test.Service/TestMethod",
		service:      "test-service",
		collector:    collector,
		enableSizes:  true,
	}

	err := stream.SendMsg("test message")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if collector.streamMessagesTotal != 1 {
		t.Errorf("Expected 1 stream message, got: %d", collector.streamMessagesTotal)
	}
	if len(collector.payloadSizes) == 0 {
		t.Error("Expected payload size to be recorded")
	}
}

func TestMetricsServerStream_SendMsg_Error(t *testing.T) {
	collector := &mockMetricsCollector{}
	expectedError := errors.New("send error")
	stream := &metricsServerStream{
		ServerStream: &mockServerStream{sendErr: expectedError},
		method:       "/test.Service/TestMethod",
		service:      "test-service",
		collector:    collector,
	}

	err := stream.SendMsg("test message")

	// Check that the error is wrapped properly
	if !errors.Is(err, expectedError) {
		t.Errorf("Expected wrapped error containing %v, got: %v", expectedError, err)
	}
	// Also verify the error message contains the wrapper text
	if err.Error() != "failed to send message: send error" {
		t.Errorf("Expected error message 'failed to send message: send error', got: %s", err.Error())
	}
	if collector.streamMessagesTotal != 0 {
		t.Errorf("Expected 0 stream messages due to error, got: %d", collector.streamMessagesTotal)
	}
}

func TestMetricsServerStream_RecvMsg(t *testing.T) {
	collector := &mockMetricsCollector{}
	stream := &metricsServerStream{
		ServerStream: &mockServerStream{recvErr: nil},
		method:       "/test.Service/TestMethod",
		service:      "test-service",
		collector:    collector,
		enableSizes:  false,
	}

	err := stream.RecvMsg("test message")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if collector.streamMessagesTotal != 1 {
		t.Errorf("Expected 1 stream message, got: %d", collector.streamMessagesTotal)
	}
	// Should not record payload size when disabled
	if len(collector.payloadSizes) != 0 {
		t.Error("Expected no payload sizes when disabled")
	}
}
