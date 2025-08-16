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
	if result != defaultResultCode {
		t.Errorf("Expected '%s', got: %v", defaultResultCode, result)
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
