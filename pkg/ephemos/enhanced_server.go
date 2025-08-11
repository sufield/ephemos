// Package ephemos provides an enhanced server with comprehensive graceful shutdown.
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

	"github.com/sufield/ephemos/internal/adapters/primary/api"
	"github.com/sufield/ephemos/internal/adapters/secondary/spiffe"
	"github.com/sufield/ephemos/internal/core/ports"
)

// EnhancedServer provides a production-ready server with graceful shutdown.
type EnhancedServer struct {
	baseServer      Server
	shutdownManager *GracefulShutdownManager
	spiffeProvider  *spiffe.Provider
	config          *ports.Configuration
	listeners       []net.Listener
	clients         []ports.Client
	mu              sync.RWMutex
	isRunning       bool
	shutdownChan    chan struct{}
}

// ServerOptions configures the enhanced server.
type ServerOptions struct {
	// Configuration for the server
	Config *ports.Configuration
	
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

// NewEnhancedIdentityServer creates a production-ready identity server with graceful shutdown.
func NewEnhancedIdentityServer(ctx context.Context, opts *ServerOptions) (*EnhancedServer, error) {
	if opts == nil {
		opts = &ServerOptions{
			ShutdownConfig:       DefaultShutdownConfig(),
			EnableSignalHandling: true,
		}
	}
	
	if opts.ShutdownConfig == nil {
		opts.ShutdownConfig = DefaultShutdownConfig()
	}
	
	// Create base server
	var baseServer *api.IdentityServer
	var err error
	
	if opts.Config != nil {
		baseServer, err = api.NewIdentityServerWithConfig(ctx, opts.Config)
	} else {
		baseServer, err = api.NewIdentityServer(ctx, opts.ConfigPath)
	}
	
	if err != nil {
		return nil, fmt.Errorf("failed to create base server: %w", err)
	}
	
	// Create SPIFFE provider for the server
	var spiffeConfig *ports.SPIFFEConfig
	if opts.Config != nil {
		spiffeConfig = opts.Config.SPIFFE
	}
	
	spiffeProvider, err := spiffe.NewProvider(spiffeConfig)
	if err != nil {
		slog.Warn("Failed to create SPIFFE provider", "error", err)
		// Continue without SPIFFE - server might not need it
	}
	
	// Create shutdown manager
	shutdownManager := NewGracefulShutdownManager(opts.ShutdownConfig)
	
	// Register the base server for shutdown
	shutdownManager.RegisterServer(baseServer)
	
	// Register SPIFFE provider if available
	if spiffeProvider != nil {
		shutdownManager.RegisterSPIFFEProvider(spiffeProvider)
	}
	
	// Set up hooks
	if opts.PreShutdownHook != nil {
		originalStart := opts.ShutdownConfig.OnShutdownStart
		opts.ShutdownConfig.OnShutdownStart = func() {
			if err := opts.PreShutdownHook(context.Background()); err != nil {
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
	
	server := &EnhancedServer{
		baseServer:      baseServer,
		shutdownManager: shutdownManager,
		spiffeProvider:  spiffeProvider,
		config:          opts.Config,
		listeners:       make([]net.Listener, 0),
		clients:         make([]ports.Client, 0),
		shutdownChan:    make(chan struct{}),
	}
	
	// Set up signal handling if enabled
	if opts.EnableSignalHandling {
		server.setupSignalHandling()
	}
	
	return server, nil
}

// RegisterService delegates to the base server.
func (s *EnhancedServer) RegisterService(ctx context.Context, serviceRegistrar ServiceRegistrar) error {
	return s.baseServer.RegisterService(ctx, serviceRegistrar)
}

// Serve starts the server with graceful shutdown support.
func (s *EnhancedServer) Serve(ctx context.Context, listener net.Listener) error {
	s.mu.Lock()
	s.isRunning = true
	s.listeners = append(s.listeners, listener)
	s.shutdownManager.RegisterListener(&netListenerAdapter{listener})
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
		// Context canceled externally
		return s.performShutdown(context.Background())
	}
}

// ServeWithDeadline starts the server with a specific deadline for operation.
func (s *EnhancedServer) ServeWithDeadline(ctx context.Context, listener net.Listener, deadline time.Time) error {
	deadlineCtx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()
	
	return s.Serve(deadlineCtx, listener)
}

// Close initiates graceful shutdown.
func (s *EnhancedServer) Close() error {
	return s.Shutdown(context.Background())
}

// Shutdown performs graceful shutdown with context deadline support.
func (s *EnhancedServer) Shutdown(ctx context.Context) error {
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

func (s *EnhancedServer) performShutdown(ctx context.Context) error {
	slog.Info("Initiating graceful shutdown")
	
	// Create shutdown context with deadline if not already set
	shutdownCtx := ctx
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		shutdownCtx, cancel = context.WithTimeout(ctx, s.shutdownManager.config.ForceTimeout)
		defer cancel()
	}
	
	// Perform the shutdown
	return s.shutdownManager.Shutdown(shutdownCtx)
}

func (s *EnhancedServer) setupSignalHandling() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	
	go func() {
		sig := <-sigChan
		slog.Info("Received shutdown signal", "signal", sig)
		
		// Create context with timeout for shutdown
		ctx, cancel := context.WithTimeout(context.Background(), s.shutdownManager.config.ForceTimeout)
		defer cancel()
		
		if err := s.Shutdown(ctx); err != nil {
			slog.Error("Shutdown failed", "error", err)
			os.Exit(1)
		}
		
		os.Exit(0)
	}()
}

// RegisterClient registers a client for cleanup during shutdown.
func (s *EnhancedServer) RegisterClient(client ports.Client) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.clients = append(s.clients, client)
	s.shutdownManager.RegisterClient(client)
}

// RegisterCleanupFunc registers a custom cleanup function.
func (s *EnhancedServer) RegisterCleanupFunc(fn func() error) {
	s.shutdownManager.RegisterCleanupFunc(fn)
}

// GetSPIFFEProvider returns the SPIFFE provider if available.
func (s *EnhancedServer) GetSPIFFEProvider() *spiffe.Provider {
	return s.spiffeProvider
}

// IsRunning returns whether the server is currently running.
func (s *EnhancedServer) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.isRunning
}

// netListenerAdapter adapts net.Listener to ports.Listener.
type netListenerAdapter struct {
	listener net.Listener
}

func (a *netListenerAdapter) Accept() (interface{}, error) {
	return a.listener.Accept()
}

func (a *netListenerAdapter) Close() error {
	return a.listener.Close()
}

func (a *netListenerAdapter) Addr() string {
	return a.listener.Addr().String()
}

// WaitForReady waits for the server to be ready to accept connections.
func (s *EnhancedServer) WaitForReady(ctx context.Context) error {
	ticker := time.NewTicker(100 * time.Millisecond)
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