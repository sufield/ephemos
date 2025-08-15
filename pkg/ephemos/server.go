// Package ephemos provides a transport-agnostic service framework with hexagonal architecture.
// Services can run over HTTP or any future transport without code changes.
//
// Architecture: This file contains the transport server implementation, which handles
// protocol-specific concerns (HTTP) and delegates business logic to the identity server.
// For the production-ready identity server with graceful shutdown, see identity_server.go.
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
	// Internal adapters temporarily removed to eliminate dependencies
)

const (
	// TransportTypeHTTP represents HTTP transport type.
	TransportTypeHTTP = "http"
	// DefaultShutdownTimeout is the default timeout for graceful shutdown.
	DefaultShutdownTimeout = 30 * time.Second
	// DefaultHTTPAddress is the default address for HTTP transport.
	DefaultHTTPAddress = ":8080"
)

// TransportServer represents a transport-agnostic service server.
// The transport (HTTP, etc.) is determined by configuration.
type TransportServer struct {
	config *Configuration

	// Transport implementations - only one will be active
	httpServer  *http.Server
	httpMux     *http.ServeMux
	httpAdapter HTTPAdapter
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
	case TransportTypeHTTP:
		server.httpAdapter = NewHTTPAdapter(config.Transport.Address)
		server.httpServer = server.httpAdapter.GetHTTPServer()
		server.httpMux = http.NewServeMux()
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
	case TransportTypeHTTP:
		if s.httpAdapter == nil {
			return fmt.Errorf("HTTP adapter not initialized")
		}
		return s.httpAdapter.Mount(impl)
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
	case TransportTypeHTTP:
		return s.serveHTTP(ctx, shutdownCtx, addr)
	default:
		return fmt.Errorf("unsupported transport type: %s", s.config.Transport.Type)
	}
}

// serveOnListener serves on a pre-created listener without signal handling.
// This is used by the Server interface implementation.
func (s *TransportServer) serveOnListener(ctx context.Context, listener net.Listener) error {
	transportType := s.config.Transport.Type
	if transportType == "" {
		transportType = TransportTypeHTTP // default
	}

	switch transportType {
	case TransportTypeHTTP:
		return s.serveHTTPOnListener(ctx, listener)
	default:
		return fmt.Errorf("unsupported transport type: %s", transportType)
	}
}

// TODO: Refactor
func (s *TransportServer) resolveAddress() string {
	addr := s.config.Transport.Address
	if addr == "" {
		switch s.config.Transport.Type {
		case TransportTypeHTTP:
			addr = DefaultHTTPAddress
		default:
			addr = DefaultHTTPAddress // fallback to HTTP default
		}
	}
	return addr
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

// serveHTTPOnListener serves HTTP on a pre-created listener
func (s *TransportServer) serveHTTPOnListener(ctx context.Context, listener net.Listener) error {
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
		if err := s.httpServer.Serve(listener); err != nil && err != http.ErrServerClosed {
			setServerError(fmt.Errorf("HTTP server error: %w", err))
		}
	}()

	// Wait for context cancellation
	go func() {
		<-ctx.Done()

		// Graceful shutdown for HTTP server
		shutdownTimeout, shutdownCancel := context.WithTimeout(context.Background(), DefaultShutdownTimeout)
		defer shutdownCancel()

		if err := s.httpServer.Shutdown(shutdownTimeout); err != nil {
			setServerError(fmt.Errorf("HTTP server shutdown error: %w", err))
		}
	}()

	wg.Wait()

	errMutex.Lock()
	defer errMutex.Unlock()
	return serverErr
}

// Close gracefully shuts down the server.
func (s *TransportServer) Close() error {
	switch s.config.Transport.Type {
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
func loadConfig(ctx context.Context, path string) (*Configuration, error) {
	// Use existing configuration loading from config.go
	cfg, err := loadAndValidateConfig(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Set transport defaults if not specified
	if cfg.Transport.Type == "" {
		cfg.Transport.Type = TransportTypeHTTP
	}
	if cfg.Transport.Address == "" {
		switch cfg.Transport.Type {
		case TransportTypeHTTP:
			cfg.Transport.Address = DefaultHTTPAddress
		}
	}

	return cfg, nil
}
