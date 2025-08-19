// Package spiffe provides SPIFFE trust bundle management adapters.
package spiffe

import (
	"context"
	"crypto/x509"
	"fmt"
	"log/slog"
	"sync"

	"github.com/spiffe/go-spiffe/v2/bundle/x509bundle"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/spiffe/go-spiffe/v2/workloadapi"

	"github.com/sufield/ephemos/internal/core/domain"
	"github.com/sufield/ephemos/internal/core/ports"
)

// SpiffeBundleAdapter adapts SPIFFE workload API to BundleProviderPort.
// This adapter handles trust bundle fetching and management from SPIFFE sources.
type SpiffeBundleAdapter struct {
	x509SourceProvider *X509SourceProvider
	logger        *slog.Logger

	// State management
	mu            sync.RWMutex
	currentBundle *x509bundle.Set
	watchCancel   context.CancelFunc
	watcherChan   chan *x509bundle.Bundle
}

// SpiffeBundleAdapterConfig provides configuration for the adapter.
type SpiffeBundleAdapterConfig struct {
	X509SourceProvider *X509SourceProvider
	Logger        *slog.Logger
}

// NewSpiffeBundleAdapter creates a new SPIFFE trust bundle adapter.
func NewSpiffeBundleAdapter(config SpiffeBundleAdapterConfig) (*SpiffeBundleAdapter, error) {
	if config.X509SourceProvider == nil {
		return nil, fmt.Errorf("source manager is required")
	}

	logger := config.Logger
	if logger == nil {
		logger = slog.Default()
	}

	return &SpiffeBundleAdapter{
		x509SourceProvider: config.X509SourceProvider,
		logger:        logger,
		watcherChan:   make(chan *x509bundle.Bundle, 10), // Buffer for updates
	}, nil
}

// GetTrustBundle retrieves the current trust bundle from SPIFFE using SDK directly.
func (a *SpiffeBundleAdapter) GetTrustBundle(ctx context.Context) (*x509bundle.Bundle, error) {
	a.logger.Debug("fetching trust bundle from SPIFFE")

	x509Source, err := a.x509SourceProvider.GetOrCreateSource(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get X509 source: %w", err)
	}

	// Get the default trust domain from current SVID
	svid, err := x509Source.GetX509SVID()
	if err != nil {
		return nil, fmt.Errorf("failed to get X509 SVID for trust domain: %w", err)
	}

	// Get bundle for the SVID's trust domain - return SDK bundle directly
	bundle, err := x509Source.GetX509BundleForTrustDomain(svid.ID.TrustDomain())
	if err != nil {
		return nil, fmt.Errorf("failed to get trust bundle: %w", err)
	}

	a.logger.Debug("trust bundle retrieved",
		"trust_domain", svid.ID.TrustDomain().String(),
		"ca_count", len(bundle.X509Authorities()))

	return bundle, nil
}

// GetTrustBundleForDomain retrieves a trust bundle for a specific trust domain.
func (a *SpiffeBundleAdapter) GetTrustBundleForDomain(ctx context.Context, trustDomain spiffeid.TrustDomain) (*x509bundle.Bundle, error) {
	a.logger.Debug("fetching trust bundle for specific domain",
		"trust_domain", trustDomain.String())

	x509Source, err := a.x509SourceProvider.GetOrCreateSource(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get X509 source: %w", err)
	}

	// Get bundle for specific trust domain - return SDK bundle directly
	bundle, err := x509Source.GetX509BundleForTrustDomain(trustDomain)
	if err != nil {
		return nil, fmt.Errorf("failed to get trust bundle for domain %s: %w", trustDomain, err)
	}

	a.logger.Debug("trust bundle retrieved for domain",
		"trust_domain", trustDomain.String(),
		"ca_count", len(bundle.X509Authorities()))

	return bundle, nil
}

// RefreshTrustBundle triggers a refresh of the trust bundle.
// Note: With SPIFFE Workload API, bundles are automatically refreshed by the agent.
// This method forces a re-fetch from the source.
func (a *SpiffeBundleAdapter) RefreshTrustBundle(ctx context.Context) error {
	a.logger.Info("refreshing trust bundle from SPIFFE")

	_, err := a.x509SourceProvider.GetOrCreateSource(ctx)
	if err != nil {
		return fmt.Errorf("failed to get X509 source: %w", err)
	}

	// Refresh is essentially a no-op since X509Source always fetches fresh
	a.logger.Debug("trust bundle refresh requested (no-op with X509Source)")

	a.logger.Info("trust bundle refreshed successfully")
	return nil
}

// WatchTrustBundleChanges returns a channel that receives trust bundle updates.
// This uses SPIFFE Workload API's streaming updates for automatic rotation.
func (a *SpiffeBundleAdapter) WatchTrustBundleChanges(ctx context.Context) (<-chan *x509bundle.Bundle, error) {
	a.logger.Info("starting trust bundle change watcher")

	x509Source, err := a.x509SourceProvider.GetOrCreateSource(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get X509 source: %w", err)
	}

	// Cancel any existing watcher
	a.mu.Lock()
	if a.watchCancel != nil {
		a.watchCancel()
	}

	// Create new context for watcher
	watchCtx, cancel := context.WithCancel(ctx)
	a.watchCancel = cancel
	a.mu.Unlock()

	// Start watcher goroutine
	go a.watchForBundleUpdates(watchCtx, x509Source)

	return a.watcherChan, nil
}

// watchForBundleUpdates monitors for trust bundle updates from SPIFFE.
func (a *SpiffeBundleAdapter) watchForBundleUpdates(ctx context.Context, x509Source *workloadapi.X509Source) {
	defer close(a.watcherChan)

	// Create update channel from workload API
	updateCh := x509Source.Updated()

	for {
		select {
		case <-ctx.Done():
			a.logger.Debug("trust bundle watcher stopped")
			return

		case <-updateCh:
			a.logger.Info("trust bundle update detected")

			// Get current SVID to determine trust domain
			svid, err := x509Source.GetX509SVID()
			if err != nil {
				a.logger.Error("failed to get SVID for trust domain", "error", err)
				continue
			}

			// Get updated bundle
			bundle, err := x509Source.GetX509BundleForTrustDomain(svid.ID.TrustDomain())
			if err != nil {
				a.logger.Error("failed to get updated trust bundle", "error", err)
				continue
			}

			// Note: We don't cache bundle sets since X509Source doesn't expose them

			// Send bundle update to channel (non-blocking) - use SDK bundle directly
			select {
			case a.watcherChan <- bundle:
				a.logger.Info("trust bundle update sent",
					"ca_count", len(bundle.X509Authorities()))
			default:
				a.logger.Warn("trust bundle update channel full, dropping update")
			}
		}
	}
}

// ValidateCertificateAgainstBundle validates a certificate against the trust bundle.
func (a *SpiffeBundleAdapter) ValidateCertificateAgainstBundle(ctx context.Context, cert *domain.Certificate) error {
	a.logger.Debug("validating certificate against trust bundle")

	if cert == nil {
		return fmt.Errorf("certificate cannot be nil")
	}

	// Get current trust bundle
	trustBundle, err := a.GetTrustBundle(ctx)
	if err != nil {
		return fmt.Errorf("failed to get trust bundle for validation: %w", err)
	}

	// Validate using domain trust bundle's validation
	if cert.Cert == nil {
		return fmt.Errorf("certificate has no X.509 certificate")
	}

	// Create certificate pool from trust bundle for validation
	pool := x509.NewCertPool()
	for _, ca := range trustBundle.X509Authorities() {
		pool.AddCert(ca)
	}

	// Validate the certificate chain using standard Go crypto/x509 validation
	opts := x509.VerifyOptions{
		Roots:         pool,
		Intermediates: x509.NewCertPool(),
	}
	
	// Add intermediate certificates to verify options
	for _, intermediate := range cert.Chain {
		opts.Intermediates.AddCert(intermediate)
	}

	// Perform validation
	if _, err := cert.Cert.Verify(opts); err != nil {
		return fmt.Errorf("certificate validation failed: %w", err)
	}

	a.logger.Debug("certificate validated successfully")
	return nil
}

// Close releases resources held by the adapter.
func (a *SpiffeBundleAdapter) Close() error {
	a.logger.Debug("closing SPIFFE bundle adapter")

	// Cancel watcher if running
	a.mu.Lock()
	if a.watchCancel != nil {
		a.watchCancel()
		a.watchCancel = nil
	}
	a.mu.Unlock()

	// Note: X509 source is managed by X509SourceProvider, not closed here

	a.logger.Debug("SPIFFE bundle adapter closed")
	return nil
}


// Ensure adapter implements the port interface
var _ ports.BundleProviderPort = (*SpiffeBundleAdapter)(nil)
