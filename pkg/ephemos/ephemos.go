// Package ephemos provides identity-based authentication for backend services using SPIFFE/SPIRE.
// It abstracts away all SPIFFE/SPIRE complexity, making identity-based authentication as simple as using API keys.
package ephemos

import (
	"context"
	"fmt"
	"log/slog"
	"net"
)

// ServiceRegistrar is the interface that service implementations must implement.
// This interface abstracts away transport registration details from service developers.
type ServiceRegistrar interface {
	// Register registers the service with the appropriate transport
	Register(transport interface{})
}

// GenericServiceRegistrar is a generic implementation that can register any service
// without requiring developers to write service-specific registrars. This eliminates
// boilerplate code and makes service registration a one-liner.
type GenericServiceRegistrar struct {
	registerFunc func(interface{})
}

// NewServiceRegistrar creates a generic registrar that can be used for any service.
// This eliminates the need to write service-specific registrars, reducing boilerplate code.
func NewServiceRegistrar(registerFunc func(interface{})) ServiceRegistrar {
	return &GenericServiceRegistrar{
		registerFunc: registerFunc,
	}
}

// Register implements the ServiceRegistrar interface by calling the provided registration function.
func (r *GenericServiceRegistrar) Register(transport interface{}) {
	if r.registerFunc != nil {
		r.registerFunc(transport)
	}
}

// Server represents an identity-aware HTTP server that handles automatic mTLS authentication.
// Services registered with this server will automatically use SPIFFE/SPIRE for identity verification.
// The server handles all certificate management and peer authentication transparently.
type Server interface {
	// RegisterService registers a service implementation with the server.
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

// Client represents an identity-aware HTTP client that handles automatic mTLS authentication.
// Connections made through this client will automatically use SPIFFE/SPIRE for identity verification.
// The client handles all certificate management and server authentication transparently.
type Client interface {
	// Connect establishes a connection to the specified service at the given address.
	// The serviceName is used for identity verification and must be non-empty.
	// The address should be in the format "host:port" and must be non-empty.
	// The context can be used for cancellation and timeouts.
	// Returns a ClientConnection that provides access to the underlying HTTP connection.
	// The caller must call Close() on the returned connection to release resources.
	Connect(ctx context.Context, serviceName, address string) (*ClientConnection, error)

	// Close releases any resources held by the client.
	// Should be called when the client is no longer needed.
	Close() error
}

// NewIdentityServer creates a new identity-aware server instance with automatic authentication enforcement.
//
// IDENTITY AUTHENTICATION ENFORCEMENT:
// This function sets up transport-layer authentication using SPIFFE/SPIRE X.509 certificates.
// Authentication is enforced automatically at the TLS handshake level, NOT in application code.
//
// How Authentication Works:
// 1. Server obtains its SPIFFE identity certificate from SPIRE (e.g., spiffe://example.org/echo-server)
// 2. All incoming connections MUST present valid client certificates with SPIFFE IDs
// 3. Server verifies client certificates against SPIRE trust bundle during TLS handshake
// 4. Only authenticated clients with valid certificates can establish connections
// 5. Unauthorized clients are rejected before any application code runs
//
// Configuration-Based Authorization:
// The server enforces service-level authorization via the 'authorized_clients' config:
//
//	authorized_clients:
//	  - "echo-client"      # ✅ This service can connect
//	  - "payment-service"  # ✅ This service can connect
//	  # Any other services are automatically rejected
//
// Security Guarantees:
// - X.509 certificate authentication (not API keys)
// - Mutual TLS (both client and server authenticate each other)
// - Short-lived certificates (1-hour expiration, auto-rotated by SPIRE)
// - Transport-layer enforcement (authentication happens before app logic)
// - Zero Trust model (every connection authenticated)
//
// Authentication Failure Behavior:
// When authentication fails, clients receive transport-layer errors:
// - "transport: authentication handshake failed"
// - "x509: certificate signed by unknown authority"
// - "rpc error: code = Unavailable desc = connection error"
// The server's application code (your service methods) is never executed.
//
// It reads configuration from the EPHEMOS_CONFIG environment variable or uses defaults.
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
//	registrar := ephemos.NewServiceRegistrar(func(s interface{}) {
//		// Register your HTTP handlers or other transport-specific services
//		if httpServer, ok := s.(YourHTTPServerInterface); ok {
//			httpServer.RegisterHandlers(&myService{})
//		}
//	})
//	server.RegisterService(ctx, registrar)
//	lis, _ := net.Listen("tcp", ":50051")
//	defer lis.Close()
//	server.Serve(ctx, lis)
func NewIdentityServer(ctx context.Context, configPath string) (Server, error) {
	if ctx == nil {
		return nil, &ValidationError{
			Field:   "context",
			Value:   nil,
			Message: "context cannot be nil",
		}
	}

	// EARLY CONFIG VALIDATION INTEGRATION:
	// Load and validate configuration transparently during API call.
	// This prevents partial setups and provides immediate feedback if config is invalid.
	// Developers get clear error messages without needing to call separate validation functions.
	config, err := loadAndValidateConfig(ctx, configPath)
	if err != nil {
		// Return domain-specific errors that developers can handle with standard Go error handling
		return nil, fmt.Errorf("server initialization failed: %w", err)
	}

	// Create server with validated configuration using internal factory
	server, err := createServerWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create identity server: %w", err)
	}

	return server, nil
}

// NewIdentityClient creates a new identity-aware client instance with automatic authentication.
//
// IDENTITY AUTHENTICATION ENFORCEMENT:
// This function sets up transport-layer authentication using SPIFFE/SPIRE X.509 certificates.
// The client automatically presents its identity certificate during TLS handshakes.
//
// How Client Authentication Works:
// 1. Client obtains its SPIFFE identity certificate from SPIRE (e.g., spiffe://example.org/echo-client)
// 2. During client.Connect(), client presents its certificate to the server
// 3. Server verifies client's certificate against SPIRE trust bundle
// 4. Client also verifies server's certificate for mutual authentication
// 5. Connection succeeds ONLY if both client and server authenticate successfully
//
// Authentication Enforcement Points:
// - client.Connect("service-name", "address") performs the mTLS handshake
// - If authentication fails, Connect() returns transport-layer errors
// - Application code never runs if authentication fails
//
// Client Identity Verification:
// The client verifies servers via configuration-based trust:
//
//	trusted_servers:
//	  - "echo-server"     # ✅ This client will connect to this server
//	  - "payment-api"     # ✅ This client will connect to this server
//	  # Connections to unlisted servers may be rejected
//
// Security Guarantees:
// - Client presents X.509 certificate (not API keys or tokens)
// - Mutual TLS authentication (client verifies server identity too)
// - Short-lived certificates (1-hour expiration, auto-rotated)
// - Transport-layer security (authentication happens before app requests)
// - Connection-level enforcement (failed auth = no connection)
//
// Authentication Failure Behavior:
// When authentication fails during Connect(), client receives errors like:
// - "transport: authentication handshake failed"
// - "connection error: desc = transport: Error while dialing"
// - "x509: certificate signed by unknown authority"
// - "rpc error: code = Unavailable desc = connection error"
// No application RPC calls are made if authentication fails.
//
// It reads configuration from the EPHEMOS_CONFIG environment variable or uses defaults.
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
//	serviceClient := yourservice.NewServiceClient(conn.GetClientConnection())
func NewIdentityClient(ctx context.Context, configPath string) (Client, error) {
	if ctx == nil {
		return nil, &ValidationError{
			Field:   "context",
			Value:   nil,
			Message: "context cannot be nil",
		}
	}

	// EARLY CONFIG VALIDATION INTEGRATION:
	// Load and validate configuration transparently during API call.
	// This prevents partial setups and provides immediate feedback if config is invalid.
	// Developers get clear error messages without needing to call separate validation functions.
	config, err := loadAndValidateConfig(ctx, configPath)
	if err != nil {
		// Return domain-specific errors that developers can handle with standard Go error handling
		return nil, fmt.Errorf("client initialization failed: %w", err)
	}

	// Create client with validated configuration using internal factory
	client, err := createClientWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create identity client: %w", err)
	}

	return client, nil
}

// Built-in Interceptors

// MetricsConfig configures metrics collection.
type MetricsConfig struct {
	AuthMetricsCollector interface{}
}

// InterceptorConfig configures server interceptors.
type InterceptorConfig struct {
	// EnableAuth enables authentication interceptor
	EnableAuth bool
	// EnableLogging enables audit logging interceptor
	EnableLogging bool
	// EnableMetrics enables metrics collection interceptor
	EnableMetrics bool
	// EnableIdentityPropagation enables identity propagation between services
	EnableIdentityPropagation bool
	// Logger for interceptor logging
	Logger *slog.Logger
	// MetricsConfig for metrics configuration
	MetricsConfig *MetricsConfig
	// CustomInterceptors allows adding custom interceptors
	CustomInterceptors []interface{}
}

// NewDefaultInterceptorConfig creates a default interceptor configuration.
func NewDefaultInterceptorConfig() *InterceptorConfig {
	return &InterceptorConfig{
		EnableAuth:                true,
		EnableLogging:             true,
		EnableMetrics:             true,
		EnableIdentityPropagation: false,
		Logger:                    slog.Default(),
		MetricsConfig:             &MetricsConfig{},
	}
}

// NewProductionInterceptorConfig creates a production-optimized interceptor configuration.
func NewProductionInterceptorConfig(serviceName string) *InterceptorConfig {
	logger := slog.Default().With("service", serviceName)
	return &InterceptorConfig{
		EnableAuth:                true,
		EnableLogging:             true,
		EnableMetrics:             true,
		EnableIdentityPropagation: true,
		Logger:                    logger,
		MetricsConfig:             &MetricsConfig{},
	}
}

// NewDevelopmentInterceptorConfig creates a development-friendly interceptor configuration.
func NewDevelopmentInterceptorConfig(serviceName string) *InterceptorConfig {
	logger := slog.Default().With("service", serviceName)
	return &InterceptorConfig{
		EnableAuth:                false, // Disabled for easier development
		EnableLogging:             true,
		EnableMetrics:             true,
		EnableIdentityPropagation: true, // Enabled for development testing
		Logger:                    logger,
		MetricsConfig:             &MetricsConfig{},
	}
}

// Transport-Agnostic API
//
// The following functions are defined in server.go and provide a transport-agnostic
// API where services are written with plain Go types and can run over HTTP
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
// Remov NewIdentityServer and change this implementation to be the new IdentityServer
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
