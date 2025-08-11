// Package ephemos provides a transport-agnostic service framework with hexagonal architecture.
// Services can run over gRPC, HTTP, or any future transport without code changes.
package ephemos

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"google.golang.org/grpc"

	grpcAdapter "github.com/sufield/ephemos/internal/adapters/grpc"
	httpAdapter "github.com/sufield/ephemos/internal/adapters/http"
	"github.com/sufield/ephemos/internal/adapters/secondary/config"
	"github.com/sufield/ephemos/internal/core/ports"
)

const (
	// TransportTypeGRPC represents gRPC transport type.
	TransportTypeGRPC = "grpc"
	// TransportTypeHTTP represents HTTP transport type.
	TransportTypeHTTP = "http"
)

// TransportServer represents a transport-agnostic service server.
// The transport (gRPC, HTTP, etc.) is determined by configuration.
type TransportServer struct {
	config *ports.Configuration

	// Transport implementations - only one will be active
	grpcServer  *grpc.Server
	grpcAdapter *grpcAdapter.Adapter
	httpServer  *http.Server
	httpMux     *http.ServeMux
	httpAdapter *httpAdapter.Adapter
}

// newTransportServer creates a new server instance from configuration.
// The configuration determines the transport type and settings.
// This is an internal function - use the exported wrapper in ephemos.go.
func newTransportServer(ctx context.Context, configPath string) (*TransportServer, error) {
	// Load configuration (simplified for now)
	config, err := loadConfig(ctx, configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	server := &TransportServer{
		config: config,
	}

	// Initialize the appropriate transport based on config
	switch config.Transport.Type {
	case TransportTypeGRPC:
		server.grpcServer = grpc.NewServer()
		server.grpcAdapter = grpcAdapter.NewAdapter(server.grpcServer)
	case TransportTypeHTTP:
		server.httpMux = http.NewServeMux()
		server.httpAdapter = httpAdapter.NewAdapter(server.httpMux)
		server.httpServer = &http.Server{
			Handler: server.httpMux,
		}
	default:
		return nil, fmt.Errorf("unsupported transport type: %s", config.Transport.Type)
	}

	return server, nil
}

// mount registers a service implementation with the server.
// T should be an interface type that defines the service contract.
// impl should be a struct that implements T.
// This is an internal function - use the exported wrapper in ephemos.go.
func mount[T any](server *TransportServer, impl T) error {
	return server.mountService(impl)
}

// mountService handles the actual service registration based on transport type.
func (s *TransportServer) mountService(impl any) error {
	switch s.config.Transport.Type {
	case TransportTypeGRPC:
		if err := s.grpcAdapter.Mount(impl); err != nil {
			return fmt.Errorf("failed to mount gRPC service: %w", err)
		}
		return nil
	case TransportTypeHTTP:
		if err := s.httpAdapter.Mount(impl); err != nil {
			return fmt.Errorf("failed to mount HTTP service: %w", err)
		}
		return nil
	default:
		return fmt.Errorf("unsupported transport type: %s", s.config.Transport.Type)
	}
}

// ListenAndServe starts the server on the configured address and transport.
func (s *TransportServer) ListenAndServe(_ context.Context) error {
	addr := s.config.Transport.Address
	if addr == "" {
		addr = ":8080" // default
	}

	switch s.config.Transport.Type {
	case TransportTypeGRPC:
		lis, err := net.Listen("tcp", addr)
		if err != nil {
			return fmt.Errorf("failed to listen on %s: %w", addr, err)
		}
		defer func() {
			if closeErr := lis.Close(); closeErr != nil {
				// Log the error but don't override the main error
				// In a real implementation, this would use a logger
				_ = closeErr
			}
		}()

		if err := s.grpcServer.Serve(lis); err != nil {
			return fmt.Errorf("gRPC server error: %w", err)
		}
		return nil

	case TransportTypeHTTP:
		s.httpServer.Addr = addr
		if err := s.httpServer.ListenAndServe(); err != nil {
			return fmt.Errorf("HTTP server error: %w", err)
		}
		return nil

	default:
		return fmt.Errorf("unsupported transport type: %s", s.config.Transport.Type)
	}
}

// Close gracefully shuts down the server.
func (s *TransportServer) Close() error {
	switch s.config.Transport.Type {
	case TransportTypeGRPC:
		if s.grpcServer != nil {
			s.grpcServer.GracefulStop()
		}
	case TransportTypeHTTP:
		if s.httpServer != nil {
			if err := s.httpServer.Close(); err != nil {
				return fmt.Errorf("failed to close HTTP server: %w", err)
			}
		}
	}
	return nil
}

// loadConfig loads configuration from the specified path using the existing config system.
func loadConfig(ctx context.Context, path string) (*ports.Configuration, error) {
	// Use existing configuration loading
	configProvider := config.NewFileProvider()
	cfg, err := configProvider.LoadConfiguration(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Set transport defaults if not specified
	if cfg.Transport.Type == "" {
		cfg.Transport.Type = TransportTypeGRPC
	}
	if cfg.Transport.Address == "" {
		switch cfg.Transport.Type {
		case TransportTypeGRPC:
			cfg.Transport.Address = ":50051"
		case TransportTypeHTTP:
			cfg.Transport.Address = ":8080"
		}
	}

	return cfg, nil
}
