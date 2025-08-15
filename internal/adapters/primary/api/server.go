package api

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"sync"

	"google.golang.org/grpc"

	"github.com/sufield/ephemos/internal/adapters/secondary/config"
	"github.com/sufield/ephemos/internal/adapters/secondary/spiffe"
	"github.com/sufield/ephemos/internal/adapters/secondary/transport"
	"github.com/sufield/ephemos/internal/core/errors"
	"github.com/sufield/ephemos/internal/core/ports"
	"github.com/sufield/ephemos/internal/core/services"
)

// Server provides a secure gRPC server with SPIFFE-based workload identity management.
type Server struct {
	identityService *services.IdentityService
	configProvider  ports.ConfigurationProvider
	serviceName     string
	domainServer    ports.ServerPort
	mu              sync.Mutex
}

// NewServerFromConfig creates a new workload server from configuration path.
// Handles all provider creation internally to hide implementation details from public API.
func NewServerFromConfig(ctx context.Context, configPath string) (*Server, error) {
	// Load configuration
	configProvider := config.NewFileProvider()
	cfg, err := configProvider.LoadConfiguration(ctx, configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Create identity provider
	identityProvider, err := spiffe.NewProvider(cfg.Agent)
	if err != nil {
		return nil, fmt.Errorf("failed to create identity provider: %w", err)
	}

	// Create default transport provider
	transportProvider := transport.NewGRPCProvider(cfg)
	
	return WorkloadServer(identityProvider, transportProvider, configProvider, cfg)
}

// NewServer creates a new workload server with minimal dependencies.
// Uses default transport provider internally to hide implementation details from public API.
func NewServer(
	identityProvider ports.IdentityProvider,
	configProvider ports.ConfigurationProvider,
	cfg *ports.Configuration,
) (*Server, error) {
	// Create default transport provider internally
	transportProvider := transport.NewGRPCProvider(cfg)
	
	return WorkloadServer(identityProvider, transportProvider, configProvider, cfg)
}

// WorkloadServer creates a new workload server with injected dependencies.
// This constructor follows proper dependency injection and hexagonal architecture principles.
func WorkloadServer(
	identityProvider ports.IdentityProvider,
	transportProvider ports.TransportProvider,
	configProvider ports.ConfigurationProvider,
	cfg *ports.Configuration,
) (*Server, error) {
	if cfg == nil {
		return nil, &errors.ValidationError{
			Field:   "configuration",
			Value:   nil,
			Message: "configuration cannot be nil",
		}
	}

	if identityProvider == nil {
		return nil, &errors.ValidationError{
			Field:   "identityProvider",
			Value:   nil,
			Message: "identity provider cannot be nil",
		}
	}

	if transportProvider == nil {
		return nil, &errors.ValidationError{
			Field:   "transportProvider",
			Value:   nil,
			Message: "transport provider cannot be nil",
		}
	}

	if configProvider == nil {
		return nil, &errors.ValidationError{
			Field:   "configProvider",
			Value:   nil,
			Message: "configuration provider cannot be nil",
		}
	}

	identityService, err := services.NewIdentityService(
		identityProvider,
		transportProvider,
		cfg,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create identity service: %w", err)
	}

	return &Server{
		identityService: identityService,
		configProvider:  configProvider,
		serviceName:     cfg.Service.Name,
	}, nil
}

// RegisterService registers a gRPC service with the identity server.
func (s *Server) RegisterService(ctx context.Context, serviceRegistrar ServiceRegistrar) error {
	// Input validation
	if serviceRegistrar == nil {
		return &errors.ValidationError{
			Field:   "serviceRegistrar",
			Value:   serviceRegistrar,
			Message: "service registrar cannot be nil",
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.domainServer == nil {
		if err := s.initializeServer(ctx); err != nil {
			return fmt.Errorf("failed to initialize server: %w", err)
		}
	}

	// Adapt our ServiceRegistrar to ports.ServiceRegistrarPort
	portServiceRegistrar := &serviceRegistrarAdapter{registrar: serviceRegistrar}
	if err := s.domainServer.RegisterService(portServiceRegistrar); err != nil {
		return fmt.Errorf("failed to register service: %w", err)
	}

	slog.Info("Service registered successfully", "service", s.serviceName)
	return nil
}

// Serve starts the identity server on the provided listener.
func (s *Server) Serve(ctx context.Context, listener net.Listener) error {
	// Input validation
	if listener == nil {
		return &errors.ValidationError{
			Field:   "listener",
			Value:   listener,
			Message: "listener cannot be nil",
		}
	}

	s.mu.Lock()
	if s.domainServer == nil {
		if err := s.initializeServer(ctx); err != nil {
			s.mu.Unlock()
			return fmt.Errorf("failed to initialize server: %w", err)
		}
	}
	s.mu.Unlock()

	slog.Info("Server ready", "service", s.serviceName, "address", listener.Addr().String())

	// Adapt net.Listener to ports.ListenerPort
	portListener := &netListenerAdapter{listener: listener}
	if err := s.domainServer.Start(portListener); err != nil {
		return fmt.Errorf("failed to serve domain server: %w", err)
	}
	return nil
}

// Close gracefully shuts down the identity server.
func (s *Server) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.domainServer != nil {
		if err := s.domainServer.Stop(); err != nil {
			return fmt.Errorf("failed to stop server: %w", err)
		}
		slog.Info("Server stopped gracefully", "service", s.serviceName)
	}

	return nil
}

func (s *Server) initializeServer(_ context.Context) error {
	server, err := s.identityService.CreateServerIdentity()
	if err != nil {
		return fmt.Errorf("failed to create server identity: %w", err)
	}
	s.domainServer = server
	slog.Info("Server identity created", "service", s.serviceName)
	return nil
}

// serviceRegistrarAdapter adapts our API ServiceRegistrar to ports.ServiceRegistrarPort.
type serviceRegistrarAdapter struct {
	registrar ServiceRegistrar
}

func (a *serviceRegistrarAdapter) Register(server interface{}) {
	if grpcServer, ok := server.(*grpc.Server); ok {
		a.registrar.Register(grpcServer)
	}
}

// netListenerAdapter adapts net.Listener to ports.ListenerPort.
type netListenerAdapter struct {
	listener net.Listener
}

// Accept waits for and returns the next connection.
func (l *netListenerAdapter) Accept() (interface{}, error) {
	return l.listener.Accept()
}

// Close closes the listener.
func (l *netListenerAdapter) Close() error {
	return l.listener.Close()
}

// Addr returns the listener's network address.
func (l *netListenerAdapter) Addr() string {
	return l.listener.Addr().String()
}
