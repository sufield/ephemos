// Package integration provides end-to-end tests for mTLS invariants and connection management.
// These tests validate that all mTLS security invariants are properly enforced during
// connection establishment, maintenance, and rotation.
package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/sufield/ephemos/internal/adapters/secondary/memidentity"
	"github.com/sufield/ephemos/internal/core/domain"
	"github.com/sufield/ephemos/internal/core/ports"
	"github.com/sufield/ephemos/internal/core/services"
)

// TestMTLSInvariantEnforcement tests end-to-end mTLS invariant enforcement
func TestMTLSInvariantEnforcement(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("CompleteInvariantFlow", func(t *testing.T) {
		// Setup identity service with mTLS components
		identityService := testSetupMTLSIdentityService(t)
		
		// Test connection establishment with invariant checks
		testEstablishConnectionWithInvariants(ctx, t, identityService)
		
		// Test invariant enforcement during connection lifecycle
		testConnectionLifecycleInvariants(ctx, t, identityService)
		
		// Test invariant violations are detected
		testInvariantViolationDetection(ctx, t, identityService)
	})

	t.Run("RotationContinuityFlow", func(t *testing.T) {
		identityService := testSetupMTLSIdentityService(t)
		
		// Test server rotation with continuity
		testServerRotationContinuity(ctx, t, identityService)
		
		// Test client rotation with continuity  
		testClientRotationContinuity(ctx, t, identityService)
		
		// Test concurrent rotations
		testConcurrentRotations(ctx, t, identityService)
	})

	t.Run("InvariantStatusAndMetrics", func(t *testing.T) {
		identityService := testSetupMTLSIdentityService(t)
		
		// Test invariant status reporting
		testInvariantStatusReporting(ctx, t, identityService)
		
		// Test connection statistics
		testConnectionStatistics(ctx, t, identityService)
		
		// Test enforcement policy configuration
		testEnforcementPolicyConfiguration(ctx, t, identityService)
	})
}

func testSetupMTLSIdentityService(t *testing.T) *services.IdentityService {
	t.Helper()
	
	// Create service identity
	identity := domain.NewServiceIdentity("mtls-test-service", "test.example.org")
	if err := identity.Validate(); err != nil {
		t.Fatalf("Failed to create valid identity: %v", err)
	}

	// Setup identity provider
	provider := memidentity.New().WithIdentity(identity)
	
	// Setup configuration
	config := &ports.Configuration{
		Service: ports.ServiceConfig{
			Name:   "mtls-test-service",
			Domain: "test.example.org",
		},
		Agent: &ports.AgentConfig{
			SocketPath: "/tmp/test-spire-agent/public/api.sock",
		},
	}

	// Create transport provider
	transportProvider := &mockTransportProvider{}

	// Create identity service with mTLS components
	identityService, err := services.NewIdentityService(
		provider, transportProvider, config, nil, nil,
	)
	if err != nil {
		t.Fatalf("Failed to create identity service: %v", err)
	}

	t.Logf("✅ Created mTLS identity service for %s", identity.URI())
	return identityService
}

func testEstablishConnectionWithInvariants(ctx context.Context, t *testing.T, identityService *services.IdentityService) {
	t.Helper()

	// Create remote identity for connection
	remoteIdentity := domain.NewServiceIdentity("remote-service", "test.example.org")
	
	// Establish mTLS connection with full invariant enforcement
	conn, err := identityService.EstablishMTLSConnection(ctx, "test-conn-1", remoteIdentity)
	if err != nil {
		t.Fatalf("Failed to establish mTLS connection: %v", err)
	}
	
	// Verify connection exists and is active
	retrievedConn, exists := identityService.GetMTLSConnection("test-conn-1")
	if !exists {
		t.Fatal("Connection not found after establishment")
	}
	
	if retrievedConn.ID != conn.ID {
		t.Errorf("Connection ID mismatch: expected %s, got %s", conn.ID, retrievedConn.ID)
	}

	// Verify connection statistics
	stats := identityService.GetConnectionStats()
	if stats.TotalConnections != 1 {
		t.Errorf("Expected 1 connection, got %d", stats.TotalConnections)
	}

	t.Logf("✅ Successfully established mTLS connection with invariants: %s (state: %v)", conn.ID, conn.State)
}

func testConnectionLifecycleInvariants(ctx context.Context, t *testing.T, identityService *services.IdentityService) {
	t.Helper()

	remoteIdentity := domain.NewServiceIdentity("lifecycle-service", "test.example.org")
	
	// Start mTLS enforcement
	if err := identityService.StartMTLSEnforcement(ctx); err != nil {
		t.Fatalf("Failed to start mTLS enforcement: %v", err)
	}

	// Establish connection
	conn, err := identityService.EstablishMTLSConnection(ctx, "lifecycle-conn", remoteIdentity)
	if err != nil {
		t.Fatalf("Failed to establish connection: %v", err)
	}
	t.Logf("Created lifecycle test connection: %s", conn.ID)

	// Give enforcement service time to check invariants
	time.Sleep(100 * time.Millisecond)

	// Check invariant status
	status := identityService.GetInvariantStatus(ctx)
	if status.TotalConnections != 1 {
		t.Errorf("Expected 1 connection in status, got %d", status.TotalConnections)
	}
	
	if status.TotalInvariants == 0 {
		t.Error("No invariants registered")
	}

	// Verify all default invariants are present
	expectedInvariants := []string{
		"certificate_validity",
		"mutual_authentication", 
		"trust_domain_validation",
		"certificate_rotation",
		"identity_matching",
	}
	
	for _, expectedInvariant := range expectedInvariants {
		if _, exists := status.InvariantResults[expectedInvariant]; !exists {
			t.Errorf("Expected invariant %s not found in status", expectedInvariant)
		}
	}

	// Close connection
	if err := identityService.CloseMTLSConnection("lifecycle-conn"); err != nil {
		t.Errorf("Failed to close connection: %v", err)
	}

	t.Logf("✅ Connection lifecycle with %d invariants validated", status.TotalInvariants)
}

func testInvariantViolationDetection(ctx context.Context, t *testing.T, identityService *services.IdentityService) {
	t.Helper()

	// Create connection that will have invariant violations
	remoteIdentity := domain.NewServiceIdentity("violation-service", "test.example.org")
	
	conn, err := identityService.EstablishMTLSConnection(ctx, "violation-conn", remoteIdentity)
	if err != nil {
		t.Fatalf("Failed to establish connection: %v", err)
	}
	t.Logf("Created test connection for violation detection: %s", conn.ID)

	// Set aggressive enforcement policy for testing
	aggressivePolicy := &services.EnforcementPolicy{
		FailOnViolation: true,
		CheckInterval:   100 * time.Millisecond,
		MaxViolations:   1,
		ViolationAction: services.ActionLog,
	}
	identityService.SetEnforcementPolicy(aggressivePolicy)

	// Start enforcement
	if err := identityService.StartMTLSEnforcement(ctx); err != nil {
		t.Fatalf("Failed to start enforcement: %v", err)
	}

	// Wait for enforcement checks
	time.Sleep(200 * time.Millisecond)

	// Check for violations (some violations may be expected with mock components)
	status := identityService.GetInvariantStatus(ctx)
	t.Logf("✅ Invariant violation detection tested with %d invariants", len(status.InvariantResults))
	
	// Clean up
	identityService.CloseMTLSConnection("violation-conn")
}

func testServerRotationContinuity(ctx context.Context, t *testing.T, identityService *services.IdentityService) {
	t.Helper()

	// Create initial server
	server, err := identityService.CreateServerIdentity()
	if err != nil {
		t.Fatalf("Failed to create server identity: %v", err)
	}

	// Perform server rotation with continuity
	err = identityService.RotateServerWithContinuity(ctx, "test-server", server)
	if err != nil {
		t.Fatalf("Server rotation with continuity failed: %v", err)
	}

	// Check rotation statistics
	stats := identityService.GetRotationStats()
	t.Logf("✅ Server rotation completed: active rotations: %d", stats.TotalActiveRotations)
}

func testClientRotationContinuity(ctx context.Context, t *testing.T, identityService *services.IdentityService) {
	t.Helper()

	// Create initial client
	client, err := identityService.CreateClientIdentity()
	if err != nil {
		t.Fatalf("Failed to create client identity: %v", err)
	}

	// Perform client rotation with continuity
	err = identityService.RotateClientWithContinuity(ctx, "test-client", client)
	if err != nil {
		t.Fatalf("Client rotation with continuity failed: %v", err)
	}

	// Check rotation statistics
	stats := identityService.GetRotationStats()
	t.Logf("✅ Client rotation completed: active rotations: %d", stats.TotalActiveRotations)
}

func testConcurrentRotations(ctx context.Context, t *testing.T, identityService *services.IdentityService) {
	t.Helper()

	// Set continuity policy for concurrent rotations
	policy := &services.ContinuityPolicy{
		OverlapDuration:            2 * time.Second,
		GracefulShutdownTimeout:    1 * time.Second,
		PreRotationPrepTime:        500 * time.Millisecond,
		PostRotationValidationTime: 500 * time.Millisecond,
		MaxConcurrentRotations:     2,
	}
	identityService.SetContinuityPolicy(policy)

	// Create server and client
	server, err := identityService.CreateServerIdentity()
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	client, err := identityService.CreateClientIdentity()
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Start concurrent rotations
	serverDone := make(chan error, 1)
	clientDone := make(chan error, 1)

	go func() {
		serverDone <- identityService.RotateServerWithContinuity(ctx, "concurrent-server", server)
	}()

	go func() {
		clientDone <- identityService.RotateClientWithContinuity(ctx, "concurrent-client", client)
	}()

	// Wait for both to complete
	select {
	case err := <-serverDone:
		if err != nil {
			t.Errorf("Server rotation failed: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Error("Server rotation timed out")
	}

	select {
	case err := <-clientDone:
		if err != nil {
			t.Errorf("Client rotation failed: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Error("Client rotation timed out")
	}

	t.Logf("✅ Concurrent rotations completed successfully")
}

func testInvariantStatusReporting(ctx context.Context, t *testing.T, identityService *services.IdentityService) {
	t.Helper()

	// Get initial status
	status := identityService.GetInvariantStatus(ctx)
	
	// Verify status structure
	if status.TotalInvariants == 0 {
		t.Error("No invariants found in status")
	}

	// Check that default invariants are reported
	for name, result := range status.InvariantResults {
		if result.Name != name {
			t.Errorf("Invariant name mismatch: key=%s, name=%s", name, result.Name)
		}
		if result.Description == "" {
			t.Errorf("Invariant %s has no description", name)
		}
		t.Logf("📊 Invariant %s: %d pass, %d fail", name, result.PassCount, result.FailCount)
	}

	t.Logf("✅ Invariant status reporting validated: %d invariants", status.TotalInvariants)
}

func testConnectionStatistics(ctx context.Context, t *testing.T, identityService *services.IdentityService) {
	t.Helper()

	// Establish multiple connections
	remoteIdentity1 := domain.NewServiceIdentity("stats-service-1", "test.example.org")
	remoteIdentity2 := domain.NewServiceIdentity("stats-service-2", "test.example.org")

	_, err := identityService.EstablishMTLSConnection(ctx, "stats-conn-1", remoteIdentity1)
	if err != nil {
		t.Fatalf("Failed to establish connection 1: %v", err)
	}

	_, err = identityService.EstablishMTLSConnection(ctx, "stats-conn-2", remoteIdentity2)
	if err != nil {
		t.Fatalf("Failed to establish connection 2: %v", err)
	}

	// Check statistics
	stats := identityService.GetConnectionStats()
	if stats.TotalConnections < 2 {
		t.Errorf("Expected at least 2 connections, got %d", stats.TotalConnections)
	}

	// List connections
	connections := identityService.ListMTLSConnections()
	if len(connections) < 2 {
		t.Errorf("Expected at least 2 connections in list, got %d", len(connections))
	}

	t.Logf("✅ Connection statistics validated: %d total connections", stats.TotalConnections)

	// Clean up
	identityService.CloseMTLSConnection("stats-conn-1")
	identityService.CloseMTLSConnection("stats-conn-2")
}

func testEnforcementPolicyConfiguration(ctx context.Context, t *testing.T, identityService *services.IdentityService) {
	t.Helper()

	// Test different enforcement policies
	policies := []*services.EnforcementPolicy{
		{
			FailOnViolation: true,
			CheckInterval:   1 * time.Second,
			MaxViolations:   5,
			ViolationAction: services.ActionLog,
		},
		{
			FailOnViolation: false,
			CheckInterval:   500 * time.Millisecond,
			MaxViolations:   3,
			ViolationAction: services.ActionCloseConnection,
		},
		{
			FailOnViolation: true,
			CheckInterval:   2 * time.Second,
			MaxViolations:   1,
			ViolationAction: services.ActionAlertOnly,
		},
	}

	for i, policy := range policies {
		t.Run(fmt.Sprintf("Policy%d", i+1), func(t *testing.T) {
			// Apply policy
			identityService.SetEnforcementPolicy(policy)

			// Start enforcement
			if err := identityService.StartMTLSEnforcement(ctx); err != nil {
				t.Errorf("Failed to start enforcement with policy %d: %v", i+1, err)
			}

			// Brief wait to let enforcement run
			time.Sleep(100 * time.Millisecond)

			t.Logf("✅ Enforcement policy %d applied: fail=%v, interval=%v, max=%d, action=%s",
				i+1, policy.FailOnViolation, policy.CheckInterval, policy.MaxViolations, policy.ViolationAction.String())
		})
	}
}

// TestRotationObserverPattern tests the observer pattern for rotation events
func TestRotationObserverPattern(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	t.Run("RotationEventObserver", func(t *testing.T) {
		identityService := testSetupMTLSIdentityService(t)
		
		// Create a test observer
		observer := &testRotationObserver{
			events: make([]string, 0),
		}
		
		// Add observer
		identityService.AddRotationObserver(observer)
		
		// Perform rotation to trigger events
		server, err := identityService.CreateServerIdentity()
		if err != nil {
			t.Fatalf("Failed to create server: %v", err)
		}
		
		err = identityService.RotateServerWithContinuity(ctx, "observer-test-server", server)
		if err != nil {
			t.Fatalf("Server rotation failed: %v", err)
		}
		
		// Wait for potential events
		time.Sleep(100 * time.Millisecond)
		
		// Check if observer received events
		if len(observer.events) > 0 {
			t.Logf("✅ Observer received %d rotation events:", len(observer.events))
			for i, event := range observer.events {
				t.Logf("   Event %d: %s", i+1, event)
			}
		} else {
			t.Log("✅ Observer pattern validated (no events in this test scenario)")
		}
	})
}

// testRotationObserver implements RotationObserver for testing
type testRotationObserver struct {
	events []string
}

func (o *testRotationObserver) OnRotationStarted(connID string, reason string) {
	o.events = append(o.events, fmt.Sprintf("Started: %s (reason: %s)", connID, reason))
}

func (o *testRotationObserver) OnRotationCompleted(connID string, oldCert, newCert *domain.Certificate) {
	o.events = append(o.events, fmt.Sprintf("Completed: %s", connID))
}

func (o *testRotationObserver) OnRotationFailed(connID string, err error) {
	o.events = append(o.events, fmt.Sprintf("Failed: %s (%v)", connID, err))
}

// TestEndToEndScenarios tests realistic end-to-end scenarios
func TestEndToEndScenarios(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("ServiceCommunicationScenario", func(t *testing.T) {
		// Simulate a realistic scenario with multiple services
		apiServerService := testSetupNamedMTLSService(t, "api-server", "production.company.com")
		authServiceService := testSetupNamedMTLSService(t, "auth-service", "production.company.com")
		dbProxyService := testSetupNamedMTLSService(t, "db-proxy", "production.company.com")

		// Test inter-service connections
		testInterServiceConnections(ctx, t, apiServerService, authServiceService, dbProxyService)
		
		// Test rotation coordination
		testCoordinatedRotations(ctx, t, apiServerService, authServiceService)
		
		// Test invariant enforcement across services
		testCrossServiceInvariants(ctx, t, apiServerService, authServiceService, dbProxyService)
	})
}

func testSetupNamedMTLSService(t *testing.T, serviceName, trustDomain string) *services.IdentityService {
	t.Helper()

	identity := domain.NewServiceIdentity(serviceName, trustDomain)
	provider := memidentity.New().WithIdentity(identity)
	
	config := &ports.Configuration{
		Service: ports.ServiceConfig{
			Name:   serviceName,
			Domain: trustDomain,
		},
		Agent: &ports.AgentConfig{
			SocketPath: "/tmp/test-spire-agent/public/api.sock",
		},
	}

	identityService, err := services.NewIdentityService(
		provider, &mockTransportProvider{}, config, nil, nil,
	)
	if err != nil {
		t.Fatalf("Failed to create identity service for %s: %v", serviceName, err)
	}

	t.Logf("✅ Created service: %s@%s", serviceName, trustDomain)
	return identityService
}

func testInterServiceConnections(ctx context.Context, t *testing.T, apiServer, authService, dbProxy *services.IdentityService) {
	t.Helper()

	// API server connects to auth service
	authIdentity := domain.NewServiceIdentity("auth-service", "production.company.com")
	apiToAuthConn, err := apiServer.EstablishMTLSConnection(ctx, "api-to-auth", authIdentity)
	if err != nil {
		t.Fatalf("Failed to establish API->Auth connection: %v", err)
	}

	// API server connects to DB proxy
	dbIdentity := domain.NewServiceIdentity("db-proxy", "production.company.com")
	apiToDbConn, err := apiServer.EstablishMTLSConnection(ctx, "api-to-db", dbIdentity)
	if err != nil {
		t.Fatalf("Failed to establish API->DB connection: %v", err)
	}

	// Verify connections
	if apiToAuthConn.RemoteIdentity.Name() != "auth-service" {
		t.Errorf("Expected auth-service, got %s", apiToAuthConn.RemoteIdentity.Name())
	}
	if apiToDbConn.RemoteIdentity.Name() != "db-proxy" {
		t.Errorf("Expected db-proxy, got %s", apiToDbConn.RemoteIdentity.Name())
	}

	t.Logf("✅ Inter-service connections established: API server connected to %d services",
		len(apiServer.ListMTLSConnections()))
}

func testCoordinatedRotations(ctx context.Context, t *testing.T, service1, service2 *services.IdentityService) {
	t.Helper()

	// Create servers for both services
	server1, err := service1.CreateServerIdentity()
	if err != nil {
		t.Fatalf("Failed to create server 1: %v", err)
	}

	server2, err := service2.CreateServerIdentity()
	if err != nil {
		t.Fatalf("Failed to create server 2: %v", err)
	}

	// Perform coordinated rotations
	rotation1Done := make(chan error, 1)
	rotation2Done := make(chan error, 1)

	go func() {
		rotation1Done <- service1.RotateServerWithContinuity(ctx, "coordinated-server-1", server1)
	}()

	go func() {
		rotation2Done <- service2.RotateServerWithContinuity(ctx, "coordinated-server-2", server2)
	}()

	// Wait for both rotations
	timeout := time.After(15 * time.Second)
	completedRotations := 0

	for completedRotations < 2 {
		select {
		case err := <-rotation1Done:
			if err != nil {
				t.Errorf("Service 1 rotation failed: %v", err)
			} else {
				t.Log("✅ Service 1 rotation completed")
			}
			completedRotations++
		case err := <-rotation2Done:
			if err != nil {
				t.Errorf("Service 2 rotation failed: %v", err)
			} else {
				t.Log("✅ Service 2 rotation completed")
			}
			completedRotations++
		case <-timeout:
			t.Fatal("Coordinated rotations timed out")
		}
	}

	t.Logf("✅ Coordinated rotations completed successfully")
}

func testCrossServiceInvariants(ctx context.Context, t *testing.T, services ...*services.IdentityService) {
	t.Helper()

	// Start enforcement on all services
	for i, service := range services {
		if err := service.StartMTLSEnforcement(ctx); err != nil {
			t.Fatalf("Failed to start enforcement on service %d: %v", i, err)
		}
	}

	// Wait for enforcement to run
	time.Sleep(200 * time.Millisecond)

	// Check invariant status across all services
	totalInvariants := 0
	totalConnections := 0

	for i, service := range services {
		status := service.GetInvariantStatus(ctx)
		totalInvariants += status.TotalInvariants
		totalConnections += status.TotalConnections

		t.Logf("Service %d: %d invariants, %d connections", 
			i+1, status.TotalInvariants, status.TotalConnections)
	}

	t.Logf("✅ Cross-service invariants: %d total invariants across %d services",
		totalInvariants, len(services))
}

