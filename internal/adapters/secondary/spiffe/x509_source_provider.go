// Package spiffe provides shared utilities for SPIFFE Workload API client management.
package spiffe

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/spiffe/go-spiffe/v2/workloadapi"

	"github.com/sufield/ephemos/internal/core/domain"
)

// X509SourceProvider provides shared X509Source creation and lifecycle management.
// This simplifies Workload API client access by reducing duplication
// and providing consistent source creation patterns across adapters.
type X509SourceProvider struct {
	mu         sync.RWMutex
	socketPath domain.SocketPath
	logger     *slog.Logger
	source     *workloadapi.X509Source
}

// NewX509SourceProvider creates a new X509 source provider for the given socket path.
func NewX509SourceProvider(socketPath domain.SocketPath, logger *slog.Logger) *X509SourceProvider {
	if logger == nil {
		logger = slog.Default()
	}
	
	return &X509SourceProvider{
		socketPath: socketPath,
		logger:     logger,
	}
}

// GetOrCreateSource ensures an X509Source is created and returns it.
// This method is thread-safe and will create the source only once.
func (sp *X509SourceProvider) GetOrCreateSource(ctx context.Context) (*workloadapi.X509Source, error) {
	// Check if source already exists
	sp.mu.RLock()
	if sp.source != nil {
		source := sp.source
		sp.mu.RUnlock()
		return source, nil
	}
	sp.mu.RUnlock()

	// Create source with write lock
	sp.mu.Lock()
	defer sp.mu.Unlock()

	// Double-check pattern (source might have been created while waiting for lock)
	if sp.source != nil {
		return sp.source, nil
	}

	// Require explicit socket path configuration - no fallback patterns
	if sp.socketPath.IsEmpty() {
		return nil, fmt.Errorf("SPIFFE socket path must be explicitly configured - no fallback patterns allowed")
	}
	actualSocketPath := sp.socketPath.WithUnixPrefix()

	sp.logger.Debug("creating X509 source", "socket_path", actualSocketPath)

	source, err := workloadapi.NewX509Source(
		ctx,
		workloadapi.WithClientOptions(
			workloadapi.WithAddr(actualSocketPath),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create X509 source: %w", err)
	}

	sp.source = source
	sp.logger.Info("X509 source created successfully")
	return source, nil
}

// Close closes the managed X509Source if it exists.
func (sp *X509SourceProvider) Close() error {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	if sp.source != nil {
		if err := sp.source.Close(); err != nil {
			return fmt.Errorf("failed to close X509 source: %w", err)
		}
		sp.source = nil
	}
	
	return nil
}

// IsInitialized returns true if the source has been created.
func (sp *X509SourceProvider) IsInitialized() bool {
	sp.mu.RLock()
	defer sp.mu.RUnlock()
	return sp.source != nil
}