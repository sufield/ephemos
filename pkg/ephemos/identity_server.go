// Package ephemos provides an identity server with comprehensive graceful shutdown.
package ephemos

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

const (
	// DefaultReadyCheckInterval is the default polling interval for server ready checks.
	DefaultReadyCheckInterval = 100 * time.Millisecond
)

// IdentityOrchestrator provides a production-ready identity server with graceful shutdown.
type IdentityOrchestrator struct {
	baseServer          Server
	shutdownCoordinator *ShutdownCoordinator
	spiffeProvider      SPIFFEProvider
	config              *Configuration
	listeners           []net.Listener
	clients             []Client
	mu                  sync.RWMutex
	isRunning           bool
	shutdownChan        chan struct{}
}

// ServerOptions configures the identity server.
type ServerOptions struct {
	// Configuration for the server
	Config *Configuration

	// ConfigPath for loading configuration from file
	ConfigPath string

	// ShutdownConfig for graceful shutdown behavior
	ShutdownConfig *ShutdownConfig

	// EnableSignalHandling automatically handles OS signals for shutdown
	EnableSignalHandling bool

	// PreShutdownHook is called before shutdown starts
	PreShutdownHook func(ctx context.Context) error

	// PostShutdownHook is called after shutdown completes
	PostShutdownHook func(err error)
}

// initializeServerOptions ensures options are properly initialized.
func initializeServerOptions(opts *ServerOptions) *ServerOptions {
	if opts == nil {
		opts = &ServerOptions{
			ShutdownConfig:       DefaultShutdownConfig(),
			EnableSignalHandling: true,
		}
	}
	if opts.ShutdownConfig == nil {
		opts.ShutdownConfig = DefaultShutdownConfig()
	}
	return opts
}

// createBaseServer creates the base identity server.
func createBaseServer(ctx context.Context, opts *ServerOptions) (Server, error) {
	if opts.Config != nil {
		server, err := createServerWithConfig(ctx, opts.Config)
		if err != nil {
			return nil, fmt.Errorf("failed to create server with config: %w", err)
		}
		return server, nil
	}
	server, err := NewIdentityServer(ctx, opts.ConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create server from path: %w", err)
	}
	return server, nil
}

// createSPIFFEProvider creates a SPIFFE provider based on options.
func createSPIFFEProvider(opts *ServerOptions) (SPIFFEProvider, error) {
	var spiffeConfig *SPIFFEConfig
	if opts.Config != nil && opts.Config.SPIFFE != nil {
		spiffeConfig = opts.Config.SPIFFE
	}
	return NewSPIFFEProvider(spiffeConfig)
}

// setupShutdownHooks configures shutdown hooks.
func setupShutdownHooks(ctx context.Context, opts *ServerOptions) {
	if opts.PreShutdownHook != nil {
		originalStart := opts.ShutdownConfig.OnShutdownStart
		opts.ShutdownConfig.OnShutdownStart = func() {
			if err := opts.PreShutdownHook(ctx); err != nil {
				slog.Error("Pre-shutdown hook failed", "error", err)
			}
			if originalStart != nil {
				originalStart()
			}
		}
	}

	if opts.PostShutdownHook != nil {
		originalComplete := opts.ShutdownConfig.OnShutdownComplete
		opts.ShutdownConfig.OnShutdownComplete = func(err error) {
			opts.PostShutdownHook(err)
			if originalComplete != nil {
				originalComplete(err)
			}
		}
	}
}

// NewIdentityOrchestrator creates a production-ready identity server with graceful shutdown.
func NewIdentityOrchestrator(ctx context.Context, opts *ServerOptions) (*IdentityOrchestrator, error) {
	opts = initializeServerOptions(opts)

	// Create base server
	baseServer, err := createBaseServer(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create base server: %w", err)
	}

	// Create SPIFFE provider
	spiffeProvider, err := createSPIFFEProvider(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create SPIFFE provider: %w", err)
	}

	// Create and configure shutdown coordinator
	shutdownCoordinator := NewShutdownCoordinator(opts.ShutdownConfig)
	shutdownCoordinator.RegisterServer(baseServer)
	shutdownCoordinator.RegisterSPIFFEProvider(spiffeProvider)

	// Set up hooks
	setupShutdownHooks(ctx, opts)

	server := &IdentityOrchestrator{
		baseServer:          baseServer,
		shutdownCoordinator: shutdownCoordinator,
		spiffeProvider:      spiffeProvider,
		config:              opts.Config,
		listeners:           make([]net.Listener, 0),
		clients:             make([]Client, 0),
		shutdownChan:        make(chan struct{}),
	}

	// Set up signal handling if enabled
	if opts.EnableSignalHandling {
		server.setupSignalHandling(ctx)
	}

	return server, nil
}

// RegisterService delegates to the base server.
func (s *IdentityOrchestrator) RegisterService(ctx context.Context, serviceRegistrar ServiceRegistrar) error {
	if err := s.baseServer.RegisterService(ctx, serviceRegistrar); err != nil {
		return fmt.Errorf("failed to register service: %w", err)
	}
	return nil
}

// Serve starts the server with graceful shutdown support.
func (s *IdentityOrchestrator) Serve(ctx context.Context, listener net.Listener) error {
	s.mu.Lock()
	s.isRunning = true
	s.listeners = append(s.listeners, listener)
	s.shutdownCoordinator.RegisterListener(&netListenerAdapter{listener})
	s.mu.Unlock()

	// Create a context that will be canceled on shutdown
	serveCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Start server in goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- s.baseServer.Serve(serveCtx, listener)
	}()

	// Wait for either error or shutdown signal
	select {
	case err := <-errChan:
		return err
	case <-s.shutdownChan:
		// Shutdown initiated
		cancel() // Cancel the serve context
		return s.performShutdown(ctx)
	case <-ctx.Done():
		// Context canceled externally - derive new timeout from parent context for cleanup
		shutdownCtx, cancel := context.WithTimeout(ctx, DefaultShutdownTimeout)
		defer cancel()
		return s.performShutdown(shutdownCtx)
	}
}

// ServeWithDeadline starts the server with a specific deadline for operation.
func (s *IdentityOrchestrator) ServeWithDeadline(ctx context.Context, listener net.Listener, deadline time.Time) error {
	deadlineCtx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()

	return s.Serve(deadlineCtx, listener)
}

// Close initiates graceful shutdown.
func (s *IdentityOrchestrator) Close() error {
	return s.Shutdown(context.Background())
}

// Shutdown performs graceful shutdown with context deadline support.
func (s *IdentityOrchestrator) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	if !s.isRunning {
		s.mu.Unlock()
		return nil
	}
	s.isRunning = false
	s.mu.Unlock()

	// Signal shutdown to Serve method
	close(s.shutdownChan)

	// Perform shutdown
	return s.performShutdown(ctx)
}

func (s *IdentityOrchestrator) performShutdown(ctx context.Context) error {
	slog.Info("Initiating graceful shutdown")

	// Create shutdown context with deadline if not already set
	shutdownCtx := ctx
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		shutdownCtx, cancel = context.WithTimeout(ctx, s.shutdownCoordinator.config.ForceTimeout)
		defer cancel()
	}

	// Perform the shutdown
	return s.shutdownCoordinator.Shutdown(shutdownCtx)
}

func (s *IdentityOrchestrator) setupSignalHandling(ctx context.Context) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		slog.Info("Received shutdown signal", "signal", sig)

		// Create context with timeout for shutdown, derived from parent context
		shutdownCtx, cancel := context.WithTimeout(ctx, s.shutdownCoordinator.config.ForceTimeout)
		defer cancel()

		if err := s.Shutdown(shutdownCtx); err != nil {
			slog.Error("Shutdown failed", "error", err)
			os.Exit(1)
		}

		os.Exit(0)
	}()
}

// RegisterClient registers a client for cleanup during shutdown.
func (s *IdentityOrchestrator) RegisterClient(client Client) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.clients = append(s.clients, client)
	s.shutdownCoordinator.RegisterClient(client)
}

// RegisterCleanupFunc registers a custom cleanup function.
func (s *IdentityOrchestrator) RegisterCleanupFunc(fn func() error) {
	s.shutdownCoordinator.RegisterCleanupFunc(fn)
}

// GetSPIFFEProvider returns the server's SPIFFE provider.
func (s *IdentityOrchestrator) GetSPIFFEProvider() SPIFFEProvider {
	return s.spiffeProvider
}

// IsRunning returns whether the server is currently running.
func (s *IdentityOrchestrator) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.isRunning
}

// netListenerAdapter adapts net.Listener to ports.Listener.
type netListenerAdapter struct {
	listener net.Listener
}

func (a *netListenerAdapter) Accept() (net.Conn, error) {
	conn, err := a.listener.Accept()
	if err != nil {
		return nil, fmt.Errorf("listener accept failed: %w", err)
	}
	return conn, nil
}

func (a *netListenerAdapter) Close() error {
	if err := a.listener.Close(); err != nil {
		return fmt.Errorf("listener close failed: %w", err)
	}
	return nil
}

func (a *netListenerAdapter) Addr() net.Addr {
	return a.listener.Addr()
}

// WaitForReady waits for the server to be ready to accept connections.
func (s *IdentityOrchestrator) WaitForReady(ctx context.Context) error {
	ticker := time.NewTicker(DefaultReadyCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for server ready: %w", ctx.Err())
		case <-ticker.C:
			s.mu.RLock()
			running := s.isRunning
			s.mu.RUnlock()

			if running {
				// Server is running, check if it can accept connections
				if len(s.listeners) > 0 {
					return nil
				}
			}
		}
	}
}
