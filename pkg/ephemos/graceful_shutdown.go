// Package ephemos provides graceful shutdown capabilities for server resources.
package ephemos

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/sufield/ephemos/internal/adapters/primary/api"
	"github.com/sufield/ephemos/internal/adapters/secondary/spiffe"
	"github.com/sufield/ephemos/internal/core/ports"
)

const (
	// DefaultGracePeriod is the default maximum time to wait for graceful shutdown.
	DefaultGracePeriod = 30 * time.Second
	// DefaultDrainTimeout is the default time to wait for existing requests to complete.
	DefaultDrainTimeout = 20 * time.Second
	// DefaultForceTimeout is the default time after which forceful shutdown occurs.
	DefaultForceTimeout = 45 * time.Second
)

// ShutdownConfig configures graceful shutdown behavior.
type ShutdownConfig struct {
	// GracePeriod is the maximum time to wait for graceful shutdown.
	// Default is 30 seconds if not specified.
	GracePeriod time.Duration

	// DrainTimeout is the time to wait for existing requests to complete.
	// Default is 20 seconds if not specified.
	DrainTimeout time.Duration

	// ForceTimeout is the time after which forceful shutdown occurs.
	// Default is 45 seconds if not specified.
	ForceTimeout time.Duration

	// OnShutdownStart is called when shutdown begins.
	OnShutdownStart func()

	// OnShutdownComplete is called when shutdown completes.
	OnShutdownComplete func(err error)
}

// DefaultShutdownConfig returns sensible shutdown defaults.
func DefaultShutdownConfig() *ShutdownConfig {
	return &ShutdownConfig{
		GracePeriod:  DefaultGracePeriod,
		DrainTimeout: DefaultDrainTimeout,
		ForceTimeout: DefaultForceTimeout,
	}
}

// GracefulShutdownManager coordinates shutdown of all server resources.
type GracefulShutdownManager struct {
	config          *ShutdownConfig
	servers         []ShutdownableServer
	spiffeProviders []*spiffe.Provider
	clients         []ports.Client
	listeners       []ports.Listener
	cleanupFuncs    []func() error
	mu              sync.Mutex
	shutdownOnce    sync.Once
	isShuttingDown  bool
}

// ShutdownableServer represents a server that can be gracefully stopped.
type ShutdownableServer interface {
	Close() error
}

// NewGracefulShutdownManager creates a new shutdown manager.
func NewGracefulShutdownManager(config *ShutdownConfig) *GracefulShutdownManager {
	if config == nil {
		config = DefaultShutdownConfig()
	}
	return &GracefulShutdownManager{
		config:          config,
		servers:         make([]ShutdownableServer, 0),
		spiffeProviders: make([]*spiffe.Provider, 0),
		clients:         make([]ports.Client, 0),
		listeners:       make([]ports.Listener, 0),
		cleanupFuncs:    make([]func() error, 0),
	}
}

// RegisterServer registers a server for graceful shutdown.
func (m *GracefulShutdownManager) RegisterServer(server ShutdownableServer) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if server != nil && !m.isShuttingDown {
		m.servers = append(m.servers, server)
	}
}

// RegisterSPIFFEProvider registers a SPIFFE provider for cleanup.
func (m *GracefulShutdownManager) RegisterSPIFFEProvider(provider *spiffe.Provider) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if provider != nil && !m.isShuttingDown {
		m.spiffeProviders = append(m.spiffeProviders, provider)
	}
}

// RegisterClient registers a client for cleanup.
func (m *GracefulShutdownManager) RegisterClient(client ports.Client) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if client != nil && !m.isShuttingDown {
		m.clients = append(m.clients, client)
	}
}

// RegisterListener registers a listener for cleanup.
func (m *GracefulShutdownManager) RegisterListener(listener ports.Listener) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if listener != nil && !m.isShuttingDown {
		m.listeners = append(m.listeners, listener)
	}
}

// RegisterCleanupFunc registers a custom cleanup function.
func (m *GracefulShutdownManager) RegisterCleanupFunc(fn func() error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if fn != nil && !m.isShuttingDown {
		m.cleanupFuncs = append(m.cleanupFuncs, fn)
	}
}

// Shutdown performs graceful shutdown of all registered resources.
func (m *GracefulShutdownManager) Shutdown(ctx context.Context) error {
	var finalErr error

	m.shutdownOnce.Do(func() {
		m.mu.Lock()
		m.isShuttingDown = true
		m.mu.Unlock()

		// Call shutdown start callback
		if m.config.OnShutdownStart != nil {
			m.config.OnShutdownStart()
		}

		// Create contexts with timeouts
		graceCtx, graceCancel := context.WithTimeout(ctx, m.config.GracePeriod)
		defer graceCancel()

		drainCtx, drainCancel := context.WithTimeout(ctx, m.config.DrainTimeout)
		defer drainCancel()

		forceCtx, forceCancel := context.WithTimeout(ctx, m.config.ForceTimeout)
		defer forceCancel()

		// Perform shutdown in phases
		finalErr = m.performShutdown(graceCtx, drainCtx, forceCtx)

		// Call shutdown complete callback
		if m.config.OnShutdownComplete != nil {
			m.config.OnShutdownComplete(finalErr)
		}
	})

	return finalErr
}

func (m *GracefulShutdownManager) shutdownServers(graceCtx context.Context, wg *sync.WaitGroup, addError func(error)) {
	slog.Info("Phase 1: Stopping servers gracefully")
	for _, server := range m.servers {
		wg.Add(1)
		go func(s ShutdownableServer) {
			defer wg.Done()
			if err := m.stopServerWithTimeout(graceCtx, s); err != nil {
				addError(fmt.Errorf("server stop error: %w", err))
			}
		}(server)
	}
}

func (m *GracefulShutdownManager) shutdownListeners(wg *sync.WaitGroup, addError func(error)) {
	slog.Info("Phase 2: Closing listeners")
	for _, listener := range m.listeners {
		wg.Add(1)
		go func(l ports.Listener) {
			defer wg.Done()
			if err := l.Close(); err != nil {
				addError(fmt.Errorf("listener close error: %w", err))
			}
		}(listener)
	}
}

func (m *GracefulShutdownManager) waitForShutdown(timeoutCtx context.Context, wg *sync.WaitGroup, successMsg, warnMsg string) bool {
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		slog.Info(successMsg)
		return true
	case <-timeoutCtx.Done():
		slog.Warn(warnMsg)
		return false
	}
}

func (m *GracefulShutdownManager) shutdownClients(addError func(error)) {
	slog.Info("Phase 3: Closing clients and connections")
	var clientWg sync.WaitGroup
	for _, client := range m.clients {
		clientWg.Add(1)
		go func(c ports.Client) {
			defer clientWg.Done()
			if err := c.Close(); err != nil {
				addError(fmt.Errorf("client close error: %w", err))
			}
		}(client)
	}
	clientWg.Wait()
}

func (m *GracefulShutdownManager) shutdownSpiffeProviders(graceCtx, forceCtx context.Context, addError func(error)) {
	slog.Info("Phase 4: Closing SPIFFE providers and SVID watchers")
	var spiffeWg sync.WaitGroup
	for _, provider := range m.spiffeProviders {
		spiffeWg.Add(1)
		go func(p *spiffe.Provider) {
			defer spiffeWg.Done()
			if err := m.closeSPIFFEProviderWithTimeout(graceCtx, p); err != nil {
				addError(fmt.Errorf("SPIFFE provider close error: %w", err))
			}
		}(provider)
	}
	success := m.waitForShutdown(forceCtx, &spiffeWg, "SPIFFE providers closed successfully", "Force timeout exceeded during SPIFFE cleanup")
	if !success {
		addError(fmt.Errorf("force timeout during SPIFFE cleanup"))
	}
}

func (m *GracefulShutdownManager) runCleanupFunctions(addError func(error)) {
	slog.Info("Phase 5: Running cleanup functions")
	for _, fn := range m.cleanupFuncs {
		if err := fn(); err != nil {
			addError(fmt.Errorf("cleanup function error: %w", err))
		}
	}
}

func (m *GracefulShutdownManager) performShutdown(graceCtx, drainCtx, forceCtx context.Context) error {
	var wg sync.WaitGroup
	var errMutex sync.Mutex
	var collectedErrors []error
	// Helper to safely add errors
	addError := func(err error) {
		if err != nil {
			errMutex.Lock()
			collectedErrors = append(collectedErrors, err)
			errMutex.Unlock()
		}
	}
	slog.Info("Starting graceful shutdown",
		"grace_period", m.config.GracePeriod,
		"drain_timeout", m.config.DrainTimeout)
	m.shutdownServers(graceCtx, &wg, addError)
	m.shutdownListeners(&wg, addError)
	// Wait for Phase 1 & 2 with drain timeout (no error added on timeout, per original logic)
	m.waitForShutdown(drainCtx, &wg, "Servers and listeners stopped successfully", "Drain timeout exceeded, continuing shutdown")
	m.shutdownClients(addError)
	m.shutdownSpiffeProviders(graceCtx, forceCtx, addError)
	m.runCleanupFunctions(addError)
	// Return collected errors
	errMutex.Lock()
	defer errMutex.Unlock()
	for _, err := range collectedErrors {
		slog.Error("Shutdown error", "error", err)
	}
	if len(collectedErrors) > 0 {
		return fmt.Errorf("shutdown completed with %d errors: %v", len(collectedErrors), collectedErrors)
	}
	slog.Info("Graceful shutdown completed successfully")
	return nil
}

func (m *GracefulShutdownManager) stopServerWithTimeout(ctx context.Context, server ShutdownableServer) error {
	done := make(chan error, 1)

	go func() {
		done <- server.Close()
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return fmt.Errorf("server stop timeout: %w", ctx.Err())
	}
}

func (m *GracefulShutdownManager) closeSPIFFEProviderWithTimeout(ctx context.Context, provider *spiffe.Provider) error {
	done := make(chan error, 1)

	go func() {
		// Close the provider which will close the X509Source and stop SVID watchers
		done <- provider.Close()
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return fmt.Errorf("SPIFFE provider close timeout: %w", ctx.Err())
	}
}

// ExtendedIdentityServer wraps the standard IdentityServer with graceful shutdown support.
type ExtendedIdentityServer struct {
	*api.IdentityServer
	shutdownManager *GracefulShutdownManager
	spiffeProvider  *spiffe.Provider
}

// NewExtendedIdentityServer creates an identity server with graceful shutdown support.
func NewExtendedIdentityServer(ctx context.Context, configPath string, shutdownConfig *ShutdownConfig) (*ExtendedIdentityServer, error) {
	// Create the base server
	baseServer, err := api.NewIdentityServer(ctx, configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create identity server: %w", err)
	}

	// Create shutdown manager
	manager := NewGracefulShutdownManager(shutdownConfig)

	// Create extended server
	extServer := &ExtendedIdentityServer{
		IdentityServer:  baseServer,
		shutdownManager: manager,
	}

	// Register the server for shutdown
	manager.RegisterServer(baseServer)

	return extServer, nil
}

// SetSPIFFEProvider sets the SPIFFE provider for cleanup during shutdown.
func (s *ExtendedIdentityServer) SetSPIFFEProvider(provider *spiffe.Provider) {
	s.spiffeProvider = provider
	s.shutdownManager.RegisterSPIFFEProvider(provider)
}

// GracefulShutdown performs graceful shutdown with context deadline.
func (s *ExtendedIdentityServer) GracefulShutdown(ctx context.Context) error {
	return s.shutdownManager.Shutdown(ctx)
}

// RegisterCleanupFunc adds a custom cleanup function.
func (s *ExtendedIdentityServer) RegisterCleanupFunc(fn func() error) {
	s.shutdownManager.RegisterCleanupFunc(fn)
}
