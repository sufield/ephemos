// Package main demonstrates how to use built-in interceptors with ephemos.
// This example shows different interceptor configurations for various environments.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"google.golang.org/grpc"

	"github.com/sufield/ephemos/examples/proto"
	"github.com/sufield/ephemos/pkg/ephemos"
)

// ExampleServer demonstrates interceptor usage.
type ExampleServer struct {
	proto.UnimplementedEchoServiceServer
}

// Echo implements the echo service with automatic identity-based authentication.
// When interceptors are enabled, authentication happens automatically at the transport layer.
func (s *ExampleServer) Echo(ctx context.Context, req *proto.EchoRequest) (*proto.EchoResponse, error) {
	// If this method is called, authentication has already succeeded!
	// Ephemos interceptors handle identity verification before reaching this code
	slog.Info("Processing authenticated request", "message", req.Message)

	return &proto.EchoResponse{
		Message: fmt.Sprintf("Echo: %s", req.Message),
		From:    "interceptor-example-server",
	}, nil
}

func main() {
	// Setup logging
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))

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

// createCustomInterceptorConfig demonstrates using preset interceptor configurations.
func createCustomInterceptorConfig() *ephemos.InterceptorConfig {
	// Use production preset configuration which enables all interceptors
	// with secure defaults appropriate for production environments
	return ephemos.NewProductionInterceptorConfig("custom-service")
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

// Example of implementing a custom authentication metrics collector
type CustomAuthMetricsCollector struct {
	// Add your metrics backend here (Prometheus, StatsD, etc.)
}

func (c *CustomAuthMetricsCollector) IncAuthenticationTotal(service, result string) {
	// Implement your authentication metrics collection logic
	fmt.Printf("AUTH: %s -> %s\n", service, result)
}

// customMetricsExample demonstrates using a custom authentication metrics collector.
func customAuthMetricsExample() *ephemos.InterceptorConfig {
	config := ephemos.NewDefaultInterceptorConfig()

	// Use custom authentication metrics collector
	config.MetricsConfig.AuthMetricsCollector = &CustomAuthMetricsCollector{}

	return config
}
