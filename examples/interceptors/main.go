// Package main demonstrates how to use built-in interceptors with ephemos.
// This example shows different interceptor configurations for various environments.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"google.golang.org/grpc"

	"github.com/sufield/ephemos/examples/proto"
	"github.com/sufield/ephemos/internal/adapters/interceptors"
	"github.com/sufield/ephemos/internal/adapters/logging"
	"github.com/sufield/ephemos/pkg/ephemos"
)

// ExampleServer demonstrates interceptor usage.
type ExampleServer struct {
	proto.UnimplementedEchoServiceServer
}

// Echo implements the echo service with interceptor support.
func (s *ExampleServer) Echo(ctx context.Context, req *proto.EchoRequest) (*proto.EchoResponse, error) {
	// Extract authenticated identity (if auth interceptor is enabled)
	if identity, ok := interceptors.GetIdentityFromContext(ctx); ok {
		slog.Info("Processing request from authenticated client",
			"spiffe_id", identity.SPIFFEID,
			"service", identity.ServiceName,
			"message", req.Message)
	}

	// Extract propagated identity information
	if requestID, ok := interceptors.GetRequestID(ctx); ok {
		slog.Info("Processing request", "request_id", requestID)
	}

	if originalCaller, ok := interceptors.GetOriginalCaller(ctx); ok {
		slog.Info("Request originated from", "original_caller", originalCaller)
	}

	return &proto.EchoResponse{
		Message: fmt.Sprintf("Echo: %s", req.Message),
		From:    "interceptor-example-server",
	}, nil
}

func main() {
	// Setup logging
	baseHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	secureLogger := logging.NewSecureLogger(baseHandler)
	slog.SetDefault(secureLogger)

	// Example 1: Production configuration with all interceptors
	fmt.Println("=== Production Configuration Example ===")
	runServerExample("production", ephemos.NewProductionInterceptorConfig("example-service"))

	// Example 2: Development configuration with relaxed auth
	fmt.Println("\n=== Development Configuration Example ===")
	runServerExample("development", ephemos.NewDevelopmentInterceptorConfig("example-service"))

	// Example 3: Custom configuration
	fmt.Println("\n=== Custom Configuration Example ===")
	runServerExample("custom", createCustomInterceptorConfig())
}

func runServerExample(configType string, interceptorConfig *ephemos.InterceptorConfig) {
	ctx := context.Background()

	// Create identity server
	server, err := ephemos.NewIdentityServer(ctx, "")
	if err != nil {
		slog.Error("Failed to create server", "config_type", configType, "error", err)
		return
	}
	defer server.Close()

	// Register service with interceptors
	serviceRegistrar := ephemos.NewServiceRegistrar(func(s *grpc.Server) {
		proto.RegisterEchoServiceServer(s, &ExampleServer{})
	})

	if err := server.RegisterService(ctx, serviceRegistrar); err != nil {
		slog.Error("Failed to register service", "config_type", configType, "error", err)
		return
	}

	slog.Info("Server configured successfully",
		"config_type", configType,
		"auth_enabled", interceptorConfig.EnableAuth,
		"logging_enabled", interceptorConfig.EnableLogging,
		"metrics_enabled", interceptorConfig.EnableMetrics,
		"identity_propagation_enabled", interceptorConfig.EnableIdentityPropagation)

	// In a real application, you would start the server here:
	// lis, _ := net.Listen("tcp", ":50051")
	// server.Serve(ctx, lis)
}

// createCustomInterceptorConfig demonstrates creating a custom interceptor configuration.
func createCustomInterceptorConfig() *ephemos.InterceptorConfig {
	// Create custom auth config with specific allowed services
	authConfig := interceptors.NewAllowListAuthConfig([]string{
		"spiffe://example.org/echo-client",
		"spiffe://example.org/admin-service",
	})
	authConfig.SkipMethods = []string{
		"/grpc.health.v1.Health/Check",
		"/example.EchoService/Ping", // Custom health check method
	}

	// Create custom logging config for audit requirements
	loggingConfig := interceptors.DefaultLoggingConfig()
	loggingConfig.LogPayloads = true // Enable payload logging for audit
	loggingConfig.IncludeHeaders = []string{
		"authorization",
		"x-request-id",
		"x-forwarded-for",
	}

	// Create custom metrics config
	metricsConfig := interceptors.DefaultMetricsConfig("custom-service")
	metricsConfig.EnablePayloadSize = true
	metricsConfig.EnableActiveRequests = true

	return &ephemos.InterceptorConfig{
		EnableAuth:                true,
		AuthConfig:                authConfig,
		EnableIdentityPropagation: true,
		IdentityPropagationConfig: nil, // Will use defaults
		EnableLogging:             true,
		LoggingConfig:             loggingConfig,
		EnableMetrics:             true,
		MetricsConfig:             metricsConfig,
	}
}

// Example of using interceptors with a client
func clientExample() {
	ctx := context.Background()

	// Create identity client
	client, err := ephemos.NewIdentityClient(ctx, "")
	if err != nil {
		slog.Error("Failed to create client", "error", err)
		return
	}
	defer client.Close()

	// Connect to service
	conn, err := client.Connect(ctx, "echo-service", "localhost:50051")
	if err != nil {
		slog.Error("Failed to connect", "error", err)
		return
	}
	defer conn.Close()

	// Create gRPC client
	echoClient := proto.NewEchoServiceClient(conn.GetClientConnection())

	// Make request (interceptors will automatically add identity propagation)
	resp, err := echoClient.Echo(ctx, &proto.EchoRequest{
		Message: "Hello with interceptors!",
	})
	if err != nil {
		slog.Error("Failed to make request", "error", err)
		return
	}

	slog.Info("Received response", "message", resp.Message, "from", resp.From)
}

// Example of implementing a custom metrics collector
type CustomMetricsCollector struct {
	// Add your metrics backend here (Prometheus, StatsD, etc.)
}

func (c *CustomMetricsCollector) IncRequestsTotal(method, service, code string) {
	// Implement your metrics collection logic
	fmt.Printf("REQUEST: %s %s -> %s\n", service, method, code)
}

func (c *CustomMetricsCollector) ObserveRequestDuration(method, service, code string, duration time.Duration) {
	// Implement your metrics collection logic
	fmt.Printf("DURATION: %s %s -> %s: %v\n", service, method, code, duration)
}

func (c *CustomMetricsCollector) IncActiveRequests(method, service string) {
	fmt.Printf("ACTIVE+: %s %s\n", service, method)
}

func (c *CustomMetricsCollector) DecActiveRequests(method, service string) {
	fmt.Printf("ACTIVE-: %s %s\n", service, method)
}

func (c *CustomMetricsCollector) IncStreamMessagesTotal(method, service, direction string) {
	fmt.Printf("STREAM_MSG: %s %s %s\n", service, method, direction)
}

func (c *CustomMetricsCollector) IncAuthenticationTotal(service, result string) {
	fmt.Printf("AUTH: %s -> %s\n", service, result)
}

func (c *CustomMetricsCollector) ObservePayloadSize(method, service, direction string, size int) {
	fmt.Printf("PAYLOAD_SIZE: %s %s %s -> %d bytes\n", service, method, direction, size)
}

// customMetricsExample demonstrates using a custom metrics collector.
func customMetricsExample() *ephemos.InterceptorConfig {
	config := ephemos.NewDefaultInterceptorConfig()

	// Use custom metrics collector
	config.MetricsConfig.MetricsCollector = &CustomMetricsCollector{}

	return config
}
