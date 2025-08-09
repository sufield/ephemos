// Package ephemos provides identity-based authentication for backend services using SPIFFE/SPIRE.
// It abstracts away all SPIFFE/SPIRE complexity, making identity-based authentication as simple as using API keys.
package ephemos

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/sufield/ephemos/internal/adapters/primary/api"
	"github.com/sufield/ephemos/internal/core/errors"
)

// ServiceRegistrar is the interface that service implementations must implement.
// Implementations should register their gRPC service with the provided server.
// This interface abstracts away gRPC registration details from service developers.
//
// Example implementation:
//
//	type EchoServiceRegistrar struct {
//		service EchoServiceServer
//	}
//
//	func (r *EchoServiceRegistrar) Register(grpcServer *grpc.Server) {
//		proto.RegisterEchoServiceServer(grpcServer, r.service)
//	}
type ServiceRegistrar = api.ServiceRegistrar

// Server represents an identity-aware gRPC server that handles automatic mTLS authentication.
// Services registered with this server will automatically use SPIFFE/SPIRE for identity verification.
// The server handles all certificate management and peer authentication transparently.
//
// Usage:
//
//	ctx := context.Background()
//	server, err := ephemos.NewIdentityServer(ctx, "")
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer server.Close() // Ensure proper cleanup
//
//	serviceRegistrar := proto.NewYourServiceRegistrar(&yourService{})
//	server.RegisterService(ctx, serviceRegistrar)
//
//	lis, _ := net.Listen("tcp", ":50051")
//	defer lis.Close() // Ensure listener cleanup
//	server.Serve(ctx, lis)
type Server interface {
	// RegisterService registers a gRPC service implementation with the server.
	// The serviceRegistrar must implement the ServiceRegistrar interface.
	// The context can be used for cancellation and timeouts during initialization.
	// Returns an error if registration fails or if serviceRegistrar is nil.
	RegisterService(ctx context.Context, serviceRegistrar ServiceRegistrar) error

	// Serve starts the server and listens for incoming connections on the provided listener.
	// The context can be used for cancellation and timeouts during server initialization.
	// This method blocks until the server is shut down or the context is canceled.
	// The caller should defer listener.Close() to ensure proper resource cleanup.
	Serve(ctx context.Context, listener net.Listener) error

	// Close gracefully shuts down the server and releases resources.
	// Should be called when the server is no longer needed.
	Close() error
}

// Client represents an identity-aware gRPC client that handles automatic mTLS authentication.
// Connections made through this client will automatically use SPIFFE/SPIRE for identity verification.
// The client handles all certificate management and server authentication transparently.
//
// Usage:
//
//	ctx := context.Background()
//	client, err := ephemos.NewIdentityClient(ctx, "")
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer client.Close() // Ensure proper cleanup
//
//	conn, err := client.Connect(ctx, "service-name", "localhost:50051")
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer conn.Close() // Always defer connection cleanup
type Client interface {
	// Connect establishes a connection to the specified service at the given address.
	// The serviceName is used for identity verification and must be non-empty.
	// The address should be in the format "host:port" and must be non-empty.
	// The context can be used for cancellation and timeouts.
	// Returns a ClientConnection that provides access to the underlying gRPC connection.
	// The caller must call Close() on the returned connection to release resources.
	Connect(ctx context.Context, serviceName, address string) (*api.ClientConnection, error)

	// Close releases any resources held by the client.
	// Should be called when the client is no longer needed.
	Close() error
}

// NewIdentityServer creates a new identity-aware server instance.
// It reads configuration from the EPHEMOS_CONFIG environment variable or uses defaults.
// The server will automatically handle mTLS authentication using SPIFFE/SPIRE.
// Returns an error if server creation fails, allowing callers to handle failures gracefully.
//
// Example:
//
//	ctx := context.Background()
//	server, err := ephemos.NewIdentityServer(ctx, "")
//	if err != nil {
//		return fmt.Errorf("failed to create server: %w", err)
//	}
//	defer server.Close()
//
//	registrar := proto.NewServiceRegistrar(&myService{})
//	server.RegisterService(ctx, registrar)
//	lis, _ := net.Listen("tcp", ":50051")
//	defer lis.Close()
//	server.Serve(ctx, lis)
func NewIdentityServer(ctx context.Context, configPath string) (Server, error) {
	if ctx == nil {
		return nil, &errors.ValidationError{
			Field:   "context",
			Value:   nil,
			Message: "context cannot be nil",
		}
	}

	validConfigPath, err := validateConfigPath(configPath)
	if err != nil {
		return nil, fmt.Errorf("invalid config path: %w", err)
	}

	server, err := api.NewIdentityServer(validConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create identity server: %w", err)
	}

	return server, nil
}

// NewIdentityClient creates a new identity-aware client instance.
// It reads configuration from the EPHEMOS_CONFIG environment variable or uses defaults.
// The client will automatically handle mTLS authentication using SPIFFE/SPIRE.
// Returns an error if client creation fails, allowing callers to handle failures gracefully.
//
// Example:
//
//	ctx := context.Background()
//	client, err := ephemos.NewIdentityClient(ctx, "")
//	if err != nil {
//		return fmt.Errorf("failed to create client: %w", err)
//	}
//	defer client.Close()
//
//	conn, err := client.Connect(ctx, "my-service", "localhost:50051")
//	if err != nil {
//		return fmt.Errorf("failed to connect: %w", err)
//	}
//	defer conn.Close()
//
//	serviceClient := proto.NewServiceClient(conn.GetClientConnection())
func NewIdentityClient(ctx context.Context, configPath string) (Client, error) {
	if ctx == nil {
		return nil, &errors.ValidationError{
			Field:   "context",
			Value:   nil,
			Message: "context cannot be nil",
		}
	}

	validConfigPath, err := validateConfigPath(configPath)
	if err != nil {
		return nil, fmt.Errorf("invalid config path: %w", err)
	}

	client, err := api.NewIdentityClient(validConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create identity client: %w", err)
	}

	return client, nil
}

// validateConfigPath validates and returns a proper config path.
// If configPath is empty, it checks EPHEMOS_CONFIG environment variable.
// Returns an error if validation fails.
func validateConfigPath(configPath string) (string, error) {
	// If explicitly provided, validate it
	if configPath != "" {
		if strings.TrimSpace(configPath) == "" {
			return "", &errors.ValidationError{
				Field:   "configPath",
				Value:   configPath,
				Message: "config path cannot be whitespace only",
			}
		}
		return strings.TrimSpace(configPath), nil
	}

	// Check environment variable
	envConfig := os.Getenv("EPHEMOS_CONFIG")
	if envConfig != "" {
		return strings.TrimSpace(envConfig), nil
	}

	// Return empty string to use defaults
	return "", nil
}

// Legacy compatibility functions - deprecated, use New* functions instead

// IdentityServer creates a new identity-aware server instance.
// Deprecated: Use NewIdentityServer for better error handling.
func IdentityServer() Server {
	ctx := context.Background()
	server, err := NewIdentityServer(ctx, "")
	if err != nil {
		panic(fmt.Sprintf("Failed to create identity server: %v", err))
	}
	return server
}

// IdentityClient creates a new identity-aware client instance.
// Deprecated: Use NewIdentityClient for better error handling.
func IdentityClient() Client {
	ctx := context.Background()
	client, err := NewIdentityClient(ctx, "")
	if err != nil {
		panic(fmt.Sprintf("Failed to create identity client: %v", err))
	}
	return client
}
