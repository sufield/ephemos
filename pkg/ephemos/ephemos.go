// Package ephemos provides identity-based authentication for backend services using SPIFFE/SPIRE.
// It abstracts away all SPIFFE/SPIRE complexity, making identity-based authentication as simple as using API keys.
package ephemos

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"

	"google.golang.org/grpc"

	"github.com/sufield/ephemos/internal/adapters/interceptors"
	"github.com/sufield/ephemos/internal/adapters/primary/api"
	"github.com/sufield/ephemos/internal/core/errors"
	"github.com/sufield/ephemos/internal/core/ports"
)

// ServiceRegistrar is the interface that service implementations must implement.
// Implementations should register their gRPC service with the provided server.
// This interface abstracts away gRPC registration details from service developers.
//
// Most developers should use the generic NewServiceRegistrar function instead of
// implementing this interface directly:
//
//	// Recommended approach (no boilerplate):
//	registrar := ephemos.NewServiceRegistrar(func(s *grpc.Server) {
//		proto.RegisterYourServiceServer(s, &yourService{})
//	})
//
// Advanced users can implement this interface directly for custom registration logic.
type ServiceRegistrar = api.ServiceRegistrar

// GenericServiceRegistrar is a generic implementation that can register any gRPC service
// without requiring developers to write service-specific registrars. This eliminates
// boilerplate code and makes service registration a one-liner.
//
// Example usage:
//
//	// Instead of writing a custom registrar, use the generic one:
//	registrar := ephemos.NewServiceRegistrar(func(s *grpc.Server) {
//		proto.RegisterYourServiceServer(s, &YourServiceImpl{})
//	})
//	server.RegisterService(ctx, registrar)
type GenericServiceRegistrar struct {
	registerFunc func(*grpc.Server)
}

// NewServiceRegistrar creates a generic registrar that can be used for any gRPC service.
// This eliminates the need to write service-specific registrars, reducing boilerplate code.
//
// Parameters:
//   - registerFunc: A function that registers your service with the gRPC server.
//     This is typically just calling your generated Register*Server function.
//
// Example:
//
//	// For an Echo service:
//	registrar := ephemos.NewServiceRegistrar(func(s *grpc.Server) {
//		proto.RegisterEchoServiceServer(s, &MyEchoServer{})
//	})
//
//	// For any other service:
//	registrar := ephemos.NewServiceRegistrar(func(s *grpc.Server) {
//		proto.RegisterUserServiceServer(s, &MyUserService{})
//	})
func NewServiceRegistrar(registerFunc func(*grpc.Server)) ServiceRegistrar {
	return &GenericServiceRegistrar{
		registerFunc: registerFunc,
	}
}

// Register implements the ServiceRegistrar interface by calling the provided registration function.
func (r *GenericServiceRegistrar) Register(grpcServer *grpc.Server) {
	if r.registerFunc != nil {
		r.registerFunc(grpcServer)
	}
}

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
//	registrar := ephemos.NewServiceRegistrar(func(s *grpc.Server) {
//		proto.RegisterYourServiceServer(s, &myService{})
//	})
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

	server, err := api.NewIdentityServer(ctx, validConfigPath)
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

	client, err := api.NewIdentityClient(ctx, validConfigPath)
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

// Built-in Interceptors

// InterceptorConfig provides configuration for built-in interceptors.
type InterceptorConfig struct {
	// EnableAuth enables authentication interceptor
	EnableAuth bool
	// AuthConfig configuration for authentication
	AuthConfig *interceptors.AuthConfig

	// EnableIdentityPropagation enables identity propagation for outgoing calls
	EnableIdentityPropagation bool
	// IdentityPropagationConfig configuration for identity propagation
	IdentityPropagationConfig *interceptors.IdentityPropagationConfig

	// EnableLogging enables audit logging interceptor
	EnableLogging bool
	// LoggingConfig configuration for logging
	LoggingConfig *interceptors.LoggingConfig

	// EnableMetrics enables metrics collection interceptor
	EnableMetrics bool
	// MetricsConfig configuration for metrics
	MetricsConfig *interceptors.MetricsConfig
}

// NewDefaultInterceptorConfig creates a default interceptor configuration.
func NewDefaultInterceptorConfig() *InterceptorConfig {
	return &InterceptorConfig{
		EnableAuth:                true,
		AuthConfig:                interceptors.DefaultAuthConfig(),
		EnableIdentityPropagation: false, // Disabled by default
		EnableLogging:             true,
		LoggingConfig:             interceptors.NewSecureLoggingConfig(),
		EnableMetrics:             true,
		MetricsConfig:             interceptors.DefaultMetricsConfig("ephemos-service"),
	}
}

// NewProductionInterceptorConfig creates a production-ready interceptor configuration.
func NewProductionInterceptorConfig(serviceName string) *InterceptorConfig {
	return &InterceptorConfig{
		EnableAuth:                true,
		AuthConfig:                interceptors.DefaultAuthConfig(),
		EnableIdentityPropagation: true,
		EnableLogging:             true,
		LoggingConfig:             interceptors.NewSecureLoggingConfig(),
		EnableMetrics:             true,
		MetricsConfig:             interceptors.DefaultMetricsConfig(serviceName),
	}
}

// NewDevelopmentInterceptorConfig creates a development-friendly interceptor configuration.
func NewDevelopmentInterceptorConfig(serviceName string) *InterceptorConfig {
	return &InterceptorConfig{
		EnableAuth:                false, // Disabled for easier development
		AuthConfig:                interceptors.DefaultAuthConfig(),
		EnableIdentityPropagation: true,
		EnableLogging:             true,
		LoggingConfig:             interceptors.NewDebugLoggingConfig(),
		EnableMetrics:             true,
		MetricsConfig:             interceptors.DefaultMetricsConfig(serviceName),
	}
}

// CreateServerInterceptors creates gRPC server interceptors based on configuration.
func CreateServerInterceptors(
	config *InterceptorConfig,
	_ ports.IdentityProvider,
) ([]grpc.UnaryServerInterceptor, []grpc.StreamServerInterceptor) {
	var unaryInterceptors []grpc.UnaryServerInterceptor
	var streamInterceptors []grpc.StreamServerInterceptor

	// Identity propagation server interceptor (extracts metadata)
	if config.EnableIdentityPropagation {
		serverPropagation := interceptors.NewIdentityPropagationServerInterceptor(config.LoggingConfig.Logger)
		unaryInterceptors = append(unaryInterceptors, serverPropagation.UnaryServerInterceptor())
		streamInterceptors = append(streamInterceptors, serverPropagation.StreamServerInterceptor())
	}

	// Authentication interceptor
	if config.EnableAuth {
		authInterceptor := interceptors.NewAuthInterceptor(config.AuthConfig)
		unaryInterceptors = append(unaryInterceptors, authInterceptor.UnaryServerInterceptor())
		streamInterceptors = append(streamInterceptors, authInterceptor.StreamServerInterceptor())
	}

	// Logging interceptor
	if config.EnableLogging {
		loggingInterceptor := interceptors.NewLoggingInterceptor(config.LoggingConfig)
		unaryInterceptors = append(unaryInterceptors, loggingInterceptor.UnaryServerInterceptor())
		streamInterceptors = append(streamInterceptors, loggingInterceptor.StreamServerInterceptor())
	}

	// Metrics interceptor
	if config.EnableMetrics {
		metricsInterceptor := interceptors.NewMetricsInterceptor(config.MetricsConfig)
		unaryInterceptors = append(unaryInterceptors, metricsInterceptor.UnaryServerInterceptor())
		streamInterceptors = append(streamInterceptors, metricsInterceptor.StreamServerInterceptor())
	}

	return unaryInterceptors, streamInterceptors
}

// CreateClientInterceptors creates gRPC client interceptors based on configuration.
func CreateClientInterceptors(
	config *InterceptorConfig,
	identityProvider ports.IdentityProvider,
) ([]grpc.UnaryClientInterceptor, []grpc.StreamClientInterceptor) {
	var unaryInterceptors []grpc.UnaryClientInterceptor
	var streamInterceptors []grpc.StreamClientInterceptor

	// Identity propagation client interceptor (adds metadata)
	if config.EnableIdentityPropagation && identityProvider != nil {
		if config.IdentityPropagationConfig == nil {
			config.IdentityPropagationConfig = interceptors.DefaultIdentityPropagationConfig(identityProvider)
		}
		config.IdentityPropagationConfig.IdentityProvider = identityProvider

		clientPropagation := interceptors.NewIdentityPropagationInterceptor(config.IdentityPropagationConfig)
		unaryInterceptors = append(unaryInterceptors, clientPropagation.UnaryClientInterceptor())
		streamInterceptors = append(streamInterceptors, clientPropagation.StreamClientInterceptor())
	}

	// Metrics interceptor
	if config.EnableMetrics {
		metricsInterceptor := interceptors.NewMetricsInterceptor(config.MetricsConfig)
		unaryInterceptors = append(unaryInterceptors, metricsInterceptor.UnaryClientInterceptor())
		streamInterceptors = append(streamInterceptors, metricsInterceptor.StreamClientInterceptor())
	}

	return unaryInterceptors, streamInterceptors
}

// Transport-Agnostic API
//
// The following functions are defined in server.go and provide a transport-agnostic
// API where services are written with plain Go types and can run over gRPC, HTTP,
// or any future transport without code changes.
//
// Example usage:
//
//	ctx := context.Background()
//	server, err := ephemos.NewTransportServer(ctx, "config/service.yaml")
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer server.Close()
//
//	echoService := &MyEchoService{}
//	ephemos.Mount[ports.EchoService](server, echoService)
//	server.ListenAndServe(ctx)

// NewTransportServer creates a new transport-agnostic server instance.
// This is a wrapper around the function defined in server.go to ensure proper export.
func NewTransportServer(ctx context.Context, configPath string) (*TransportServer, error) {
	return newTransportServer(ctx, configPath)
}

// Mount registers a service implementation with a transport-agnostic server.
// This is a wrapper around the function defined in server.go to ensure proper export.
func Mount[T any](server *TransportServer, impl T) error {
	return mount[T](server, impl)
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
