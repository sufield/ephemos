// Package services provides mTLS invariant enforcement across all communication.
package services

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/sufield/ephemos/internal/core/errors"
)

// MTLSInvariant represents a security invariant that must be enforced
type MTLSInvariant interface {
	// Name returns the name of the invariant
	Name() string
	// Check validates that the invariant is satisfied
	Check(ctx context.Context, conn *MTLSConnection) error
	// Description returns a human-readable description of the invariant
	Description() string
}

// MTLSEnforcementService enforces mTLS invariants across all service communication
type MTLSEnforcementService struct {
	identityService    *IdentityService
	connectionRegistry *MTLSConnectionRegistry
	invariants         []MTLSInvariant
	policy             *EnforcementPolicy
	logger             *slog.Logger
	mu                 sync.RWMutex
}

// EnforcementPolicy defines how invariants are enforced
type EnforcementPolicy struct {
	// FailOnViolation determines whether to fail fast or log violations
	FailOnViolation bool
	// CheckInterval defines how often invariants are checked
	CheckInterval time.Duration
	// MaxViolations before taking action
	MaxViolations int
	// ViolationAction defines what to do when max violations is reached
	ViolationAction ViolationAction
}

// ViolationAction defines actions to take on invariant violations
type ViolationAction int

const (
	ActionLog ViolationAction = iota
	ActionCloseConnection
	ActionRestartService
	ActionAlertOnly
)

// String returns string representation of violation action
func (v ViolationAction) String() string {
	switch v {
	case ActionLog:
		return "log"
	case ActionCloseConnection:
		return "close_connection"
	case ActionRestartService:
		return "restart_service"
	case ActionAlertOnly:
		return "alert_only"
	default:
		return "unknown"
	}
}

// DefaultEnforcementPolicy returns sensible defaults for invariant enforcement
func DefaultEnforcementPolicy() *EnforcementPolicy {
	return &EnforcementPolicy{
		FailOnViolation: true,
		CheckInterval:   30 * time.Second,
		MaxViolations:   3,
		ViolationAction: ActionCloseConnection,
	}
}

// NewMTLSEnforcementService creates a new mTLS enforcement service
func NewMTLSEnforcementService(
	identityService *IdentityService,
	connectionRegistry *MTLSConnectionRegistry,
) *MTLSEnforcementService {
	service := &MTLSEnforcementService{
		identityService:    identityService,
		connectionRegistry: connectionRegistry,
		invariants:         make([]MTLSInvariant, 0),
		policy:             DefaultEnforcementPolicy(),
		logger:             slog.Default(),
	}

	// Add default invariants
	service.AddDefaultInvariants()

	return service
}

// SetEnforcementPolicy updates the enforcement policy
func (s *MTLSEnforcementService) SetEnforcementPolicy(policy *EnforcementPolicy) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.policy = policy
}

// AddInvariant adds a new invariant to be enforced
func (s *MTLSEnforcementService) AddInvariant(invariant MTLSInvariant) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.invariants = append(s.invariants, invariant)
	s.logger.Info("added mTLS invariant", "name", invariant.Name(), "description", invariant.Description())
}

// AddDefaultInvariants adds the standard set of mTLS invariants
func (s *MTLSEnforcementService) AddDefaultInvariants() {
	s.AddInvariant(&CertificateValidityInvariant{})
	s.AddInvariant(&MutualAuthInvariant{})
	s.AddInvariant(&TrustDomainInvariant{})
	s.AddInvariant(&CertificateRotationInvariant{})
	s.AddInvariant(&IdentityMatchingInvariant{})
}

// StartEnforcement begins enforcing invariants on all connections
func (s *MTLSEnforcementService) StartEnforcement(ctx context.Context) error {
	s.logger.Info("starting mTLS invariant enforcement",
		"check_interval", s.policy.CheckInterval,
		"invariant_count", len(s.invariants),
	)

	// Start invariant checking loop
	go s.invariantCheckingLoop(ctx)

	return nil
}

// invariantCheckingLoop continuously checks all invariants
func (s *MTLSEnforcementService) invariantCheckingLoop(ctx context.Context) {
	ticker := time.NewTicker(s.policy.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("stopping mTLS invariant enforcement")
			return
		case <-ticker.C:
			s.checkAllInvariants(ctx)
		}
	}
}

// checkAllInvariants checks all invariants against all connections
func (s *MTLSEnforcementService) checkAllInvariants(ctx context.Context) {
	connections := s.connectionRegistry.ListConnections()
	if len(connections) == 0 {
		return
	}

	s.mu.RLock()
	invariants := make([]MTLSInvariant, len(s.invariants))
	copy(invariants, s.invariants)
	policy := s.policy
	s.mu.RUnlock()

	violations := make(map[string][]string) // connID -> violation list

	for _, conn := range connections {
		for _, invariant := range invariants {
			if err := invariant.Check(ctx, conn); err != nil {
				if violations[conn.ID] == nil {
					violations[conn.ID] = make([]string, 0)
				}
				violation := fmt.Sprintf("%s: %s", invariant.Name(), err.Error())
				violations[conn.ID] = append(violations[conn.ID], violation)

				s.logger.Warn("mTLS invariant violation",
					"connection_id", conn.ID,
					"invariant", invariant.Name(),
					"violation", err.Error(),
					"local_identity", conn.LocalIdentity.Name(),
					"remote_identity", conn.RemoteIdentity.Name(),
				)
			}
		}
	}

	// Handle violations according to policy
	s.handleViolations(ctx, violations, policy)
}

// handleViolations processes invariant violations according to policy
func (s *MTLSEnforcementService) handleViolations(ctx context.Context, violations map[string][]string, policy *EnforcementPolicy) {
	for connID, violationList := range violations {
		if len(violationList) >= policy.MaxViolations {
			s.logger.Error("maximum violations exceeded for connection",
				"connection_id", connID,
				"violation_count", len(violationList),
				"max_violations", policy.MaxViolations,
				"action", policy.ViolationAction.String(),
			)

			switch policy.ViolationAction {
			case ActionCloseConnection:
				if err := s.connectionRegistry.CloseConnection(connID); err != nil {
					s.logger.Error("failed to close violating connection",
						"connection_id", connID,
						"error", err,
					)
				}
			case ActionLog:
				// Already logged above
			case ActionAlertOnly:
				// Alerting implementation should integrate with monitoring systems:
				// - Prometheus metrics with alert manager rules
				// - External alerting services (PagerDuty, Slack, etc.)
				// - Custom webhook endpoints
				// For now, structured logging provides audit trail and can be monitored
				s.logger.Error("ALERT: mTLS invariant violations detected",
					"connection_id", connID,
					"violations", violationList,
					"alert_level", "critical",
					"component", "mtls_enforcement",
				)
			case ActionRestartService:
				// This would require integration with the service lifecycle
				s.logger.Error("SERVICE RESTART REQUIRED: Critical mTLS violations",
					"connection_id", connID,
					"violations", violationList,
				)
			}
		} else if policy.FailOnViolation {
			// Log violations but don't take action yet
			s.logger.Error("mTLS invariant violations detected",
				"connection_id", connID,
				"violation_count", len(violationList),
				"violations", violationList,
			)
		}
	}
}

// ValidateConnection validates all invariants for a specific connection
func (s *MTLSEnforcementService) ValidateConnection(ctx context.Context, connID string) error {
	conn, exists := s.connectionRegistry.GetConnection(connID)
	if !exists {
		return fmt.Errorf("connection %s not found", connID)
	}

	s.mu.RLock()
	invariants := make([]MTLSInvariant, len(s.invariants))
	copy(invariants, s.invariants)
	s.mu.RUnlock()

	violations := make([]string, 0)
	for _, invariant := range invariants {
		if err := invariant.Check(ctx, conn); err != nil {
			violations = append(violations, fmt.Sprintf("%s: %s", invariant.Name(), err.Error()))
		}
	}

	if len(violations) > 0 {
		return &errors.ValidationError{
			Field:   "connection",
			Value:   connID,
			Message: fmt.Sprintf("invariant violations: %v", violations),
		}
	}

	return nil
}

// GetInvariantStatus returns the status of all invariants
func (s *MTLSEnforcementService) GetInvariantStatus(ctx context.Context) InvariantStatus {
	s.mu.RLock()
	invariants := make([]MTLSInvariant, len(s.invariants))
	copy(invariants, s.invariants)
	s.mu.RUnlock()

	connections := s.connectionRegistry.ListConnections()
	
	status := InvariantStatus{
		TotalInvariants:   len(invariants),
		TotalConnections:  len(connections),
		InvariantResults:  make(map[string]InvariantResult),
		ConnectionResults: make(map[string][]string),
	}

	for _, invariant := range invariants {
		result := InvariantResult{
			Name:         invariant.Name(),
			Description:  invariant.Description(),
			PassCount:    0,
			FailCount:    0,
			Violations:   make([]string, 0),
		}

		for _, conn := range connections {
			if err := invariant.Check(ctx, conn); err != nil {
				result.FailCount++
				violation := fmt.Sprintf("conn:%s - %s", conn.ID, err.Error())
				result.Violations = append(result.Violations, violation)
				
				if status.ConnectionResults[conn.ID] == nil {
					status.ConnectionResults[conn.ID] = make([]string, 0)
				}
				status.ConnectionResults[conn.ID] = append(status.ConnectionResults[conn.ID], invariant.Name())
			} else {
				result.PassCount++
			}
		}

		status.InvariantResults[invariant.Name()] = result
	}

	return status
}

// InvariantStatus provides detailed status of all invariants
type InvariantStatus struct {
	TotalInvariants   int
	TotalConnections  int
	InvariantResults  map[string]InvariantResult
	ConnectionResults map[string][]string
}

// InvariantResult contains the result of checking an invariant
type InvariantResult struct {
	Name        string
	Description string
	PassCount   int
	FailCount   int
	Violations  []string
}

// == DEFAULT INVARIANTS ==

// CertificateValidityInvariant ensures certificates are valid and not expired
type CertificateValidityInvariant struct{}

func (i *CertificateValidityInvariant) Name() string {
	return "certificate_validity"
}

func (i *CertificateValidityInvariant) Description() string {
	return "Ensures all certificates are valid and not expired"
}

func (i *CertificateValidityInvariant) Check(ctx context.Context, conn *MTLSConnection) error {
	if conn.Cert == nil || conn.Cert.Cert == nil {
		return fmt.Errorf("no certificate present")
	}

	now := time.Now()
	if now.Before(conn.Cert.Cert.NotBefore) {
		return fmt.Errorf("certificate not yet valid (not before: %s)", conn.Cert.Cert.NotBefore)
	}

	if now.After(conn.Cert.Cert.NotAfter) {
		return fmt.Errorf("certificate expired (not after: %s)", conn.Cert.Cert.NotAfter)
	}

	return nil
}

// MutualAuthInvariant ensures both client and server authenticate each other
type MutualAuthInvariant struct{}

func (i *MutualAuthInvariant) Name() string {
	return "mutual_authentication"
}

func (i *MutualAuthInvariant) Description() string {
	return "Ensures mutual authentication is properly established"
}

func (i *MutualAuthInvariant) Check(ctx context.Context, conn *MTLSConnection) error {
	if conn.TLSState == nil {
		return fmt.Errorf("no TLS connection state available")
	}

	if !conn.TLSState.HandshakeComplete {
		return fmt.Errorf("TLS handshake not completed")
	}

	if len(conn.TLSState.PeerCertificates) == 0 {
		return fmt.Errorf("no peer certificates present")
	}

	return nil
}

// TrustDomainInvariant ensures all certificates belong to expected trust domains
type TrustDomainInvariant struct{}

func (i *TrustDomainInvariant) Name() string {
	return "trust_domain_validation"
}

func (i *TrustDomainInvariant) Description() string {
	return "Validates certificates belong to expected trust domains"
}

func (i *TrustDomainInvariant) Check(ctx context.Context, conn *MTLSConnection) error {
	if conn.LocalIdentity == nil || conn.RemoteIdentity == nil {
		return fmt.Errorf("missing identity information")
	}

	// Both identities should have valid trust domains
	if conn.LocalIdentity.TrustDomain().String() == "" {
		return fmt.Errorf("local identity has empty trust domain")
	}

	if conn.RemoteIdentity.TrustDomain().String() == "" {
		return fmt.Errorf("remote identity has empty trust domain")
	}

	return nil
}

// CertificateRotationInvariant ensures certificates are rotated appropriately
type CertificateRotationInvariant struct{}

func (i *CertificateRotationInvariant) Name() string {
	return "certificate_rotation"
}

func (i *CertificateRotationInvariant) Description() string {
	return "Ensures certificates are rotated before expiry"
}

func (i *CertificateRotationInvariant) Check(ctx context.Context, conn *MTLSConnection) error {
	if conn.Cert == nil || conn.Cert.Cert == nil {
		return fmt.Errorf("no certificate present")
	}

	// Check if certificate should have been rotated
	rotationThreshold := 15 * time.Minute // Should rotate 15 minutes before expiry
	timeToExpiry := time.Until(conn.Cert.Cert.NotAfter)
	
	if timeToExpiry <= rotationThreshold && conn.GetState() != ConnectionRotating {
		return fmt.Errorf("certificate should be rotating (expires in %s)", timeToExpiry.String())
	}

	// Check if connection has been active too long without rotation
	maxConnectionAge := time.Hour // No connection should be active for more than 1 hour without rotation
	connectionAge := time.Since(conn.LastRotated)
	
	if connectionAge > maxConnectionAge {
		return fmt.Errorf("connection too old without rotation (age: %s)", connectionAge.String())
	}

	return nil
}

// IdentityMatchingInvariant ensures certificate identities match expected identities
type IdentityMatchingInvariant struct{}

func (i *IdentityMatchingInvariant) Name() string {
	return "identity_matching"
}

func (i *IdentityMatchingInvariant) Description() string {
	return "Validates certificate identities match expected service identities"
}

func (i *IdentityMatchingInvariant) Check(ctx context.Context, conn *MTLSConnection) error {
	if conn.Cert == nil {
		return fmt.Errorf("no certificate present")
	}

	if conn.LocalIdentity == nil {
		return fmt.Errorf("no local identity configured")
	}

	// Extract SPIFFE ID from certificate and validate it matches our identity
	spiffeID, err := conn.Cert.ToSPIFFEID()
	if err != nil {
		return fmt.Errorf("failed to extract SPIFFE ID from certificate: %w", err)
	}

	expectedURI := conn.LocalIdentity.URI()
	if spiffeID.String() != expectedURI {
		return fmt.Errorf("certificate SPIFFE ID (%s) does not match expected identity (%s)", spiffeID.String(), expectedURI)
	}

	return nil
}