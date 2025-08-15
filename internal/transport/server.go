// Package transport provides transport-agnostic service framework implementation.
package transport

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
)

const (
	// TypeHTTP represents HTTP transport type.
	TypeHTTP = "http"
	// DefaultShutdownTimeout is the default timeout for graceful shutdown.
	DefaultShutdownTimeout = 30 * time.Second
	// DefaultHTTPAddress is the default address for HTTP transport.
	DefaultHTTPAddress = ":8080"
)

// Configuration interface for transport layer (imported from public API)
type Configuration interface {
	GetTransportType() string
	GetTransportAddress() string
}

// Server represents a transport-agnostic service server.
type Server struct {
	config Configuration

	// Transport implementations - only one will be active
	httpServer  *http.Server
	httpMux     *http.ServeMux
	httpAdapter HTTPAdapter
}

// NewServer creates a new server instance from configuration.
func NewServer(config Configuration) (*Server, error) {
	server := &Server{
		config: config,
	}

	// Initialize the appropriate transport based on config
	switch config.GetTransportType() {
	case TypeHTTP:
		server.httpAdapter = NewHTTPAdapter(config.GetTransportAddress())
		server.httpServer = server.httpAdapter.GetHTTPServer()
		server.httpMux = http.NewServeMux()
	default:
		return nil, fmt.Errorf("unsupported transport type: %s", config.GetTransportType())
	}

	return server, nil
}

// Mount registers a service implementation with the server.
func (s *Server) Mount(impl any) error {
	switch s.config.GetTransportType() {
	case TypeHTTP:
		if s.httpAdapter == nil {
			return fmt.Errorf("HTTP adapter not initialized")
		}
		return s.httpAdapter.Mount(impl)
	default:
		return fmt.Errorf("unsupported transport type: %s", s.config.GetTransportType())
	}
}

// ListenAndServe starts the server on the configured address and transport with graceful shutdown.
func (s *Server) ListenAndServe(ctx context.Context) error {
	addr := s.resolveAddress()

	// Create a context that will be canceled on receiving shutdown signals
	shutdownCtx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	switch s.config.GetTransportType() {
	case TypeHTTP:
		return s.serveHTTP(ctx, shutdownCtx, addr)
	default:
		return fmt.Errorf("unsupported transport type: %s", s.config.GetTransportType())
	}
}

// ServeOnListener serves on a pre-created listener without signal handling.
func (s *Server) ServeOnListener(ctx context.Context, listener net.Listener) error {
	transportType := s.config.GetTransportType()
	if transportType == "" {
		transportType = TypeHTTP // default
	}

	switch transportType {
	case TypeHTTP:
		return s.serveHTTPOnListener(ctx, listener)
	default:
		return fmt.Errorf("unsupported transport type: %s", transportType)
	}
}

func (s *Server) resolveAddress() string {
	addr := s.config.GetTransportAddress()
	if addr == "" {
		switch s.config.GetTransportType() {
		case TypeHTTP:
			addr = DefaultHTTPAddress
		default:
			addr = DefaultHTTPAddress // fallback to HTTP default
		}
	}
	return addr
}

func (s *Server) serveHTTP(ctx, shutdownCtx context.Context, addr string) error {
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
func (s *Server) serveHTTPOnListener(ctx context.Context, listener net.Listener) error {
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
func (s *Server) Close() error {
	switch s.config.GetTransportType() {
	case TypeHTTP:
		if s.httpServer != nil {
			if err := s.httpServer.Close(); err != nil {
				return fmt.Errorf("failed to close HTTP server: %w", err)
			}
		}
	}
	return nil
}