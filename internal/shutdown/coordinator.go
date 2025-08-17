// Package shutdown provides internal shutdown coordination and lifecycle management.
package shutdown

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

const (
	// DefaultGracePeriod is the default maximum time to wait for graceful shutdown.
	DefaultGracePeriod = 30 * time.Second
	// DefaultDrainTimeout is the default time to wait for existing requests to complete.
	DefaultDrainTimeout = 20 * time.Second
	// DefaultForceTimeout is the default time after which forceful shutdown occurs.
	DefaultForceTimeout = 45 * time.Second
)

// Config configures graceful shutdown behavior.
type Config struct {
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

// DefaultConfig returns sensible shutdown defaults.
func DefaultConfig() *Config {
	return &Config{
		GracePeriod:  DefaultGracePeriod,
		DrainTimeout: DefaultDrainTimeout,
		ForceTimeout: DefaultForceTimeout,
	}
}

// Coordinator coordinates shutdown of all server resources.
type Coordinator struct {
	config         *Config
	servers        []Server
	clients        []Client
	listeners      []Listener
	cleanupFuncs   []func() error
	mu             sync.Mutex
	shutdownOnce   sync.Once
	isShuttingDown bool
}

// Server represents a server that can be gracefully stopped.
type Server interface {
	Close() error
}

// Client represents a client that can be gracefully closed.
type Client interface {
	Close() error
}

// Listener represents a network listener that can be closed.
type Listener interface {
	Close() error
}

// NewCoordinator creates a new shutdown coordinator.
func NewCoordinator(config *Config) *Coordinator {
	if config == nil {
		config = DefaultConfig()
	}
	return &Coordinator{
		config:       config,
		servers:      make([]Server, 0),
		clients:      make([]Client, 0),
		listeners:    make([]Listener, 0),
		cleanupFuncs: make([]func() error, 0),
	}
}

// RegisterServer registers a server for graceful shutdown.
func (c *Coordinator) RegisterServer(server Server) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if server != nil && !c.isShuttingDown {
		c.servers = append(c.servers, server)
	}
}

// RegisterClient registers a client for graceful shutdown.
func (c *Coordinator) RegisterClient(client Client) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if client != nil && !c.isShuttingDown {
		c.clients = append(c.clients, client)
	}
}

// RegisterListener registers a listener for graceful shutdown.
func (c *Coordinator) RegisterListener(listener Listener) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if listener != nil && !c.isShuttingDown {
		c.listeners = append(c.listeners, listener)
	}
}

// RegisterCleanupFunc registers a cleanup function to run during shutdown.
func (c *Coordinator) RegisterCleanupFunc(fn func() error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if fn != nil && !c.isShuttingDown {
		c.cleanupFuncs = append(c.cleanupFuncs, fn)
	}
}

// Shutdown performs graceful shutdown of all registered resources.
func (c *Coordinator) Shutdown(ctx context.Context) error {
	var finalErr error

	c.shutdownOnce.Do(func() {
		c.mu.Lock()
		c.isShuttingDown = true
		c.mu.Unlock()

		if c.config.OnShutdownStart != nil {
			c.config.OnShutdownStart()
		}

		graceCtx, graceCancel := context.WithTimeout(ctx, c.config.GracePeriod)
		defer graceCancel()

		forceCtx, forceCancel := context.WithTimeout(ctx, c.config.ForceTimeout)
		defer forceCancel()

		slog.Info("Starting graceful shutdown",
			"grace_period", c.config.GracePeriod,
			"drain_timeout", c.config.DrainTimeout)

		var errors []error
		addError := func(err error) {
			errors = append(errors, err)
		}

		// Phase 1: Stop servers gracefully
		c.shutdownServers(graceCtx, forceCtx, addError)

		// Phase 2: Close listeners
		c.shutdownListeners(graceCtx, forceCtx, addError)

		// Phase 3: Close clients and connections
		c.shutdownClients(graceCtx, forceCtx, addError)

		// Phase 4: Run cleanup functions
		c.runCleanupFunctions(addError)

		// Collect all errors
		if len(errors) > 0 {
			for _, err := range errors {
				slog.Error("Shutdown error", "error", err)
				if finalErr == nil {
					finalErr = err
				}
			}
		} else {
			slog.Info("Graceful shutdown completed successfully")
		}

		if c.config.OnShutdownComplete != nil {
			c.config.OnShutdownComplete(finalErr)
		}
	})

	return finalErr
}

func (c *Coordinator) shutdownServers(graceCtx, forceCtx context.Context, addError func(error)) {
	slog.Info("Phase 1: Stopping servers gracefully")
	var serverWg sync.WaitGroup
	for _, server := range c.servers {
		serverWg.Add(1)
		go func(s Server) {
			defer serverWg.Done()
			if err := s.Close(); err != nil {
				addError(fmt.Errorf("server stop error: %w", err))
			}
		}(server)
	}

	success := c.waitForShutdown(forceCtx, &serverWg, "Phase 2: Closing listeners", "Grace timeout exceeded during server shutdown")
	if !success {
		addError(fmt.Errorf("server shutdown exceeded grace timeout of %v", c.config.GracePeriod))
	}
}

func (c *Coordinator) shutdownListeners(graceCtx, forceCtx context.Context, addError func(error)) {
	var listenerWg sync.WaitGroup
	for _, listener := range c.listeners {
		listenerWg.Add(1)
		go func(l Listener) {
			defer listenerWg.Done()
			if err := l.Close(); err != nil {
				addError(fmt.Errorf("listener close error: %w", err))
			}
		}(listener)
	}

	success := c.waitForShutdown(forceCtx, &listenerWg, "Servers and listeners stopped successfully", "Force timeout exceeded during listener shutdown")
	if !success {
		addError(fmt.Errorf("force timeout during listener cleanup"))
	}
}

func (c *Coordinator) shutdownClients(graceCtx, forceCtx context.Context, addError func(error)) {
	slog.Info("Phase 3: Closing clients and connections")
	var clientWg sync.WaitGroup
	for _, client := range c.clients {
		clientWg.Add(1)
		go func(cl Client) {
			defer clientWg.Done()
			if err := cl.Close(); err != nil {
				addError(fmt.Errorf("client close error: %w", err))
			}
		}(client)
	}
	clientWg.Wait()
}

func (c *Coordinator) runCleanupFunctions(addError func(error)) {
	slog.Info("Phase 4: Running cleanup functions")
	for _, fn := range c.cleanupFuncs {
		if err := fn(); err != nil {
			addError(fmt.Errorf("cleanup function error: %w", err))
		}
	}
}

func (c *Coordinator) waitForShutdown(ctx context.Context, wg *sync.WaitGroup, successMsg, timeoutMsg string) bool {
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		slog.Info(successMsg)
		return true
	case <-ctx.Done():
		slog.Warn(timeoutMsg)
		return false
	}
}
