package services

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/sufield/ephemos/internal/core/domain"
	"github.com/sufield/ephemos/internal/core/errors"
	"github.com/sufield/ephemos/internal/core/ports"
	"google.golang.org/grpc"
)

// IdentityService manages service identities and provides authenticated transport.
// It handles certificate management, identity validation, and secure connection establishment.
// The service caches validated identities for performance and thread-safety.
type IdentityService struct {
	identityProvider  ports.IdentityProvider
	transportProvider ports.TransportProvider
	config            *ports.Configuration
	cachedIdentity    *domain.ServiceIdentity
	mu                sync.RWMutex
}

// NewIdentityService creates a new IdentityService with the provided configuration.
// The configuration is validated and cached during initialization for better performance.
// Returns an error if the configuration is invalid.
func NewIdentityService(
	identityProvider ports.IdentityProvider,
	transportProvider ports.TransportProvider,
	config *ports.Configuration,
) (*IdentityService, error) {
	if config == nil {
		return nil, &errors.ValidationError{
			Field:   "config",
			Value:   nil,
			Message: "configuration cannot be nil",
		}
	}
	
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}
	
	// Create and validate identity during initialization
	identity := domain.NewServiceIdentity(config.Service.Name, config.Service.Domain)
	if err := identity.Validate(); err != nil {
		return nil, fmt.Errorf("invalid service identity: %w", err)
	}
	
	return &IdentityService{
		identityProvider:  identityProvider,
		transportProvider: transportProvider,
		config:            config,
		cachedIdentity:    identity,
	}, nil
}

// CreateServerIdentity creates a gRPC server with identity-based authentication.
// Uses the cached identity and configuration to avoid redundant validation.
// Returns a configured gRPC server ready for service registration.
func (s *IdentityService) CreateServerIdentity(ctx context.Context) (*grpc.Server, error) {
	if ctx == nil {
		return nil, &errors.ValidationError{
			Field:   "context",
			Value:   nil,
			Message: "context cannot be nil",
		}
	}
	
	s.mu.RLock()
	identity := s.cachedIdentity
	config := s.config
	s.mu.RUnlock()

	cert, err := s.identityProvider.GetCertificate(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get certificate for service %s: %w", identity.Name, err)
	}

	trustBundle, err := s.identityProvider.GetTrustBundle(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get trust bundle for service %s: %w", identity.Name, err)
	}

	policy := domain.NewAuthenticationPolicy(identity)
	for _, client := range config.AuthorizedClients {
		policy.AddAuthorizedClient(client)
	}

	server, err := s.transportProvider.CreateServerTransport(ctx, cert, trustBundle, policy)
	if err != nil {
		return nil, fmt.Errorf("failed to create server transport for service %s: %w", identity.Name, err)
	}

	return server, nil
}

// CreateClientIdentity creates a client connection with identity-based authentication.
// Uses the cached identity and configuration to avoid redundant validation.
// Returns a ClientConnection ready for establishing secure connections to servers.
func (s *IdentityService) CreateClientIdentity(ctx context.Context) (ClientConnection, error) {
	if ctx == nil {
		return nil, &errors.ValidationError{
			Field:   "context",
			Value:   nil,
			Message: "context cannot be nil",
		}
	}
	
	s.mu.RLock()
	identity := s.cachedIdentity
	config := s.config
	s.mu.RUnlock()

	cert, err := s.identityProvider.GetCertificate(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get certificate for service %s: %w", identity.Name, err)
	}

	trustBundle, err := s.identityProvider.GetTrustBundle(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get trust bundle for service %s: %w", identity.Name, err)
	}

	policy := domain.NewAuthenticationPolicy(identity)
	for _, server := range config.TrustedServers {
		policy.AddTrustedServer(server)
	}

	dialOption, err := s.transportProvider.CreateClientTransport(ctx, cert, trustBundle, policy)
	if err != nil {
		return nil, fmt.Errorf("failed to create client transport for service %s: %w", identity.Name, err)
	}

	return &clientConnection{
		identity:   identity,
		dialOption: dialOption,
	}, nil
}


// ClientConnection represents a client connection with identity-based authentication.
// It provides secure connection establishment to servers with automatic certificate validation.
type ClientConnection interface {
	// Connect establishes a secure connection to the specified service.
	// The serviceName is used for identity verification and must be non-empty.
	// The address should be in "host:port" format and must be non-empty.
	// The context is used for cancellation and timeouts during connection establishment.
	Connect(ctx context.Context, serviceName, address string) (*grpc.ClientConn, error)
}

type clientConnection struct {
	identity   *domain.ServiceIdentity
	dialOption grpc.DialOption
	mu         sync.Mutex
}

func (c *clientConnection) Connect(ctx context.Context, serviceName, address string) (*grpc.ClientConn, error) {
	// Input validation
	if ctx == nil {
		return nil, &errors.ValidationError{
			Field:   "context",
			Value:   nil,
			Message: "context cannot be nil",
		}
	}
	
	if strings.TrimSpace(serviceName) == "" {
		return nil, &errors.ValidationError{
			Field:   "serviceName",
			Value:   serviceName,
			Message: "service name cannot be empty or whitespace",
		}
	}
	
	if strings.TrimSpace(address) == "" {
		return nil, &errors.ValidationError{
			Field:   "address",
			Value:   address,
			Message: "address cannot be empty or whitespace",
		}
	}
	
	// Validate address format
	if _, _, err := net.SplitHostPort(address); err != nil {
		return nil, &errors.ValidationError{
			Field:   "address",
			Value:   address,
			Message: "address must be in format 'host:port'",
		}
	}
	
	serviceName = strings.TrimSpace(serviceName)
	address = strings.TrimSpace(address)
	
	// Thread-safe connection establishment
	c.mu.Lock()
	defer c.mu.Unlock()
	
	conn, err := grpc.NewClient(address, c.dialOption)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to service %s at %s: %w", serviceName, address, err)
	}
	
	return conn, nil
}
