// Package application provides use cases that orchestrate domain logic.
package application

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/spiffe/go-spiffe/v2/svid/x509svid"
	"github.com/sufield/ephemos/internal/core/domain"
	"github.com/sufield/ephemos/internal/core/ports"
)

// IdentityRotationService manages automatic identity rotation and renewal.
// This service monitors identity expiration and coordinates rotation through ports.
type IdentityRotationService struct {
	identityProvider ports.IdentityProviderPort
	bundleProvider   ports.BundleProviderPort
	logger           *slog.Logger

	// Rotation configuration
	rotationThreshold time.Duration // When to trigger rotation before expiry
	checkInterval     time.Duration // How often to check for rotation needs
	maxRotationDelay  time.Duration // Maximum random delay for rotation jitter

	// State management
	mu                sync.RWMutex
	isRunning         bool
	stopChan          chan struct{}
	currentSVID       *x509svid.SVID
	lastRotation      time.Time
	rotationCallbacks []RotationCallback
}

// RotationCallback is called when identity rotation occurs.
type RotationCallback func(oldSVID, newSVID *x509svid.SVID)

// IdentityRotationServiceConfig provides configuration for the IdentityRotationService.
type IdentityRotationServiceConfig struct {
	IdentityProvider  ports.IdentityProviderPort
	BundleProvider    ports.BundleProviderPort
	Logger            *slog.Logger
	RotationThreshold time.Duration // Default: 1/3 of certificate lifetime
	CheckInterval     time.Duration // Default: 1 minute
	MaxRotationDelay  time.Duration // Default: 30 seconds
}

// NewIdentityRotationService creates a new IdentityRotationService.
func NewIdentityRotationService(config IdentityRotationServiceConfig) (*IdentityRotationService, error) {
	// Validate required dependencies
	if config.IdentityProvider == nil {
		return nil, fmt.Errorf("identity provider is required")
	}
	if config.BundleProvider == nil {
		return nil, fmt.Errorf("bundle provider is required")
	}

	// Set defaults
	logger := config.Logger
	if logger == nil {
		logger = slog.Default()
	}

	rotationThreshold := config.RotationThreshold
	if rotationThreshold == 0 {
		rotationThreshold = 5 * time.Minute // Conservative default
	}

	checkInterval := config.CheckInterval
	if checkInterval == 0 {
		checkInterval = 1 * time.Minute
	}

	maxRotationDelay := config.MaxRotationDelay
	if maxRotationDelay == 0 {
		maxRotationDelay = 30 * time.Second
	}

	return &IdentityRotationService{
		identityProvider:  config.IdentityProvider,
		bundleProvider:    config.BundleProvider,
		logger:            logger,
		rotationThreshold: rotationThreshold,
		checkInterval:     checkInterval,
		maxRotationDelay:  maxRotationDelay,
		stopChan:          make(chan struct{}),
		rotationCallbacks: make([]RotationCallback, 0),
	}, nil
}

// Start begins the identity rotation monitoring and management.
func (s *IdentityRotationService) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isRunning {
		return fmt.Errorf("rotation service is already running")
	}

	// Get initial SVID
	svid, err := s.identityProvider.GetSVID(ctx)
	if err != nil {
		return fmt.Errorf("failed to get initial SVID: %w", err)
	}

	// Validate initial SVID
	if err := s.validateSVID(ctx, svid); err != nil {
		return fmt.Errorf("initial SVID validation failed: %w", err)
	}

	s.currentSVID = svid
	s.lastRotation = time.Now()
	s.isRunning = true

	// Start monitoring goroutines
	go s.monitorIdentityExpiration(ctx)
	go s.watchIdentityChanges(ctx)

	s.logger.Info("identity rotation service started",
		"check_interval", s.checkInterval,
		"rotation_threshold", s.rotationThreshold)

	return nil
}

// Stop gracefully stops the identity rotation service.
func (s *IdentityRotationService) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isRunning {
		return fmt.Errorf("rotation service is not running")
	}

	close(s.stopChan)
	s.isRunning = false

	s.logger.Info("identity rotation service stopped")
	return nil
}

// RegisterRotationCallback registers a callback to be notified of rotations.
func (s *IdentityRotationService) RegisterRotationCallback(callback RotationCallback) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.rotationCallbacks = append(s.rotationCallbacks, callback)
}

// GetCurrentSVID returns the current valid SVID.
func (s *IdentityRotationService) GetCurrentSVID() (*x509svid.SVID, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.currentSVID == nil {
		return nil, fmt.Errorf("no current SVID available")
	}

	// Check if current SVID is still valid
	if len(s.currentSVID.Certificates) > 0 && time.Now().After(s.currentSVID.Certificates[0].NotAfter) {
		return nil, fmt.Errorf("current SVID has expired")
	}

	return s.currentSVID, nil
}

// ForceRotation triggers an immediate identity rotation.
func (s *IdentityRotationService) ForceRotation(ctx context.Context) error {
	s.logger.Info("forcing identity rotation")

	return s.rotateIdentity(ctx)
}

// monitorIdentityExpiration periodically checks if identity needs rotation.
func (s *IdentityRotationService) monitorIdentityExpiration(ctx context.Context) {
	ticker := time.NewTicker(s.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopChan:
			return
		case <-ticker.C:
			if err := s.checkAndRotateIfNeeded(ctx); err != nil {
				s.logger.Error("rotation check failed", "error", err)
			}
		}
	}
}

// watchIdentityChanges monitors for external identity updates.
func (s *IdentityRotationService) watchIdentityChanges(ctx context.Context) {
	// Attempt to watch for identity changes
	changeChan, err := s.identityProvider.WatchIdentityChanges(ctx)
	if err != nil {
		s.logger.Warn("identity watching not supported", "error", err)
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopChan:
			return
		case newSVID, ok := <-changeChan:
			if !ok {
				s.logger.Warn("identity change channel closed")
				return
			}

			if err := s.handleExternalRotation(ctx, newSVID); err != nil {
				s.logger.Error("failed to handle external rotation", "error", err)
			}
		}
	}
}

// checkAndRotateIfNeeded checks if rotation is needed and performs it.
func (s *IdentityRotationService) checkAndRotateIfNeeded(ctx context.Context) error {
	s.mu.RLock()
	currentSVID := s.currentSVID
	s.mu.RUnlock()

	if currentSVID == nil {
		return fmt.Errorf("no current SVID to check")
	}

	// Check if SVID is expiring soon
	if len(currentSVID.Certificates) > 0 {
		expiresAt := currentSVID.Certificates[0].NotAfter
		if time.Until(expiresAt) < s.rotationThreshold {
			s.logger.Info("SVID expiring soon, initiating rotation",
				"expires_at", expiresAt,
				"threshold", s.rotationThreshold)

			// Add jitter to prevent thundering herd
			jitter := s.calculateRotationJitter()
			time.Sleep(jitter)

			return s.rotateIdentity(ctx)
		}
	}

	return nil
}

// rotateIdentity performs the actual identity rotation.
func (s *IdentityRotationService) rotateIdentity(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Store old SVID for callbacks
	oldSVID := s.currentSVID

	// Refresh identity through provider
	if err := s.identityProvider.RefreshIdentity(ctx); err != nil {
		return fmt.Errorf("failed to refresh identity: %w", err)
	}

	// Get new SVID
	newSVID, err := s.identityProvider.GetSVID(ctx)
	if err != nil {
		return fmt.Errorf("failed to get new SVID after refresh: %w", err)
	}

	// Validate new SVID
	if err := s.validateSVID(ctx, newSVID); err != nil {
		return fmt.Errorf("new SVID validation failed: %w", err)
	}

	// Ensure new SVID is actually different and newer
	if oldSVID != nil && !s.isNewerSVID(oldSVID, newSVID) {
		s.logger.Warn("rotation did not produce newer SVID")
		return fmt.Errorf("rotation did not produce newer SVID")
	}

	// Update current SVID
	s.currentSVID = newSVID
	s.lastRotation = time.Now()

	oldExpiry := time.Time{}
	newExpiry := time.Time{}
	if oldSVID != nil && len(oldSVID.Certificates) > 0 {
		oldExpiry = oldSVID.Certificates[0].NotAfter
	}
	if len(newSVID.Certificates) > 0 {
		newExpiry = newSVID.Certificates[0].NotAfter
	}

	s.logger.Info("identity rotated successfully",
		"old_expiry", oldExpiry,
		"new_expiry", newExpiry)

	// Notify callbacks (do this after releasing the lock)
	go s.notifyRotationCallbacks(oldSVID, newSVID)

	return nil
}

// handleExternalRotation handles identity changes from external sources.
func (s *IdentityRotationService) handleExternalRotation(ctx context.Context, newSVID *x509svid.SVID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Validate new SVID
	if err := s.validateSVID(ctx, newSVID); err != nil {
		return fmt.Errorf("external SVID validation failed: %w", err)
	}

	oldSVID := s.currentSVID

	// Ensure it's actually newer
	if oldSVID != nil && !s.isNewerSVID(oldSVID, newSVID) {
		return fmt.Errorf("external SVID is not newer than current")
	}

	// Update current SVID
	s.currentSVID = newSVID
	s.lastRotation = time.Now()

	newExpiry := time.Time{}
	if len(newSVID.Certificates) > 0 {
		newExpiry = newSVID.Certificates[0].NotAfter
	}

	s.logger.Info("identity updated from external source",
		"new_expiry", newExpiry)

	// Notify callbacks
	go s.notifyRotationCallbacks(oldSVID, newSVID)

	return nil
}

// validateSVID validates an SVID against trust bundle.
func (s *IdentityRotationService) validateSVID(ctx context.Context, svid *x509svid.SVID) error {
	// Basic validation
	if svid == nil {
		return fmt.Errorf("SVID is nil")
	}
	if len(svid.Certificates) == 0 {
		return fmt.Errorf("SVID has no certificates")
	}

	// Get trust bundle for validation
	trustBundle, err := s.bundleProvider.GetTrustBundle(ctx)
	if err != nil {
		return fmt.Errorf("failed to get trust bundle: %w", err)
	}

	// Create domain certificate for validation
	cert, err := domain.NewCertificate(
		svid.Certificates[0],
		svid.PrivateKey,
		svid.Certificates[1:],
	)
	if err != nil {
		return fmt.Errorf("failed to create certificate from SVID: %w", err)
	}

	// Validate against trust bundle
	if err := s.bundleProvider.ValidateCertificateAgainstBundle(ctx, cert); err != nil {
		return fmt.Errorf("SVID not trusted by bundle: %w", err)
	}

	// Use trustBundle to avoid unused variable warning
	_ = trustBundle
	return nil
}

// isNewerSVID checks if new SVID is actually newer than old.
func (s *IdentityRotationService) isNewerSVID(old, new *x509svid.SVID) bool {
	if len(old.Certificates) == 0 || len(new.Certificates) == 0 {
		return len(new.Certificates) > 0
	}

	// Check if issuance time is newer
	if new.Certificates[0].NotBefore.After(old.Certificates[0].NotBefore) {
		return true
	}

	// Check if expiry is later (for same issuance time)
	if new.Certificates[0].NotBefore.Equal(old.Certificates[0].NotBefore) && 
		new.Certificates[0].NotAfter.After(old.Certificates[0].NotAfter) {
		return true
	}

	return false
}

// calculateRotationJitter calculates a random jitter for rotation timing.
func (s *IdentityRotationService) calculateRotationJitter() time.Duration {
	// Simple jitter calculation - can be made more sophisticated
	// Returns a random duration between 0 and maxRotationDelay
	return time.Duration(time.Now().UnixNano() % int64(s.maxRotationDelay))
}

// notifyRotationCallbacks notifies all registered callbacks of rotation.
func (s *IdentityRotationService) notifyRotationCallbacks(oldSVID, newSVID *x509svid.SVID) {
	s.mu.RLock()
	callbacks := make([]RotationCallback, len(s.rotationCallbacks))
	copy(callbacks, s.rotationCallbacks)
	s.mu.RUnlock()

	for _, callback := range callbacks {
		// Call each callback in a goroutine to prevent blocking
		go func(cb RotationCallback) {
			defer func() {
				if r := recover(); r != nil {
					s.logger.Error("rotation callback panicked", "error", r)
				}
			}()
			cb(oldSVID, newSVID)
		}(callback)
	}
}

// GetRotationMetrics returns metrics about rotation operations.
func (s *IdentityRotationService) GetRotationMetrics() *RotationMetrics {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var timeUntilExpiry time.Duration
	if s.currentSVID != nil && len(s.currentSVID.Certificates) > 0 {
		timeUntilExpiry = time.Until(s.currentSVID.Certificates[0].NotAfter)
	}

	return &RotationMetrics{
		LastRotation:    s.lastRotation,
		TimeUntilExpiry: timeUntilExpiry,
		IsRunning:       s.isRunning,
		CallbackCount:   len(s.rotationCallbacks),
	}
}

// RotationMetrics contains metrics about rotation operations.
type RotationMetrics struct {
	LastRotation    time.Time
	TimeUntilExpiry time.Duration
	IsRunning       bool
	CallbackCount   int
}
