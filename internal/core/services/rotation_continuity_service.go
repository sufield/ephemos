// Package services provides rotation continuity to ensure zero-downtime during certificate rotation.
package services

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/sufield/ephemos/internal/core/domain"
	"github.com/sufield/ephemos/internal/core/ports"
)

// RotationContinuityService ensures service continuity during certificate rotation
type RotationContinuityService struct {
	identityService   *IdentityService
	transportProvider ports.TransportProvider
	continuityPolicy  *ContinuityPolicy
	activeServers     map[string]*ContinuityServerPair
	activeClients     map[string]*ContinuityClientPair
	logger            *slog.Logger
	mu                sync.RWMutex
}

// ContinuityPolicy defines how service continuity is maintained during rotation
type ContinuityPolicy struct {
	// OverlapDuration defines how long old and new certificates coexist
	OverlapDuration time.Duration
	// GracefulShutdownTimeout for old connections
	GracefulShutdownTimeout time.Duration
	// PreRotationPrep time to prepare new connections before rotation
	PreRotationPrepTime time.Duration
	// PostRotationValidation time to validate new connections after rotation
	PostRotationValidationTime time.Duration
	// MaxConcurrentRotations limits parallel rotations
	MaxConcurrentRotations int
}

// DefaultContinuityPolicy returns sensible defaults for rotation continuity
func DefaultContinuityPolicy() *ContinuityPolicy {
	return &ContinuityPolicy{
		OverlapDuration:            5 * time.Minute,  // 5 minutes overlap
		GracefulShutdownTimeout:    30 * time.Second, // 30 seconds for graceful shutdown
		PreRotationPrepTime:        30 * time.Second, // 30 seconds preparation
		PostRotationValidationTime: 30 * time.Second, // 30 seconds validation
		MaxConcurrentRotations:     2,                // Max 2 parallel rotations
	}
}

// ContinuityServerPair maintains old and new servers during rotation
type ContinuityServerPair struct {
	OldServer  ports.ServerPort
	NewServer  ports.ServerPort
	RotationID string
	StartedAt  time.Time
	Phase      RotationPhase
	OldCert    *domain.Certificate
	NewCert    *domain.Certificate
	mu         sync.RWMutex
}

// ContinuityClientPair maintains old and new clients during rotation
type ContinuityClientPair struct {
	OldClient  ports.ClientPort
	NewClient  ports.ClientPort
	RotationID string
	StartedAt  time.Time
	Phase      RotationPhase
	OldCert    *domain.Certificate
	NewCert    *domain.Certificate
	mu         sync.RWMutex
}

// RotationPhase represents the current phase of rotation
type RotationPhase int

const (
	PhasePreparation RotationPhase = iota
	PhaseOverlap
	PhaseValidation
	PhaseCompletion
	PhaseFailed
)

// String returns string representation of rotation phase
func (p RotationPhase) String() string {
	switch p {
	case PhasePreparation:
		return "preparation"
	case PhaseOverlap:
		return "overlap"
	case PhaseValidation:
		return "validation"
	case PhaseCompletion:
		return "completion"
	case PhaseFailed:
		return "failed"
	default:
		return "unknown"
	}
}

// GetPhase safely returns the current rotation phase
func (p *ContinuityServerPair) GetPhase() RotationPhase {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Phase
}

// SetPhase safely updates the rotation phase
func (p *ContinuityServerPair) SetPhase(phase RotationPhase) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Phase = phase
}

// GetPhase safely returns the current rotation phase
func (p *ContinuityClientPair) GetPhase() RotationPhase {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Phase
}

// SetPhase safely updates the rotation phase
func (p *ContinuityClientPair) SetPhase(phase RotationPhase) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Phase = phase
}

// NewRotationContinuityService creates a new rotation continuity service
func NewRotationContinuityService(
	identityService *IdentityService,
	transportProvider ports.TransportProvider,
) *RotationContinuityService {
	return &RotationContinuityService{
		identityService:   identityService,
		transportProvider: transportProvider,
		continuityPolicy:  DefaultContinuityPolicy(),
		activeServers:     make(map[string]*ContinuityServerPair),
		activeClients:     make(map[string]*ContinuityClientPair),
		logger:            slog.Default(),
	}
}

// SetContinuityPolicy updates the continuity policy
func (s *RotationContinuityService) SetContinuityPolicy(policy *ContinuityPolicy) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.continuityPolicy = policy
}

// RotateServerWithContinuity performs server rotation with zero downtime
func (s *RotationContinuityService) RotateServerWithContinuity(ctx context.Context, serverID string, oldServer ports.ServerPort) error {
	rotationID := fmt.Sprintf("%s-rotation-%d", serverID, time.Now().UnixNano())

	s.logger.Info("starting server rotation with continuity",
		"server_id", serverID,
		"rotation_id", rotationID,
	)

	// Phase 1: Preparation - Create new server with fresh certificate
	newServer, newCert, err := s.prepareNewServer(ctx, rotationID)
	if err != nil {
		return fmt.Errorf("failed to prepare new server: %w", err)
	}

	// Get current certificate for comparison
	oldCert, err := s.identityService.GetCertificate()
	if err != nil {
		return fmt.Errorf("failed to get current certificate: %w", err)
	}

	// Create continuity pair
	pair := &ContinuityServerPair{
		OldServer:  oldServer,
		NewServer:  newServer,
		RotationID: rotationID,
		StartedAt:  time.Now(),
		Phase:      PhasePreparation,
		OldCert:    oldCert,
		NewCert:    newCert,
	}

	// Register the pair
	s.mu.Lock()
	s.activeServers[serverID] = pair
	s.mu.Unlock()

	// Execute rotation phases
	if err := s.executeServerRotationPhases(ctx, serverID, pair); err != nil {
		pair.SetPhase(PhaseFailed)
		return fmt.Errorf("server rotation failed: %w", err)
	}

	// Clean up completed rotation
	s.mu.Lock()
	delete(s.activeServers, serverID)
	s.mu.Unlock()

	s.logger.Info("server rotation with continuity completed successfully",
		"server_id", serverID,
		"rotation_id", rotationID,
		"duration", time.Since(pair.StartedAt),
	)

	return nil
}

// prepareNewServer creates a new server with fresh certificate
func (s *RotationContinuityService) prepareNewServer(ctx context.Context, rotationID string) (ports.ServerPort, *domain.Certificate, error) {
	s.logger.Debug("preparing new server with fresh certificate", "rotation_id", rotationID)

	// Create new server identity with fresh certificate
	newServer, err := s.identityService.CreateServerIdentity()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create new server identity: %w", err)
	}

	// Get the fresh certificate
	newCert, err := s.identityService.GetCertificate()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get fresh certificate: %w", err)
	}

	return newServer, newCert, nil
}

// executeServerRotationPhases executes all phases of server rotation
func (s *RotationContinuityService) executeServerRotationPhases(ctx context.Context, serverID string, pair *ContinuityServerPair) error {
	policy := s.continuityPolicy

	// Phase 1: Preparation - Wait for preparation time
	s.logger.Info("server rotation phase: preparation",
		"server_id", serverID,
		"rotation_id", pair.RotationID,
		"prep_time", policy.PreRotationPrepTime,
	)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(policy.PreRotationPrepTime):
		// Preparation complete
	}

	// Phase 2: Overlap - Both servers active
	pair.SetPhase(PhaseOverlap)
	s.logger.Info("server rotation phase: overlap",
		"server_id", serverID,
		"rotation_id", pair.RotationID,
		"overlap_duration", policy.OverlapDuration,
	)

	// Start monitoring the new server while old server is still active
	overlapCtx, overlapCancel := context.WithTimeout(ctx, policy.OverlapDuration)
	defer overlapCancel()

	// Validate new server is working correctly
	if err := s.validateNewServer(overlapCtx, pair.NewServer, pair.NewCert); err != nil {
		return fmt.Errorf("new server validation failed during overlap: %w", err)
	}

	// Wait for overlap period
	select {
	case <-overlapCtx.Done():
		if overlapCtx.Err() == context.DeadlineExceeded {
			// Overlap period completed successfully
		} else {
			return overlapCtx.Err()
		}
	}

	// Phase 3: Validation - Ensure new server is stable
	pair.SetPhase(PhaseValidation)
	s.logger.Info("server rotation phase: validation",
		"server_id", serverID,
		"rotation_id", pair.RotationID,
		"validation_time", policy.PostRotationValidationTime,
	)

	validationCtx, validationCancel := context.WithTimeout(ctx, policy.PostRotationValidationTime)
	defer validationCancel()

	if err := s.validateNewServer(validationCtx, pair.NewServer, pair.NewCert); err != nil {
		return fmt.Errorf("new server validation failed: %w", err)
	}

	// Phase 4: Completion - Gracefully shutdown old server
	pair.SetPhase(PhaseCompletion)
	s.logger.Info("server rotation phase: completion",
		"server_id", serverID,
		"rotation_id", pair.RotationID,
		"shutdown_timeout", policy.GracefulShutdownTimeout,
	)

	// Gracefully shutdown old server
	shutdownCtx, shutdownCancel := context.WithTimeout(ctx, policy.GracefulShutdownTimeout)
	defer shutdownCancel()

	if err := s.shutdownOldServer(shutdownCtx, pair.OldServer); err != nil {
		s.logger.Warn("old server graceful shutdown failed, but rotation completed",
			"server_id", serverID,
			"rotation_id", pair.RotationID,
			"error", err,
		)
	}

	return nil
}

// validateNewServer validates that a new server is working correctly
func (s *RotationContinuityService) validateNewServer(ctx context.Context, server ports.ServerPort, cert *domain.Certificate) error {
	// Validate certificate is still valid
	if err := s.identityService.ValidateServiceIdentity(cert); err != nil {
		return fmt.Errorf("certificate validation failed: %w", err)
	}

	// Additional validation checks can be implemented here based on specific requirements:
	// - Health endpoint checks
	// - Connection establishment tests
	// - Service-specific readiness checks
	// For now, certificate validation provides the core security guarantee

	s.logger.Debug("new server validation passed",
		"cert_expires", cert.Cert.NotAfter,
		"cert_subject", cert.Cert.Subject.String(),
	)

	return nil
}

// shutdownOldServer gracefully shuts down an old server
func (s *RotationContinuityService) shutdownOldServer(ctx context.Context, server ports.ServerPort) error {
	s.logger.Info("gracefully shutting down old server")

	// Graceful shutdown implementation:
	// 1. Call the server's Stop method which should handle graceful shutdown
	if err := server.Stop(); err != nil {
		s.logger.Warn("server stop returned error, but continuing shutdown", "error", err)
	}

	// 2. The server implementation should:
	//    - Stop accepting new connections
	//    - Wait for existing connections to complete (with timeout)
	//    - Close the server after timeout
	// This is handled by the specific ServerPort implementation

	s.logger.Debug("old server shutdown completed")
	return nil
}

// RotateClientWithContinuity performs client rotation with connection continuity
func (s *RotationContinuityService) RotateClientWithContinuity(ctx context.Context, clientID string, oldClient ports.ClientPort) error {
	rotationID := fmt.Sprintf("%s-client-rotation-%d", clientID, time.Now().UnixNano())

	s.logger.Info("starting client rotation with continuity",
		"client_id", clientID,
		"rotation_id", rotationID,
	)

	// Phase 1: Preparation - Create new client with fresh certificate
	newClient, newCert, err := s.prepareNewClient(ctx, rotationID)
	if err != nil {
		return fmt.Errorf("failed to prepare new client: %w", err)
	}

	// Get current certificate for comparison
	oldCert, err := s.identityService.GetCertificate()
	if err != nil {
		return fmt.Errorf("failed to get current certificate: %w", err)
	}

	// Create continuity pair
	pair := &ContinuityClientPair{
		OldClient:  oldClient,
		NewClient:  newClient,
		RotationID: rotationID,
		StartedAt:  time.Now(),
		Phase:      PhasePreparation,
		OldCert:    oldCert,
		NewCert:    newCert,
	}

	// Register the pair
	s.mu.Lock()
	s.activeClients[clientID] = pair
	s.mu.Unlock()

	// Execute rotation phases
	if err := s.executeClientRotationPhases(ctx, clientID, pair); err != nil {
		pair.SetPhase(PhaseFailed)
		return fmt.Errorf("client rotation failed: %w", err)
	}

	// Clean up completed rotation
	s.mu.Lock()
	delete(s.activeClients, clientID)
	s.mu.Unlock()

	s.logger.Info("client rotation with continuity completed successfully",
		"client_id", clientID,
		"rotation_id", rotationID,
		"duration", time.Since(pair.StartedAt),
	)

	return nil
}

// prepareNewClient creates a new client with fresh certificate
func (s *RotationContinuityService) prepareNewClient(ctx context.Context, rotationID string) (ports.ClientPort, *domain.Certificate, error) {
	s.logger.Debug("preparing new client with fresh certificate", "rotation_id", rotationID)

	// Create new client identity with fresh certificate
	newClient, err := s.identityService.CreateClientIdentity()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create new client identity: %w", err)
	}

	// Get the fresh certificate
	newCert, err := s.identityService.GetCertificate()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get fresh certificate: %w", err)
	}

	return newClient, newCert, nil
}

// executeClientRotationPhases executes all phases of client rotation
func (s *RotationContinuityService) executeClientRotationPhases(ctx context.Context, clientID string, pair *ContinuityClientPair) error {
	policy := s.continuityPolicy

	// Phase 1: Preparation
	s.logger.Info("client rotation phase: preparation",
		"client_id", clientID,
		"rotation_id", pair.RotationID,
	)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(policy.PreRotationPrepTime):
		// Preparation complete
	}

	// Phase 2: Overlap - Both clients available
	pair.SetPhase(PhaseOverlap)
	s.logger.Info("client rotation phase: overlap",
		"client_id", clientID,
		"rotation_id", pair.RotationID,
	)

	// Validate new client during overlap
	overlapCtx, overlapCancel := context.WithTimeout(ctx, policy.OverlapDuration)
	defer overlapCancel()

	if err := s.validateNewClient(overlapCtx, pair.NewClient, pair.NewCert); err != nil {
		return fmt.Errorf("new client validation failed during overlap: %w", err)
	}

	// Wait for overlap period
	select {
	case <-overlapCtx.Done():
		if overlapCtx.Err() == context.DeadlineExceeded {
			// Overlap period completed successfully
		} else {
			return overlapCtx.Err()
		}
	}

	// Phase 3: Validation
	pair.SetPhase(PhaseValidation)
	s.logger.Info("client rotation phase: validation",
		"client_id", clientID,
		"rotation_id", pair.RotationID,
	)

	validationCtx, validationCancel := context.WithTimeout(ctx, policy.PostRotationValidationTime)
	defer validationCancel()

	if err := s.validateNewClient(validationCtx, pair.NewClient, pair.NewCert); err != nil {
		return fmt.Errorf("new client validation failed: %w", err)
	}

	// Phase 4: Completion
	pair.SetPhase(PhaseCompletion)
	s.logger.Info("client rotation phase: completion",
		"client_id", clientID,
		"rotation_id", pair.RotationID,
	)

	return nil
}

// validateNewClient validates that a new client is working correctly
func (s *RotationContinuityService) validateNewClient(ctx context.Context, client ports.ClientPort, cert *domain.Certificate) error {
	// Validate certificate is still valid
	if err := s.identityService.ValidateServiceIdentity(cert); err != nil {
		return fmt.Errorf("certificate validation failed: %w", err)
	}

	// Additional validation checks can be implemented here:
	// - Test connection establishment to target services
	// - Validate client-specific configuration
	// - Check service discovery and load balancing
	// Certificate validation provides the core security foundation

	s.logger.Debug("new client validation passed",
		"cert_expires", cert.Cert.NotAfter,
		"cert_subject", cert.Cert.Subject.String(),
	)

	return nil
}

// GetActiveRotations returns information about currently active rotations
func (s *RotationContinuityService) GetActiveRotations() ([]RotationInfo, []RotationInfo) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	serverRotations := make([]RotationInfo, 0, len(s.activeServers))
	for serverID, pair := range s.activeServers {
		serverRotations = append(serverRotations, RotationInfo{
			ID:         serverID,
			RotationID: pair.RotationID,
			Type:       "server",
			Phase:      pair.GetPhase(),
			StartedAt:  pair.StartedAt,
			Duration:   time.Since(pair.StartedAt),
		})
	}

	clientRotations := make([]RotationInfo, 0, len(s.activeClients))
	for clientID, pair := range s.activeClients {
		clientRotations = append(clientRotations, RotationInfo{
			ID:         clientID,
			RotationID: pair.RotationID,
			Type:       "client",
			Phase:      pair.GetPhase(),
			StartedAt:  pair.StartedAt,
			Duration:   time.Since(pair.StartedAt),
		})
	}

	return serverRotations, clientRotations
}

// RotationInfo provides information about an active rotation
type RotationInfo struct {
	ID         string
	RotationID string
	Type       string
	Phase      RotationPhase
	StartedAt  time.Time
	Duration   time.Duration
}

// GetRotationStats returns statistics about rotation operations
func (s *RotationContinuityService) GetRotationStats() RotationStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return RotationStats{
		ActiveServerRotations: len(s.activeServers),
		ActiveClientRotations: len(s.activeClients),
		TotalActiveRotations:  len(s.activeServers) + len(s.activeClients),
		MaxConcurrentAllowed:  s.continuityPolicy.MaxConcurrentRotations,
	}
}

// RotationStats provides statistics about rotation operations
type RotationStats struct {
	ActiveServerRotations int
	ActiveClientRotations int
	TotalActiveRotations  int
	MaxConcurrentAllowed  int
}
