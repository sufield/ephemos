package interceptors

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const testResponse = "response"

func TestNewLoggingInterceptor(t *testing.T) {
	config := DefaultLoggingConfig()
	interceptor := NewLoggingInterceptor(config)

	if interceptor == nil {
		t.Fatal("NewLoggingInterceptor returned nil")
	}
	if interceptor.config != config {
		t.Error("Config not properly set")
	}
	if interceptor.logger == nil {
		t.Error("Logger not set")
	}
}

func TestNewLoggingInterceptor_WithNilLogger(t *testing.T) {
	config := &LoggingConfig{
		Logger: nil,
	}
	interceptor := NewLoggingInterceptor(config)

	if interceptor.logger == nil {
		t.Error("Logger should be set when nil provided")
	}
}

func TestNewLoggingInterceptor_DefaultSlowThreshold(t *testing.T) {
	config := &LoggingConfig{
		SlowRequestThreshold: 0, // Should be set to default
	}
	interceptor := NewLoggingInterceptor(config)

	if interceptor.config.SlowRequestThreshold != defaultSlowThreshold {
		t.Errorf("Expected default slow threshold %v, got: %v",
			defaultSlowThreshold, interceptor.config.SlowRequestThreshold)
	}
}

func TestLoggingInterceptor_UnaryServerInterceptor_ExcludeMethod(t *testing.T) {
	var logOutput bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logOutput, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	config := &LoggingConfig{
		Logger:         logger,
		LogRequests:    true,
		ExcludeMethods: []string{"/test.Service/ExcludedMethod"},
	}
	interceptor := NewLoggingInterceptor(config)

	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		return defaultResultCode, nil
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/ExcludedMethod",
	}

	result, err := interceptor.UnaryServerInterceptor()(
		t.Context(), "request", info, handler)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result != "success" {
		t.Errorf("Expected 'success', got: %v", result)
	}

	// Should not have logged anything for excluded method
	if logOutput.Len() > 0 {
		t.Errorf("Expected no logs for excluded method, got: %s", logOutput.String())
	}
}

func TestLoggingInterceptor_UnaryServerInterceptor_Success(t *testing.T) {
	var logOutput bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logOutput, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	config := &LoggingConfig{
		Logger:      logger,
		LogRequests: true,
		LogPayloads: true,
	}
	interceptor := NewLoggingInterceptor(config)

	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
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

	logStr := logOutput.String()

	// Should contain request received log
	if !strings.Contains(logStr, "gRPC request received") {
		t.Error("Expected 'gRPC request received' in logs")
	}

	// Should contain request completed log
	if !strings.Contains(logStr, "gRPC request completed") {
		t.Error("Expected 'gRPC request completed' in logs")
	}

	// Should contain method name
	if !strings.Contains(logStr, "/test.Service/TestMethod") {
		t.Error("Expected method name in logs")
	}

	// Should contain success=true
	if !strings.Contains(logStr, "success=true") {
		t.Error("Expected success=true in logs")
	}
}

func TestLoggingInterceptor_UnaryServerInterceptor_Error(t *testing.T) {
	var logOutput bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logOutput, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	config := &LoggingConfig{
		Logger: logger,
	}
	interceptor := NewLoggingInterceptor(config)

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

	logStr := logOutput.String()

	// Should log at ERROR level
	if !strings.Contains(logStr, "level=ERROR") {
		t.Error("Expected ERROR level log for failed request")
	}

	// Should contain error information
	if !strings.Contains(logStr, "success=false") {
		t.Error("Expected success=false in logs")
	}
	if !strings.Contains(logStr, "error_code=Internal") {
		t.Errorf("Expected error_code=Internal in logs, got: %s", logStr)
	}
	if !strings.Contains(logStr, "test error") {
		t.Error("Expected error message in logs")
	}
}

func TestLoggingInterceptor_UnaryServerInterceptor_SlowRequest(t *testing.T) {
	var logOutput bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logOutput, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	config := &LoggingConfig{
		Logger:               logger,
		SlowRequestThreshold: 1 * time.Millisecond, // Very short threshold
	}
	interceptor := NewLoggingInterceptor(config)

	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		time.Sleep(5 * time.Millisecond) // Longer than threshold
		return testResponse, nil
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/SlowMethod",
	}

	_, err := interceptor.UnaryServerInterceptor()(
		t.Context(), "request", info, handler)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	logStr := logOutput.String()

	// Should log at WARN level for slow request
	if !strings.Contains(logStr, "level=WARN") {
		t.Error("Expected WARN level log for slow request")
	}
}

func TestLoggingInterceptor_UnaryServerInterceptor_WithIdentity(t *testing.T) {
	var logOutput bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logOutput, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	config := &LoggingConfig{
		Logger: logger,
	}
	interceptor := NewLoggingInterceptor(config)

	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		return testResponse, nil
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/TestMethod",
	}

	// Create context with identity
	identity := &AuthenticatedIdentity{
		SPIFFEID:    "spiffe://example.org/test-service",
		ServiceName: "test-service",
		TrustDomain: "example.org",
	}
	ctx := context.WithValue(t.Context(), IdentityContextKey{}, identity)

	_, err := interceptor.UnaryServerInterceptor()(ctx, "request", info, handler)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	logStr := logOutput.String()

	// Should contain identity information
	if !strings.Contains(logStr, "client_spiffe_id=spiffe://example.org/test-service") {
		t.Error("Expected client SPIFFE ID in logs")
	}
	if !strings.Contains(logStr, "client_service=test-service") {
		t.Error("Expected client service in logs")
	}
	if !strings.Contains(logStr, "client_trust_domain=example.org") {
		t.Error("Expected client trust domain in logs")
	}
}

func TestLoggingInterceptor_UnaryServerInterceptor_WithPropagatedInfo(t *testing.T) {
	var logOutput bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logOutput, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	config := &LoggingConfig{
		Logger: logger,
	}
	interceptor := NewLoggingInterceptor(config)

	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		return testResponse, nil
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/TestMethod",
	}

	// Create context with propagated identity information
	ctx := t.Context()
	ctx = context.WithValue(ctx, originalCallerKey, "spiffe://example.org/original")
	ctx = context.WithValue(ctx, callChainKey, "service1 -> service2")
	ctx = context.WithValue(ctx, requestIDKey, "req-123")

	_, err := interceptor.UnaryServerInterceptor()(ctx, "request", info, handler)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	logStr := logOutput.String()

	// Should contain propagated identity information
	if !strings.Contains(logStr, "original_caller=spiffe://example.org/original") {
		t.Errorf("Expected original caller in logs, got: %s", logStr)
	}
	if !strings.Contains(logStr, `call_chain="service1 -> service2"`) {
		t.Errorf("Expected call chain in logs, got: %s", logStr)
	}
	if !strings.Contains(logStr, "request_id=req-123") {
		t.Errorf("Expected request ID in logs, got: %s", logStr)
	}
}

func TestLoggingInterceptor_StreamServerInterceptor(t *testing.T) {
	var logOutput bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logOutput, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	config := &LoggingConfig{
		Logger:      logger,
		LogRequests: true,
	}
	interceptor := NewLoggingInterceptor(config)

	handler := func(_ interface{}, ss grpc.ServerStream) error {
		// Simulate some stream operations
		wrappedStream := ss.(*loggingServerStream)
		wrappedStream.messagesSent = 3
		wrappedStream.messagesReceived = 2
		return nil
	}

	info := &grpc.StreamServerInfo{
		FullMethod: "/test.Service/StreamMethod",
	}

	mockStream := &mockLoggingServerStream{
		contextFunc: func() context.Context { return t.Context() },
	}

	err := interceptor.StreamServerInterceptor()(nil, mockStream, info, handler)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	logStr := logOutput.String()

	// Should contain stream started log
	if !strings.Contains(logStr, "gRPC stream started") {
		t.Error("Expected 'gRPC stream started' in logs")
	}

	// Should contain stream completed log
	if !strings.Contains(logStr, "gRPC stream completed") {
		t.Error("Expected 'gRPC stream completed' in logs")
	}

	// Should contain message counts
	if !strings.Contains(logStr, "messages_sent=3") {
		t.Error("Expected messages_sent=3 in logs")
	}
	if !strings.Contains(logStr, "messages_received=2") {
		t.Error("Expected messages_received=2 in logs")
	}
}

func TestLoggingServerStream_SendMsg(t *testing.T) {
	var logOutput bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logOutput, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	mockStream := &mockLoggingServerStream{}
	stream := &loggingServerStream{
		ServerStream: mockStream,
		logger:       logger,
		method:       "/test.Service/TestMethod",
		logPayloads:  true,
	}

	err := stream.SendMsg("test message")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if stream.messagesSent != 1 {
		t.Errorf("Expected 1 message sent, got: %d", stream.messagesSent)
	}

	logStr := logOutput.String()
	if !strings.Contains(logStr, "Stream message sent") {
		t.Error("Expected 'Stream message sent' in debug logs")
	}
	if !strings.Contains(logStr, "test message") {
		t.Error("Expected payload in debug logs when enabled")
	}
}

func TestLoggingServerStream_SendMsg_Error(t *testing.T) {
	mockStream := &mockLoggingServerStream{
		sendErr: errors.New("send error"),
	}
	stream := &loggingServerStream{
		ServerStream: mockStream,
		logger:       slog.Default(),
		method:       "/test.Service/TestMethod",
	}

	err := stream.SendMsg("test message")

	if err == nil {
		t.Error("Expected error from SendMsg")
	}
	if stream.messagesSent != 0 {
		t.Errorf("Expected 0 messages sent due to error, got: %d", stream.messagesSent)
	}
}

func TestLoggingServerStream_RecvMsg(t *testing.T) {
	var logOutput bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logOutput, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	mockStream := &mockLoggingServerStream{}
	stream := &loggingServerStream{
		ServerStream: mockStream,
		logger:       logger,
		method:       "/test.Service/TestMethod",
		logPayloads:  false, // Disabled
	}

	err := stream.RecvMsg("test message")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if stream.messagesReceived != 1 {
		t.Errorf("Expected 1 message received, got: %d", stream.messagesReceived)
	}

	// Should not log payload when disabled
	logStr := logOutput.String()
	if strings.Contains(logStr, "test message") {
		t.Error("Expected no payload in logs when disabled")
	}
}

func TestShouldExcludeMethod(t *testing.T) {
	config := &LoggingConfig{
		ExcludeMethods: []string{
			"/grpc.health.v1.Health/Check",
			"/test.Service/ExcludedMethod",
		},
	}
	interceptor := NewLoggingInterceptor(config)

	tests := []struct {
		method   string
		expected bool
	}{
		{"/grpc.health.v1.Health/Check", true},
		{"/test.Service/ExcludedMethod", true},
		{"/test.Service/IncludedMethod", false},
		{"/other.Service/AnyMethod", false},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			result := interceptor.shouldExcludeMethod(tt.method)
			if result != tt.expected {
				t.Errorf("Expected %v for method %s, got: %v", tt.expected, tt.method, result)
			}
		})
	}
}

func TestDefaultLoggingConfig(t *testing.T) {
	config := DefaultLoggingConfig()

	if config.Logger != nil {
		t.Error("Expected nil logger in default config")
	}
	if !config.LogRequests {
		t.Error("Expected LogRequests to be true by default")
	}
	if !config.LogResponses {
		t.Error("Expected LogResponses to be true by default")
	}
	if config.LogPayloads {
		t.Error("Expected LogPayloads to be false by default for security")
	}
	if config.SlowRequestThreshold != defaultSlowThreshold {
		t.Errorf("Expected default slow threshold %v, got: %v",
			defaultSlowThreshold, config.SlowRequestThreshold)
	}
	if len(config.ExcludeMethods) < 2 {
		t.Error("Expected health check methods to be excluded by default")
	}
}

func TestNewSecureLoggingConfig(t *testing.T) {
	config := NewSecureLoggingConfig()

	if config.LogPayloads {
		t.Error("Expected LogPayloads to be false in secure config")
	}
}

func TestNewDebugLoggingConfig(t *testing.T) {
	config := NewDebugLoggingConfig()

	if !config.LogPayloads {
		t.Error("Expected LogPayloads to be true in debug config")
	}
	if config.SlowRequestThreshold != debugSlowThreshold {
		t.Errorf("Expected debug slow threshold %v, got: %v",
			debugSlowThreshold, config.SlowRequestThreshold)
	}
	if len(config.ExcludeMethods) != 0 {
		t.Error("Expected no excluded methods in debug config")
	}
}

// Mock implementations for testing

type mockLoggingServerStream struct {
	grpc.ServerStream
	sendErr     error
	recvErr     error
	contextFunc func() context.Context
}

func (m *mockLoggingServerStream) Context() context.Context {
	if m.contextFunc != nil {
		return m.contextFunc()
	}
	return context.Background()
}

func (m *mockLoggingServerStream) SendMsg(_ interface{}) error {
	return m.sendErr
}

func (m *mockLoggingServerStream) RecvMsg(_ interface{}) error {
	return m.recvErr
}
