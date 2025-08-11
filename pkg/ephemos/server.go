// Package ephemos provides a transport-agnostic service framework with hexagonal architecture.
// Services can run over gRPC, HTTP, or any future transport without code changes.
package ephemos

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

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
	// DefaultShutdownTimeout is the default timeout for graceful shutdown.
	DefaultShutdownTimeout = 30 * time.Second
	// DefaultHTTPAddress is the default address for HTTP transport.
	DefaultHTTPAddress = ":8080"
	// DefaultGRPCAddress is the default address for gRPC transport.
	DefaultGRPCAddress = ":50051"
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

// ListenAndServe starts the server on the configured address and transport with graceful shutdown.
// It handles SIGINT and SIGTERM signals automatically for graceful shutdown.
func (s *TransportServer) ListenAndServe(ctx context.Context) error {
	addr := s.resolveAddress()

	// Create a context that will be canceled on receiving shutdown signals
	shutdownCtx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	switch s.config.Transport.Type {
	case TransportTypeGRPC:
		return s.serveGRPC(ctx, shutdownCtx, addr)
	case TransportTypeHTTP:
		return s.serveHTTP(ctx, shutdownCtx, addr)
	default:
		return fmt.Errorf("unsupported transport type: %s", s.config.Transport.Type)
	}
}

func (s *TransportServer) resolveAddress() string {
	addr := s.config.Transport.Address
	if addr == "" {
		switch s.config.Transport.Type {
		case TransportTypeGRPC:
			addr = DefaultGRPCAddress
		case TransportTypeHTTP:
			addr = DefaultHTTPAddress
		default:
			addr = DefaultGRPCAddress // fallback to gRPC default
		}
	}
	return addr
}

func (s *TransportServer) serveGRPC(_, shutdownCtx context.Context, addr string) error {
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

	var wg sync.WaitGroup
	var serverErr error
	var errMutex sync.Mutex

	setServerError := func(err error) {
		errMutex.Lock()
		defer errMutex.Unlock()
		if serverErr == nil && err != nil {
			serverErr = err
		}
	}

	// Start gRPC server in a goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := s.grpcServer.Serve(lis); err != nil {
			setServerError(fmt.Errorf("gRPC server error: %w", err))
		}
	}()

	// Wait for shutdown signal
	<-shutdownCtx.Done()

	// Graceful shutdown for gRPC server
	s.grpcServer.GracefulStop()
	wg.Wait()

	errMutex.Lock()
	defer errMutex.Unlock()
	return serverErr
}

func (s *TransportServer) serveHTTP(ctx, shutdownCtx context.Context, addr string) error {
	s.httpServer.Addr = addr

	var wg sync.WaitGroup
	var serverErr error
	var errMutex sync.Mutex

	setServerError := func(err error) {
		errMutex.Lock()
		defer errMutex.Unlock()
		if serverErr == nil && err != nil {
			serverErr = err
		}
	}

	// Start HTTP server in a goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			setServerError(fmt.Errorf("HTTP server error: %w", err))
		}
	}()

	// Wait for shutdown signal
	<-shutdownCtx.Done()

	// Graceful shutdown for HTTP server
	shutdownTimeout, shutdownCancel := context.WithTimeout(ctx, DefaultShutdownTimeout)
	defer shutdownCancel()

	if err := s.httpServer.Shutdown(shutdownTimeout); err != nil {
		setServerError(fmt.Errorf("HTTP server shutdown error: %w", err))
	}
	wg.Wait()

	errMutex.Lock()
	defer errMutex.Unlock()
	return serverErr
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
			cfg.Transport.Address = DefaultGRPCAddress
		case TransportTypeHTTP:
			cfg.Transport.Address = DefaultHTTPAddress
		}
	}

	return cfg, nil
}
