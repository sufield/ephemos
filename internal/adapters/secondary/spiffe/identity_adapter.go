// Package spiffe provides SPIFFE identity management adapters.
package spiffe

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/spiffe/go-spiffe/v2/svid/x509svid"
	"github.com/spiffe/go-spiffe/v2/workloadapi"

	"github.com/sufield/ephemos/internal/core/domain"
	"github.com/sufield/ephemos/internal/core/ports"
)

// IdentityDocumentAdapter adapts SPIFFE workload API to IdentityProviderPort.
// This adapter handles SVID fetching and identity document creation from SPIFFE sources.
type IdentityDocumentAdapter struct {
	socketPath domain.SocketPath
	x509Source *workloadapi.X509Source
	logger     *slog.Logger

	// State management
	mu          sync.RWMutex
	currentSVID *x509svid.SVID
	watchCancel context.CancelFunc
	watcherChan chan *x509svid.SVID
}

// IdentityDocumentAdapterConfig provides configuration for the adapter.
type IdentityDocumentAdapterConfig struct {
	SocketPath domain.SocketPath
	Logger     *slog.Logger
}

// NewIdentityDocumentAdapter creates a new SPIFFE identity document adapter.
func NewIdentityDocumentAdapter(config IdentityDocumentAdapterConfig) (*IdentityDocumentAdapter, error) {
	socketPath := config.SocketPath
	// Note: Empty socket paths will cause errors in connection logic (fail-fast behavior)
	// The actual connection logic will handle defaults when needed

	logger := config.Logger
	if logger == nil {
		logger = slog.Default()
	}

	return &IdentityDocumentAdapter{
		socketPath:  socketPath,
		logger:      logger,
		watcherChan: make(chan *x509svid.SVID, 10), // Buffer for updates
	}, nil
}

// GetServiceIdentity retrieves the current service identity from SPIFFE.
func (a *IdentityDocumentAdapter) GetServiceIdentity(ctx context.Context) (*domain.ServiceIdentity, error) {
	a.logger.Debug("fetching service identity from SPIFFE")

	if err := a.ensureSource(ctx); err != nil {
		return nil, fmt.Errorf("failed to ensure X509 source: %w", err)
	}

	svid, err := a.x509Source.GetX509SVID()
	if err != nil {
		return nil, fmt.Errorf("failed to get X509 SVID: %w", err)
	}

	// Store current SVID
	a.mu.Lock()
	a.currentSVID = svid
	a.mu.Unlock()

	// Extract service name from SPIFFE ID path
	serviceName := a.extractServiceName(svid.ID.Path())
	trustDomain := svid.ID.TrustDomain().String()

	a.logger.Debug("service identity retrieved",
		"service", serviceName,
		"trust_domain", trustDomain,
		"spiffe_id", svid.ID.String())

	identity, err := domain.NewServiceIdentityValidated(serviceName, trustDomain)
	if err != nil {
		return nil, fmt.Errorf("failed to create service identity: %w", err)
	}
	return identity, nil
}

// GetCertificate retrieves the current certificate from SPIFFE.
func (a *IdentityDocumentAdapter) GetCertificate(ctx context.Context) (*domain.Certificate, error) {
	a.logger.Debug("fetching certificate from SPIFFE")

	if err := a.ensureSource(ctx); err != nil {
		return nil, fmt.Errorf("failed to ensure X509 source: %w", err)
	}

	svid, err := a.x509Source.GetX509SVID()
	if err != nil {
		return nil, fmt.Errorf("failed to get X509 SVID: %w", err)
	}

	// Store current SVID
	a.mu.Lock()
	a.currentSVID = svid
	a.mu.Unlock()

	if len(svid.Certificates) == 0 {
		return nil, fmt.Errorf("SVID has no certificates")
	}

	// Create domain certificate from SVID
	cert, err := domain.NewCertificate(
		svid.Certificates[0],
		svid.PrivateKey,
		svid.Certificates[1:], // Intermediate certificates
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create domain certificate: %w", err)
	}

	a.logger.Debug("certificate retrieved",
		"expires_at", svid.Certificates[0].NotAfter,
		"spiffe_id", svid.ID.String())

	return cert, nil
}

// GetSVID retrieves the complete SPIFFE SVID directly from the SDK.
func (a *IdentityDocumentAdapter) GetSVID(ctx context.Context) (*x509svid.SVID, error) {
	a.logger.Debug("fetching SVID from SPIFFE")

	if err := a.ensureSource(ctx); err != nil {
		return nil, fmt.Errorf("failed to ensure X509 source: %w", err)
	}

	svid, err := a.x509Source.GetX509SVID()
	if err != nil {
		return nil, fmt.Errorf("failed to get X509 SVID: %w", err)
	}

	// Store current SVID
	a.mu.Lock()
	a.currentSVID = svid
	a.mu.Unlock()

	a.logger.Debug("SVID retrieved",
		"expires_at", svid.Certificates[0].NotAfter,
		"spiffe_id", svid.ID.String())

	return svid, nil
}

// RefreshIdentity triggers a refresh of the identity credentials.
// Note: With SPIFFE Workload API, SVIDs are automatically refreshed by the agent.
// This method forces a re-fetch from the source.
func (a *IdentityDocumentAdapter) RefreshIdentity(ctx context.Context) error {
	a.logger.Info("refreshing identity from SPIFFE")

	if err := a.ensureSource(ctx); err != nil {
		return fmt.Errorf("failed to ensure X509 source: %w", err)
	}

	// Force a fresh fetch from the workload API
	svid, err := a.x509Source.GetX509SVID()
	if err != nil {
		return fmt.Errorf("failed to refresh X509 SVID: %w", err)
	}

	// Update stored SVID
	a.mu.Lock()
	oldSVID := a.currentSVID
	a.currentSVID = svid
	a.mu.Unlock()

	// Check if SVID actually changed
	if oldSVID != nil && oldSVID.ID.String() == svid.ID.String() {
		if len(oldSVID.Certificates) > 0 && len(svid.Certificates) > 0 {
			if oldSVID.Certificates[0].SerialNumber.Cmp(svid.Certificates[0].SerialNumber) == 0 {
				a.logger.Debug("SVID has not changed after refresh")
			}
		}
	}

	a.logger.Info("identity refreshed successfully",
		"spiffe_id", svid.ID.String(),
		"expires_at", svid.Certificates[0].NotAfter)

	return nil
}

// WatchIdentityChanges returns a channel that receives SVID updates.
// This uses SPIFFE Workload API's streaming updates for automatic rotation.
func (a *IdentityDocumentAdapter) WatchIdentityChanges(ctx context.Context) (<-chan *x509svid.SVID, error) {
	a.logger.Info("starting identity change watcher")

	if err := a.ensureSource(ctx); err != nil {
		return nil, fmt.Errorf("failed to ensure X509 source: %w", err)
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
	go a.watchForUpdates(watchCtx)

	return a.watcherChan, nil
}

// watchForUpdates monitors for SVID updates from SPIFFE.
func (a *IdentityDocumentAdapter) watchForUpdates(ctx context.Context) {
	defer close(a.watcherChan)

	// Create update channel from workload API
	updateCh := a.x509Source.Updated()

	for {
		select {
		case <-ctx.Done():
			a.logger.Debug("identity watcher stopped")
			return

		case <-updateCh:
			a.logger.Info("SVID update detected")

			// Get updated SVID
			svid, err := a.x509Source.GetX509SVID()
			if err != nil {
				a.logger.Error("failed to get updated SVID", "error", err)
				continue
			}

			// Update stored SVID
			a.mu.Lock()
			a.currentSVID = svid
			a.mu.Unlock()

			// Send SVID update to channel (non-blocking)
			select {
			case a.watcherChan <- svid:
				a.logger.Info("SVID update sent",
					"expires_at", svid.Certificates[0].NotAfter,
					"spiffe_id", svid.ID.String())
			default:
				a.logger.Warn("SVID update channel full, dropping update")
			}
		}
	}
}

// Close releases resources held by the adapter.
func (a *IdentityDocumentAdapter) Close() error {
	a.logger.Debug("closing SPIFFE identity adapter")

	// Cancel watcher if running
	a.mu.Lock()
	if a.watchCancel != nil {
		a.watchCancel()
		a.watchCancel = nil
	}
	a.mu.Unlock()

	// Close X509 source
	if a.x509Source != nil {
		if err := a.x509Source.Close(); err != nil {
			return fmt.Errorf("failed to close X509 source: %w", err)
		}
	}

	a.logger.Debug("SPIFFE identity adapter closed")
	return nil
}

// ensureSource ensures the X509 source is initialized.
func (a *IdentityDocumentAdapter) ensureSource(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.x509Source != nil {
		return nil
	}

	// Require explicit socket path configuration - no fallback patterns
	if a.socketPath.IsEmpty() {
		return fmt.Errorf("SPIFFE socket path must be explicitly configured - no fallback patterns allowed")
	}
	actualSocketPath := a.socketPath.WithUnixPrefix()

	a.logger.Debug("initializing X509 source", "socket_path", actualSocketPath)

	source, err := workloadapi.NewX509Source(
		ctx,
		workloadapi.WithClientOptions(
			workloadapi.WithAddr(actualSocketPath),
		),
	)
	if err != nil {
		return fmt.Errorf("failed to create X509 source: %w", err)
	}

	a.x509Source = source
	a.logger.Info("X509 source initialized successfully")
	return nil
}

// extractServiceName extracts the service name from a SPIFFE ID path.
func (a *IdentityDocumentAdapter) extractServiceName(path string) string {
	if path == "" || path == "/" {
		return "unknown"
	}

	// Remove leading slash and take first segment
	path = strings.TrimPrefix(path, "/")
	segments := strings.Split(path, "/")
	if len(segments) > 0 && segments[0] != "" {
		return segments[0]
	}

	return "unknown"
}

// Ensure adapter implements the port interface
var _ ports.IdentityProviderPort = (*IdentityDocumentAdapter)(nil)
