// Package services provides enhanced mTLS connection management with rotation support.
package services

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/sufield/ephemos/internal/core/domain"
)

// ConnectionState represents the state of an mTLS connection
type ConnectionState int

const (
	ConnectionActive ConnectionState = iota
	ConnectionRotating
	ConnectionFailed
	ConnectionClosed
)

// String returns string representation of connection state
func (c ConnectionState) String() string {
	switch c {
	case ConnectionActive:
		return "active"
	case ConnectionRotating:
		return "rotating"
	case ConnectionFailed:
		return "failed"
	case ConnectionClosed:
		return "closed"
	default:
		return "unknown"
	}
}

// MTLSConnection represents an active mTLS connection with rotation tracking
type MTLSConnection struct {
	ID             string
	RemoteIdentity *domain.ServiceIdentity
	LocalIdentity  *domain.ServiceIdentity
	State          ConnectionState
	EstablishedAt  time.Time
	LastRotated    time.Time
	Cert           *domain.Certificate
	TLSState       *tls.ConnectionState
	mu             sync.RWMutex
}

// GetState safely returns the current connection state
func (c *MTLSConnection) GetState() ConnectionState {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.State
}

// SetState safely updates the connection state
func (c *MTLSConnection) SetState(state ConnectionState) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.State = state
}

// UpdateRotation marks the connection as recently rotated
func (c *MTLSConnection) UpdateRotation(newCert *domain.Certificate) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.LastRotated = time.Now()
	c.Cert = newCert
	c.State = ConnectionActive
}

// DisplaySummary returns a formatted summary of the connection (facade method).
func (c *MTLSConnection) DisplaySummary() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return fmt.Sprintf("Connection %s (state: %v)\nLocal: %s\nRemote: %s", 
		c.ID, c.State, c.LocalIdentity.URI(), c.RemoteIdentity.URI())
}

// IsHealthy returns true if the connection is active and certificate is not expired (facade method).
func (c *MTLSConnection) IsHealthy() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.State == ConnectionActive && (c.Cert == nil || !c.Cert.IsExpired())
}

// GetExpiryInfo returns certificate expiration time and whether it's expiring soon (facade method).
func (c *MTLSConnection) GetExpiryInfo() (time.Time, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.Cert == nil {
		return time.Time{}, true // Treat nil certificate as expiring
	}
	return c.Cert.ExpiresAt(), c.Cert.IsExpiringWithin(time.Hour)
}

// MTLSConnectionRegistry tracks and maintains active mTLS connections with rotation support
type MTLSConnectionRegistry struct {
	identityService *IdentityService
	connections     map[string]*MTLSConnection
	rotationPolicy  *RotationPolicy
	observers       []RotationObserver
	logger          *slog.Logger
	mu              sync.RWMutex
}

// RotationPolicy defines when and how certificate rotation occurs
type RotationPolicy struct {
	// PreRotationThreshold triggers rotation before certificate expiry
	PreRotationThreshold time.Duration
	// ForceRotationInterval forces rotation even if certificate is valid
	ForceRotationInterval time.Duration
	// MaxRetryAttempts for failed rotations
	MaxRetryAttempts int
	// RetryBackoff base duration for exponential backoff
	RetryBackoff time.Duration
}

// DefaultRotationPolicy returns sensible defaults for certificate rotation
func DefaultRotationPolicy() *RotationPolicy {
	return &RotationPolicy{
		PreRotationThreshold:  15 * time.Minute, // Rotate 15 minutes before expiry
		ForceRotationInterval: 30 * time.Minute, // Force rotation every 30 minutes
		MaxRetryAttempts:      3,
		RetryBackoff:          time.Second,
	}
}

// RotationObserver allows components to observe rotation events
type RotationObserver interface {
	OnRotationStarted(connID string, reason string)
	OnRotationCompleted(connID string, oldCert, newCert *domain.Certificate)
	OnRotationFailed(connID string, err error)
}

// NewMTLSConnectionRegistry creates a new connection registry with rotation support
func NewMTLSConnectionRegistry(identityService *IdentityService) *MTLSConnectionRegistry {
	return &MTLSConnectionRegistry{
		identityService: identityService,
		connections:     make(map[string]*MTLSConnection),
		rotationPolicy:  DefaultRotationPolicy(),
		observers:       make([]RotationObserver, 0),
		logger:          slog.Default(),
	}
}

// SetRotationPolicy updates the rotation policy
func (r *MTLSConnectionRegistry) SetRotationPolicy(policy *RotationPolicy) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rotationPolicy = policy
}

// AddRotationObserver adds an observer for rotation events
func (r *MTLSConnectionRegistry) AddRotationObserver(observer RotationObserver) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.observers = append(r.observers, observer)
}

// EstablishConnection creates a new mTLS connection with the given parameters
func (r *MTLSConnectionRegistry) EstablishConnection(ctx context.Context, connID string, remoteIdentity *domain.ServiceIdentity, cert *domain.Certificate, localIdentity *domain.ServiceIdentity) (*MTLSConnection, error) {

	// Create connection
	conn := &MTLSConnection{
		ID:             connID,
		RemoteIdentity: remoteIdentity,
		LocalIdentity:  localIdentity,
		State:          ConnectionActive,
		EstablishedAt:  time.Now(),
		LastRotated:    time.Now(),
		Cert:           cert,
	}

	// Register connection
	r.mu.Lock()
	r.connections[connID] = conn
	r.mu.Unlock()

	r.logger.Info("mTLS connection established",
		"connection_id", connID,
		"local_identity", localIdentity.Name(),
		"remote_identity", remoteIdentity.Name(),
		"cert_expires", cert.Cert.NotAfter,
	)

	// Start rotation monitoring for this connection
	go r.monitorConnectionForRotation(ctx, connID)

	return conn, nil
}

// GetConnection retrieves an active connection by ID
func (r *MTLSConnectionRegistry) GetConnection(connID string) (*MTLSConnection, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	conn, exists := r.connections[connID]
	return conn, exists
}

// ListConnections returns all active connections
func (r *MTLSConnectionRegistry) ListConnections() []*MTLSConnection {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	connections := make([]*MTLSConnection, 0, len(r.connections))
	for _, conn := range r.connections {
		connections = append(connections, conn)
	}
	return connections
}

// CloseConnection closes and removes a connection
func (r *MTLSConnectionRegistry) CloseConnection(connID string) error {
	r.mu.Lock()
	conn, exists := r.connections[connID]
	if exists {
		conn.SetState(ConnectionClosed)
		delete(r.connections, connID)
	}
	r.mu.Unlock()

	if exists {
		r.logger.Info("mTLS connection closed",
			"connection_id", connID,
			"local_identity", conn.LocalIdentity.Name(),
			"remote_identity", conn.RemoteIdentity.Name(),
		)
	}

	return nil
}

// monitorConnectionForRotation monitors a connection for rotation needs
func (r *MTLSConnectionRegistry) monitorConnectionForRotation(ctx context.Context, connID string) {
	ticker := time.NewTicker(time.Minute) // Check every minute
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := r.checkAndRotateConnection(ctx, connID); err != nil {
				r.logger.Error("connection rotation check failed",
					"connection_id", connID,
					"error", err,
				)
			}
		}
	}
}

// checkAndRotateConnection checks if a connection needs rotation and performs it
func (r *MTLSConnectionRegistry) checkAndRotateConnection(ctx context.Context, connID string) error {
	r.mu.RLock()
	conn, exists := r.connections[connID]
	policy := r.rotationPolicy
	r.mu.RUnlock()

	if !exists || conn.GetState() == ConnectionClosed {
		return nil // Connection no longer exists or is closed
	}

	// Check if rotation is needed
	rotationReason := r.determineRotationReason(conn, policy)
	if rotationReason == "" {
		return nil // No rotation needed
	}

	// Perform rotation
	return r.rotateConnection(ctx, connID, rotationReason)
}

// determineRotationReason checks if rotation is needed and returns the reason
func (r *MTLSConnectionRegistry) determineRotationReason(conn *MTLSConnection, policy *RotationPolicy) string {
	now := time.Now()
	
	// Check if certificate is expiring soon
	if conn.Cert != nil && conn.Cert.Cert != nil {
		timeToExpiry := time.Until(conn.Cert.Cert.NotAfter)
		if timeToExpiry <= policy.PreRotationThreshold {
			return fmt.Sprintf("certificate_expiring_in_%s", timeToExpiry.String())
		}
	}

	// Check if force rotation interval has passed
	if now.Sub(conn.LastRotated) >= policy.ForceRotationInterval {
		return fmt.Sprintf("force_rotation_after_%s", policy.ForceRotationInterval.String())
	}

	// Check if connection is in a failed state
	if conn.GetState() == ConnectionFailed {
		return "connection_failed"
	}

	return "" // No rotation needed
}

// rotateConnection performs the actual certificate rotation for a connection
func (r *MTLSConnectionRegistry) rotateConnection(ctx context.Context, connID, reason string) error {
	r.mu.Lock()
	conn, exists := r.connections[connID]
	r.mu.Unlock()

	if !exists {
		return fmt.Errorf("connection %s not found", connID)
	}

	// Set connection to rotating state
	conn.SetState(ConnectionRotating)

	// Notify observers
	r.notifyRotationStarted(connID, reason)

	r.logger.Info("starting certificate rotation",
		"connection_id", connID,
		"reason", reason,
		"local_identity", conn.LocalIdentity.Name(),
		"remote_identity", conn.RemoteIdentity.Name(),
	)

	// Get new certificate
	newCert, err := r.identityService.GetCertificate()
	if err != nil {
		conn.SetState(ConnectionFailed)
		r.notifyRotationFailed(connID, err)
		return fmt.Errorf("failed to get new certificate for rotation: %w", err)
	}

	// Validate new certificate
	if err := r.identityService.ValidateServiceIdentity(newCert); err != nil {
		conn.SetState(ConnectionFailed)
		r.notifyRotationFailed(connID, err)
		return fmt.Errorf("new certificate validation failed: %w", err)
	}

	oldCert := conn.Cert

	// Update connection with new certificate
	conn.UpdateRotation(newCert)

	// Notify observers
	r.notifyRotationCompleted(connID, oldCert, newCert)

	r.logger.Info("certificate rotation completed successfully",
		"connection_id", connID,
		"reason", reason,
		"old_cert_expires", oldCert.Cert.NotAfter,
		"new_cert_expires", newCert.Cert.NotAfter,
		"local_identity", conn.LocalIdentity.Name(),
		"remote_identity", conn.RemoteIdentity.Name(),
	)

	return nil
}

// notifyRotationStarted notifies observers that rotation has started
func (r *MTLSConnectionRegistry) notifyRotationStarted(connID, reason string) {
	r.mu.RLock()
	observers := make([]RotationObserver, len(r.observers))
	copy(observers, r.observers)
	r.mu.RUnlock()

	for _, observer := range observers {
		observer.OnRotationStarted(connID, reason)
	}
}

// notifyRotationCompleted notifies observers that rotation has completed
func (r *MTLSConnectionRegistry) notifyRotationCompleted(connID string, oldCert, newCert *domain.Certificate) {
	r.mu.RLock()
	observers := make([]RotationObserver, len(r.observers))
	copy(observers, r.observers)
	r.mu.RUnlock()

	for _, observer := range observers {
		observer.OnRotationCompleted(connID, oldCert, newCert)
	}
}

// notifyRotationFailed notifies observers that rotation has failed
func (r *MTLSConnectionRegistry) notifyRotationFailed(connID string, err error) {
	r.mu.RLock()
	observers := make([]RotationObserver, len(r.observers))
	copy(observers, r.observers)
	r.mu.RUnlock()

	for _, observer := range observers {
		observer.OnRotationFailed(connID, err)
	}
}

// GetConnectionStats returns statistics about managed connections
func (r *MTLSConnectionRegistry) GetConnectionStats() ConnectionStats {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats := ConnectionStats{
		TotalConnections: len(r.connections),
		StateCount:       make(map[ConnectionState]int),
	}

	for _, conn := range r.connections {
		state := conn.GetState()
		stats.StateCount[state]++
		
		// Update oldest connection
		if stats.OldestConnection.IsZero() || conn.EstablishedAt.Before(stats.OldestConnection) {
			stats.OldestConnection = conn.EstablishedAt
		}

		// Update newest connection
		if stats.NewestConnection.IsZero() || conn.EstablishedAt.After(stats.NewestConnection) {
			stats.NewestConnection = conn.EstablishedAt
		}

		// Update last rotation
		if stats.LastRotation.IsZero() || conn.LastRotated.After(stats.LastRotation) {
			stats.LastRotation = conn.LastRotated
		}
	}

	return stats
}

// ConnectionStats provides statistics about managed connections
type ConnectionStats struct {
	TotalConnections  int
	StateCount        map[ConnectionState]int
	OldestConnection  time.Time
	NewestConnection  time.Time
	LastRotation      time.Time
}

// LogRotationObserver is a basic observer that logs rotation events
type LogRotationObserver struct {
	logger *slog.Logger
}

// NewLogRotationObserver creates a new logging rotation observer
func NewLogRotationObserver(logger *slog.Logger) *LogRotationObserver {
	if logger == nil {
		logger = slog.Default()
	}
	return &LogRotationObserver{logger: logger}
}

// OnRotationStarted logs rotation start events
func (o *LogRotationObserver) OnRotationStarted(connID, reason string) {
	o.logger.Info("certificate rotation started",
		"connection_id", connID,
		"reason", reason,
	)
}

// OnRotationCompleted logs rotation completion events
func (o *LogRotationObserver) OnRotationCompleted(connID string, oldCert, newCert *domain.Certificate) {
	o.logger.Info("certificate rotation completed",
		"connection_id", connID,
		"old_cert_expires", oldCert.Cert.NotAfter,
		"new_cert_expires", newCert.Cert.NotAfter,
	)
}

// OnRotationFailed logs rotation failure events
func (o *LogRotationObserver) OnRotationFailed(connID string, err error) {
	o.logger.Error("certificate rotation failed",
		"connection_id", connID,
		"error", err.Error(),
	)
}