// Package application provides use cases that orchestrate domain logic.
package application

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

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
	currentIdentity   *domain.IdentityDocument
	lastRotation      time.Time
	rotationCallbacks []RotationCallback
}

// RotationCallback is called when identity rotation occurs.
type RotationCallback func(oldIdentity, newIdentity *domain.IdentityDocument)

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
	
	// Get initial identity
	identityDoc, err := s.identityProvider.GetIdentityDocument(ctx)
	if err != nil {
		return fmt.Errorf("failed to get initial identity: %w", err)
	}
	
	// Validate initial identity
	if err := s.validateIdentity(ctx, identityDoc); err != nil {
		return fmt.Errorf("initial identity validation failed: %w", err)
	}
	
	s.currentIdentity = identityDoc
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

// GetCurrentIdentity returns the current valid identity.
func (s *IdentityRotationService) GetCurrentIdentity() (*domain.IdentityDocument, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if s.currentIdentity == nil {
		return nil, fmt.Errorf("no current identity available")
	}
	
	// Check if current identity is still valid
	if s.currentIdentity.IsExpired(time.Now()) {
		return nil, fmt.Errorf("current identity has expired")
	}
	
	return s.currentIdentity, nil
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
		case newIdentity, ok := <-changeChan:
			if !ok {
				s.logger.Warn("identity change channel closed")
				return
			}
			
			if err := s.handleExternalRotation(ctx, newIdentity); err != nil {
				s.logger.Error("failed to handle external rotation", "error", err)
			}
		}
	}
}

// checkAndRotateIfNeeded checks if rotation is needed and performs it.
func (s *IdentityRotationService) checkAndRotateIfNeeded(ctx context.Context) error {
	s.mu.RLock()
	currentIdentity := s.currentIdentity
	s.mu.RUnlock()
	
	if currentIdentity == nil {
		return fmt.Errorf("no current identity to check")
	}
	
	// Check if identity is expiring soon
	if currentIdentity.IsExpiringSoon(s.rotationThreshold) {
		s.logger.Info("identity expiring soon, initiating rotation",
			"expires_at", currentIdentity.ValidUntil(),
			"threshold", s.rotationThreshold)
		
		// Add jitter to prevent thundering herd
		jitter := s.calculateRotationJitter()
		time.Sleep(jitter)
		
		return s.rotateIdentity(ctx)
	}
	
	return nil
}

// rotateIdentity performs the actual identity rotation.
func (s *IdentityRotationService) rotateIdentity(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Store old identity for callbacks
	oldIdentity := s.currentIdentity
	
	// Refresh identity through provider
	if err := s.identityProvider.RefreshIdentity(ctx); err != nil {
		return fmt.Errorf("failed to refresh identity: %w", err)
	}
	
	// Get new identity
	newIdentity, err := s.identityProvider.GetIdentityDocument(ctx)
	if err != nil {
		return fmt.Errorf("failed to get new identity after refresh: %w", err)
	}
	
	// Validate new identity
	if err := s.validateIdentity(ctx, newIdentity); err != nil {
		return fmt.Errorf("new identity validation failed: %w", err)
	}
	
	// Ensure new identity is actually different and newer
	if oldIdentity != nil && !s.isNewerIdentity(oldIdentity, newIdentity) {
		s.logger.Warn("rotation did not produce newer identity")
		return fmt.Errorf("rotation did not produce newer identity")
	}
	
	// Update current identity
	s.currentIdentity = newIdentity
	s.lastRotation = time.Now()
	
	s.logger.Info("identity rotated successfully",
		"old_expiry", oldIdentity.ValidUntil(),
		"new_expiry", newIdentity.ValidUntil())
	
	// Notify callbacks (do this after releasing the lock)
	go s.notifyRotationCallbacks(oldIdentity, newIdentity)
	
	return nil
}

// handleExternalRotation handles identity changes from external sources.
func (s *IdentityRotationService) handleExternalRotation(ctx context.Context, newIdentity *domain.IdentityDocument) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Validate new identity
	if err := s.validateIdentity(ctx, newIdentity); err != nil {
		return fmt.Errorf("external identity validation failed: %w", err)
	}
	
	oldIdentity := s.currentIdentity
	
	// Ensure it's actually newer
	if oldIdentity != nil && !s.isNewerIdentity(oldIdentity, newIdentity) {
		return fmt.Errorf("external identity is not newer than current")
	}
	
	// Update current identity
	s.currentIdentity = newIdentity
	s.lastRotation = time.Now()
	
	s.logger.Info("identity updated from external source",
		"new_expiry", newIdentity.ValidUntil())
	
	// Notify callbacks
	go s.notifyRotationCallbacks(oldIdentity, newIdentity)
	
	return nil
}

// validateIdentity validates an identity document against trust bundle.
func (s *IdentityRotationService) validateIdentity(ctx context.Context, identity *domain.IdentityDocument) error {
	// Basic validation
	if err := identity.Validate(); err != nil {
		return fmt.Errorf("identity validation failed: %w", err)
	}
	
	// Get trust bundle for validation
	trustBundle, err := s.bundleProvider.GetTrustBundle(ctx)
	if err != nil {
		return fmt.Errorf("failed to get trust bundle: %w", err)
	}
	
	// Validate against trust bundle
	if err := identity.ValidateAgainstBundle(trustBundle); err != nil {
		return fmt.Errorf("identity not trusted by bundle: %w", err)
	}
	
	return nil
}

// isNewerIdentity checks if new identity is actually newer than old.
func (s *IdentityRotationService) isNewerIdentity(old, new *domain.IdentityDocument) bool {
	// Check if issuance time is newer
	if new.IssuedAt().After(old.IssuedAt()) {
		return true
	}
	
	// Check if expiry is later (for same issuance time)
	if new.IssuedAt().Equal(old.IssuedAt()) && new.ValidUntil().After(old.ValidUntil()) {
		return true
	}
	
	return false
}

// calculateRotationJitter calculates a random jitter for rotation timing.
func (s *IdentityRotationService) calculateRotationJitter() time.Duration {
	// Simple jitter calculation - can be made more sophisticated
	// Returns a random duration between 0 and maxRotationDelay
	return time.Duration(time.Now().UnixNano()%int64(s.maxRotationDelay))
}

// notifyRotationCallbacks notifies all registered callbacks of rotation.
func (s *IdentityRotationService) notifyRotationCallbacks(oldIdentity, newIdentity *domain.IdentityDocument) {
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
			cb(oldIdentity, newIdentity)
		}(callback)
	}
}

// GetRotationMetrics returns metrics about rotation operations.
func (s *IdentityRotationService) GetRotationMetrics() *RotationMetrics {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	var timeUntilExpiry time.Duration
	if s.currentIdentity != nil {
		timeUntilExpiry = s.currentIdentity.TimeUntilExpiry()
	}
	
	return &RotationMetrics{
		LastRotation:      s.lastRotation,
		TimeUntilExpiry:   timeUntilExpiry,
		IsRunning:         s.isRunning,
		CallbackCount:     len(s.rotationCallbacks),
	}
}

// RotationMetrics contains metrics about rotation operations.
type RotationMetrics struct {
	LastRotation    time.Time
	TimeUntilExpiry time.Duration
	IsRunning       bool
	CallbackCount   int
}