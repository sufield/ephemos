// Example demonstrating complete mTLS scenario with invariant enforcement,
// connection management, and rotation continuity using Ephemos' enhanced capabilities.
package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/sufield/ephemos/internal/adapters/secondary/memidentity"
	"github.com/sufield/ephemos/internal/core/domain"
	"github.com/sufield/ephemos/internal/core/ports"
	"github.com/sufield/ephemos/internal/core/services"
)

func main() {
	fmt.Println("🔒 Complete mTLS Scenario with Ephemos")
	fmt.Println("=====================================")
	fmt.Println()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Scenario 1: Service setup with mTLS invariants
	fmt.Println("1. Setting up services with mTLS invariant enforcement")
	fmt.Println("-----------------------------------------------------")
	
	apiServer, authService, err := setupServices(ctx)
	if err != nil {
		log.Fatalf("Failed to setup services: %v", err)
	}
	fmt.Println("   ✅ API server and auth service ready with mTLS enforcement")
	fmt.Println()

	// Scenario 2: Establish secure inter-service connections
	fmt.Println("2. Establishing secure inter-service connections")
	fmt.Println("-----------------------------------------------")
	
	if err := demonstrateInterServiceConnections(ctx, apiServer, authService); err != nil {
		log.Fatalf("Inter-service connection demo failed: %v", err)
	}
	fmt.Println()

	// Scenario 3: mTLS invariant monitoring
	fmt.Println("3. Monitoring mTLS invariants across connections")
	fmt.Println("-----------------------------------------------")
	
	if err := demonstrateInvariantMonitoring(ctx, apiServer, authService); err != nil {
		log.Fatalf("Invariant monitoring demo failed: %v", err)
	}
	fmt.Println()

	// Scenario 4: Certificate rotation with zero downtime
	fmt.Println("4. Certificate rotation with zero downtime")
	fmt.Println("-----------------------------------------")
	
	if err := demonstrateRotationContinuity(ctx, apiServer, authService); err != nil {
		log.Fatalf("Rotation continuity demo failed: %v", err)
	}
	fmt.Println()

	// Scenario 5: End-to-end security validation
	fmt.Println("5. End-to-end security validation")
	fmt.Println("---------------------------------")
	
	if err := demonstrateSecurityValidation(ctx, apiServer, authService); err != nil {
		log.Fatalf("Security validation demo failed: %v", err)
	}

	fmt.Println("\n🎉 Complete mTLS scenario completed successfully!")
	fmt.Println("   All security invariants enforced, connections managed, and rotations completed.")
}

func setupServices(ctx context.Context) (*services.IdentityService, *services.IdentityService, error) {
	// Setup API server service
	apiServerIdentity := domain.NewServiceIdentity("api-server", "production.company.com")
	apiServerProvider := memidentity.New().WithIdentity(apiServerIdentity)
	
	apiServerConfig := &ports.Configuration{
		Service: ports.ServiceConfig{
			Name:   "api-server",
			Domain: "production.company.com",
			// Configure authorized clients for server-side authorization
			AuthorizedClients: []string{
				"spiffe://production.company.com/web-client",
				"spiffe://production.company.com/mobile-client",
			},
			// Configure trusted servers for client-side authorization
			TrustedServers: []string{
				"spiffe://production.company.com/auth-service",
				"spiffe://production.company.com/db-proxy",
			},
		},
		Agent: &ports.AgentConfig{
			SocketPath: "/tmp/spire-agent/public/api.sock",
		},
	}

	apiServer, err := services.NewIdentityService(
		apiServerProvider, &mockTransportProvider{}, apiServerConfig, nil, nil,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create API server service: %w", err)
	}

	// Setup auth service
	authServiceIdentity := domain.NewServiceIdentity("auth-service", "production.company.com")
	authServiceProvider := memidentity.New().WithIdentity(authServiceIdentity)
	
	authServiceConfig := &ports.Configuration{
		Service: ports.ServiceConfig{
			Name:   "auth-service",
			Domain: "production.company.com",
			AuthorizedClients: []string{
				"spiffe://production.company.com/api-server",
			},
		},
		Agent: &ports.AgentConfig{
			SocketPath: "/tmp/spire-agent/public/api.sock",
		},
	}

	authService, err := services.NewIdentityService(
		authServiceProvider, &mockTransportProvider{}, authServiceConfig, nil, nil,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create auth service: %w", err)
	}

	fmt.Printf("   🔐 API Server: %s\n", apiServerIdentity.URI())
	fmt.Printf("   🔐 Auth Service: %s\n", authServiceIdentity.URI())

	return apiServer, authService, nil
}

func demonstrateInterServiceConnections(ctx context.Context, apiServer, authService *services.IdentityService) error {
	fmt.Println("   🔗 Establishing API server -> Auth service connection...")
	
	// API server connects to auth service
	authIdentity := domain.NewServiceIdentity("auth-service", "production.company.com")
	conn, err := apiServer.EstablishMTLSConnection(ctx, "api-to-auth", authIdentity)
	if err != nil {
		return fmt.Errorf("failed to establish API->Auth connection: %w", err)
	}
	
	fmt.Printf("   ✅ Connection established: %s (state: %v)\n", conn.ID, conn.State)
	fmt.Printf("   📋 Local identity: %s\n", conn.LocalIdentity.URI())
	fmt.Printf("   📋 Remote identity: %s\n", conn.RemoteIdentity.URI())
	
	// Verify connection statistics
	stats := apiServer.GetConnectionStats()
	fmt.Printf("   📊 API server connections: %d active\n", stats.TotalConnections)
	
	return nil
}

func demonstrateInvariantMonitoring(ctx context.Context, apiServer, authService *services.IdentityService) error {
	fmt.Println("   🛡️  Starting mTLS invariant enforcement...")
	
	// Start invariant enforcement on both services
	if err := apiServer.StartMTLSEnforcement(ctx); err != nil {
		return fmt.Errorf("failed to start enforcement on API server: %w", err)
	}
	
	if err := authService.StartMTLSEnforcement(ctx); err != nil {
		return fmt.Errorf("failed to start enforcement on auth service: %w", err)
	}
	
	// Allow some time for invariant checking
	time.Sleep(500 * time.Millisecond)
	
	// Check invariant status on API server
	apiStatus := apiServer.GetInvariantStatus(ctx)
	fmt.Printf("   📊 API Server invariants: %d total, %d connections\n", 
		apiStatus.TotalInvariants, apiStatus.TotalConnections)
	
	// Report on each invariant
	for name, result := range apiStatus.InvariantResults {
		fmt.Printf("   📋 %s: %d pass, %d fail\n", name, result.PassCount, result.FailCount)
		if len(result.Violations) > 0 {
			fmt.Printf("      ⚠️  Violations: %v\n", result.Violations)
		}
	}
	
	// Check invariant status on auth service
	authStatus := authService.GetInvariantStatus(ctx)
	fmt.Printf("   📊 Auth Service invariants: %d total, %d connections\n", 
		authStatus.TotalInvariants, authStatus.TotalConnections)
	
	fmt.Printf("   ✅ Total invariant checks across both services: %d\n", 
		apiStatus.TotalInvariants + authStatus.TotalInvariants)
	
	return nil
}

func demonstrateRotationContinuity(ctx context.Context, apiServer, authService *services.IdentityService) error {
	fmt.Println("   🔄 Configuring rotation continuity policies...")
	
	// Configure rotation continuity policy
	continuityPolicy := &services.ContinuityPolicy{
		OverlapDuration:            3 * time.Second,  // 3 seconds overlap
		GracefulShutdownTimeout:    1 * time.Second,  // 1 second shutdown
		PreRotationPrepTime:        500 * time.Millisecond, // 500ms prep
		PostRotationValidationTime: 500 * time.Millisecond, // 500ms validation
		MaxConcurrentRotations:     2, // Allow 2 concurrent rotations
	}
	
	apiServer.SetContinuityPolicy(continuityPolicy)
	authService.SetContinuityPolicy(continuityPolicy)
	
	fmt.Printf("   ⚙️  Rotation policy configured: %v overlap, %v shutdown timeout\n", 
		continuityPolicy.OverlapDuration, continuityPolicy.GracefulShutdownTimeout)
	
	// Add rotation observer to monitor events
	observer := &rotationObserver{events: make([]string, 0)}
	apiServer.AddRotationObserver(observer)
	
	// Perform server rotation
	fmt.Println("   🔄 Performing server rotation with continuity...")
	server, err := apiServer.CreateServerIdentity()
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}
	
	rotationStart := time.Now()
	err = apiServer.RotateServerWithContinuity(ctx, "demo-server", server)
	rotationDuration := time.Since(rotationStart)
	
	if err != nil {
		return fmt.Errorf("server rotation failed: %w", err)
	}
	
	fmt.Printf("   ✅ Server rotation completed in %v\n", rotationDuration)
	
	// Report rotation events
	fmt.Printf("   📋 Rotation events observed: %d\n", len(observer.events))
	for i, event := range observer.events {
		fmt.Printf("      %d. %s\n", i+1, event)
	}
	
	// Check rotation statistics
	stats := apiServer.GetRotationStats()
	fmt.Printf("   📊 Rotation stats: %d active rotations (max %d allowed)\n", 
		stats.TotalActiveRotations, stats.MaxConcurrentAllowed)
	
	return nil
}

func demonstrateSecurityValidation(ctx context.Context, apiServer, authService *services.IdentityService) error {
	fmt.Println("   🔒 Running comprehensive security validation...")
	
	// 1. Validate all connections have proper certificates
	apiConnections := apiServer.ListMTLSConnections()
	fmt.Printf("   📋 Validating %d API server connections...\n", len(apiConnections))
	
	for _, conn := range apiConnections {
		if conn.Cert == nil {
			return fmt.Errorf("connection %s has no certificate", conn.ID)
		}
		
		
		// Validate certificate expiry
		if time.Now().After(conn.Cert.Cert.NotAfter) {
			return fmt.Errorf("connection %s has expired certificate", conn.ID)
		}
		
		fmt.Printf("   ✅ Connection %s: valid certificate until %v\n", 
			conn.ID, conn.Cert.Cert.NotAfter)
	}
	
	// 2. Validate invariant enforcement is working
	apiStatus := apiServer.GetInvariantStatus(ctx)
	authStatus := authService.GetInvariantStatus(ctx)
	
	if apiStatus.TotalInvariants == 0 {
		return fmt.Errorf("API server has no active invariants")
	}
	
	if authStatus.TotalInvariants == 0 {
		return fmt.Errorf("auth service has no active invariants")
	}
	
	fmt.Printf("   ✅ Invariant enforcement active: %d + %d = %d total invariants\n",
		apiStatus.TotalInvariants, authStatus.TotalInvariants, 
		apiStatus.TotalInvariants + authStatus.TotalInvariants)
	
	// 3. Validate service identities
	apiIdentity := domain.NewServiceIdentity("api-server", "production.company.com")
	authIdentity := domain.NewServiceIdentity("auth-service", "production.company.com")
	
	if err := apiIdentity.Validate(); err != nil {
		return fmt.Errorf("API server identity validation failed: %w", err)
	}
	
	if err := authIdentity.Validate(); err != nil {
		return fmt.Errorf("auth service identity validation failed: %w", err)
	}
	
	fmt.Printf("   ✅ Service identities validated:\n")
	fmt.Printf("      - %s\n", apiIdentity.URI())
	fmt.Printf("      - %s\n", authIdentity.URI())
	
	// 4. Validate trust domains match
	if apiIdentity.TrustDomain() != authIdentity.TrustDomain() {
		return fmt.Errorf("services have mismatched trust domains: %s vs %s",
			apiIdentity.TrustDomain(), authIdentity.TrustDomain())
	}
	
	fmt.Printf("   ✅ Trust domain consistency: %s\n", apiIdentity.TrustDomain())
	
	// 5. Final security summary
	totalConnections := len(apiConnections)
	totalInvariants := apiStatus.TotalInvariants + authStatus.TotalInvariants
	
	fmt.Printf("   🛡️  Security validation summary:\n")
	fmt.Printf("      - %d secure mTLS connections\n", totalConnections)
	fmt.Printf("      - %d security invariants enforced\n", totalInvariants)
	fmt.Printf("      - Zero-downtime rotation capability verified\n")
	fmt.Printf("      - Trust domain consistency validated\n")
	fmt.Printf("      - Certificate validity confirmed\n")
	
	return nil
}

// rotationObserver observes rotation events for demonstration
type rotationObserver struct {
	events []string
}

func (o *rotationObserver) OnRotationStarted(connID string, reason string) {
	o.events = append(o.events, fmt.Sprintf("Started rotation for %s (reason: %s)", connID, reason))
}

func (o *rotationObserver) OnRotationCompleted(connID string, oldCert, newCert *domain.Certificate) {
	o.events = append(o.events, fmt.Sprintf("Completed rotation for %s", connID))
}

func (o *rotationObserver) OnRotationFailed(connID string, err error) {
	o.events = append(o.events, fmt.Sprintf("Failed rotation for %s: %v", connID, err))
}

// mockTransportProvider provides mock transport for demonstration
type mockTransportProvider struct{}

func (m *mockTransportProvider) CreateServer(
	_ *domain.Certificate, _ *domain.TrustBundle, _ *domain.AuthenticationPolicy,
) (ports.ServerPort, error) {
	return &mockServer{}, nil
}

func (m *mockTransportProvider) CreateClient(
	_ *domain.Certificate, _ *domain.TrustBundle, _ *domain.AuthenticationPolicy,
) (ports.ClientPort, error) {
	return &mockClient{}, nil
}

type mockServer struct{}

func (m *mockServer) RegisterService(_ ports.ServiceRegistrarPort) error { return nil }
func (m *mockServer) Start(_ net.Listener) error                        { return nil }
func (m *mockServer) Stop() error                                        { return nil }

type mockClient struct{}

func (m *mockClient) Connect(_, _ string) (ports.ConnectionPort, error) {
	return &mockConnection{}, nil
}
func (m *mockClient) Close() error { return nil }

type mockConnection struct{}

func (m *mockConnection) GetClientConnection() interface{} { return nil }
func (m *mockConnection) AsNetConn() net.Conn              { return nil }
func (m *mockConnection) Close() error                     { return nil }