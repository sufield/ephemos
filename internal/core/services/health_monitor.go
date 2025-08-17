// Package services contains the core business logic for health monitoring.
// This service coordinates health checks across multiple SPIRE components
// and provides aggregated health status reporting.
package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/sufield/ephemos/internal/core/ports"
)

// HealthMonitorService implements comprehensive health monitoring for SPIRE infrastructure
type HealthMonitorService struct {
	config     *ports.HealthConfig
	checkers   map[string]ports.HealthCheckerPort
	results    map[string]*ports.HealthResult
	reporters  []ports.HealthReporterPort
	mu         sync.RWMutex
	stopCh     chan struct{}
	monitoring bool
	logger     *slog.Logger
}

// NewHealthMonitorService creates a new health monitoring service
func NewHealthMonitorService(config *ports.HealthConfig, logger *slog.Logger) (*HealthMonitorService, error) {
	if config == nil {
		return nil, fmt.Errorf("health config cannot be nil")
	}

	if logger == nil {
		logger = slog.Default()
	}

	return &HealthMonitorService{
		config:    config,
		checkers:  make(map[string]ports.HealthCheckerPort),
		results:   make(map[string]*ports.HealthResult),
		reporters: make([]ports.HealthReporterPort, 0),
		stopCh:    make(chan struct{}),
		logger:    logger,
	}, nil
}

// RegisterChecker adds a health checker for monitoring
func (h *HealthMonitorService) RegisterChecker(checker ports.HealthCheckerPort) error {
	if checker == nil {
		return fmt.Errorf("health checker cannot be nil")
	}

	componentName := checker.GetComponentName()
	if componentName == "" {
		return fmt.Errorf("health checker must have a valid component name")
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	h.checkers[componentName] = checker
	h.logger.Info("Health checker registered", "component", componentName)

	return nil
}

// UnregisterChecker removes a health checker from monitoring
func (h *HealthMonitorService) UnregisterChecker(componentName string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, exists := h.checkers[componentName]; !exists {
		return fmt.Errorf("health checker for component %s not found", componentName)
	}

	delete(h.checkers, componentName)
	delete(h.results, componentName)
	h.logger.Info("Health checker unregistered", "component", componentName)

	return nil
}

// RegisterReporter adds a health status reporter
func (h *HealthMonitorService) RegisterReporter(reporter ports.HealthReporterPort) error {
	if reporter == nil {
		return fmt.Errorf("health reporter cannot be nil")
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	h.reporters = append(h.reporters, reporter)
	h.logger.Info("Health reporter registered")

	return nil
}

// CheckAll performs health checks on all registered components
func (h *HealthMonitorService) CheckAll(ctx context.Context) (map[string]*ports.HealthResult, error) {
	h.mu.RLock()
	checkers := make(map[string]ports.HealthCheckerPort)
	for name, checker := range h.checkers {
		checkers[name] = checker
	}
	h.mu.RUnlock()

	if len(checkers) == 0 {
		h.logger.Warn("No health checkers registered")
		return make(map[string]*ports.HealthResult), nil
	}

	// Perform checks concurrently for better performance
	results := make(map[string]*ports.HealthResult)
	resultsCh := make(chan struct {
		name   string
		result *ports.HealthResult
		err    error
	}, len(checkers))

	// Start all health checks concurrently
	for name, checker := range checkers {
		go func(name string, checker ports.HealthCheckerPort) {
			result, err := checker.CheckHealth(ctx)
			if err != nil {
				// Create error result
				result = &ports.HealthResult{
					Status:       ports.HealthStatusUnknown,
					Component:    name,
					Message:      fmt.Sprintf("Health check failed: %v", err),
					CheckedAt:    time.Now(),
					ResponseTime: 0,
					Details: map[string]interface{}{
						"error": err.Error(),
					},
				}
			}
			resultsCh <- struct {
				name   string
				result *ports.HealthResult
				err    error
			}{name, result, err}
		}(name, checker)
	}

	// Collect results
	for i := 0; i < len(checkers); i++ {
		select {
		case result := <-resultsCh:
			results[result.name] = result.result
			if result.err != nil {
				h.logger.Error("Health check failed",
					"component", result.name,
					"error", result.err)
			}
		case <-ctx.Done():
			return results, ctx.Err()
		}
	}

	// Update stored results
	h.mu.Lock()
	for name, result := range results {
		h.results[name] = result
	}
	h.mu.Unlock()

	// Report results to all registered reporters
	h.reportToAll(results)

	return results, nil
}

// StartMonitoring begins periodic health monitoring
func (h *HealthMonitorService) StartMonitoring(ctx context.Context) error {
	if !h.config.Enabled {
		h.logger.Info("Health monitoring is disabled")
		return nil
	}

	h.mu.Lock()
	if h.monitoring {
		h.mu.Unlock()
		return fmt.Errorf("health monitoring is already running")
	}
	h.monitoring = true
	h.mu.Unlock()

	interval := h.config.Interval
	if interval == 0 {
		interval = 30 * time.Second // Default interval
	}

	h.logger.Info("Starting health monitoring",
		"interval", interval,
		"checkers", len(h.checkers))

	go h.monitoringLoop(ctx, interval)

	return nil
}

// StopMonitoring stops periodic health monitoring
func (h *HealthMonitorService) StopMonitoring() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if !h.monitoring {
		return fmt.Errorf("health monitoring is not running")
	}

	close(h.stopCh)
	h.monitoring = false
	h.stopCh = make(chan struct{}) // Reset for future use

	h.logger.Info("Health monitoring stopped")

	return nil
}

// GetResults returns the latest health check results
func (h *HealthMonitorService) GetResults() map[string]*ports.HealthResult {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Return a copy to prevent concurrent modification
	results := make(map[string]*ports.HealthResult)
	for name, result := range h.results {
		results[name] = result
	}

	return results
}

// GetOverallHealth returns the overall system health status
func (h *HealthMonitorService) GetOverallHealth() ports.HealthStatus {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if len(h.results) == 0 {
		return ports.HealthStatusUnknown
	}

	healthyCount := 0
	for _, result := range h.results {
		if result.Status == ports.HealthStatusHealthy {
			healthyCount++
		}
	}

	// All components must be healthy for overall health to be healthy
	if healthyCount == len(h.results) {
		return ports.HealthStatusHealthy
	}

	return ports.HealthStatusUnhealthy
}

// Close shuts down the health monitoring service and cleans up resources
func (h *HealthMonitorService) Close() error {
	// Stop monitoring if it's running
	if h.monitoring {
		if err := h.StopMonitoring(); err != nil {
			h.logger.Error("Failed to stop monitoring during close", "error", err)
		}
	}

	// Close all reporters
	h.mu.Lock()
	var errs []error
	for _, reporter := range h.reporters {
		if err := reporter.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	h.reporters = nil
	h.mu.Unlock()

	if len(errs) > 0 {
		return fmt.Errorf("failed to close some reporters: %w", errors.Join(errs...))
	}

	h.logger.Info("Health monitoring service closed")
	return nil
}

// monitoringLoop runs the periodic health checking
func (h *HealthMonitorService) monitoringLoop(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			checkCtx, cancel := context.WithTimeout(ctx, h.getCheckTimeout())
			_, err := h.CheckAll(checkCtx)
			cancel()

			if err != nil {
				h.logger.Error("Periodic health check failed", "error", err)
			}

		case <-h.stopCh:
			h.logger.Debug("Health monitoring loop stopped")
			return

		case <-ctx.Done():
			h.logger.Debug("Health monitoring loop cancelled")
			return
		}
	}
}

// getCheckTimeout returns the timeout for health checks
func (h *HealthMonitorService) getCheckTimeout() time.Duration {
	if h.config.Timeout > 0 {
		return h.config.Timeout
	}
	return 10 * time.Second // Default timeout
}

// reportToAll sends health results to all registered reporters
func (h *HealthMonitorService) reportToAll(results map[string]*ports.HealthResult) {
	h.mu.RLock()
	reporters := make([]ports.HealthReporterPort, len(h.reporters))
	copy(reporters, h.reporters)
	h.mu.RUnlock()

	for _, reporter := range reporters {
		// Report individual results
		for _, result := range results {
			if err := reporter.ReportHealth(result); err != nil {
				h.logger.Error("Failed to report health result",
					"component", result.Component,
					"error", err)
			}
		}

		// Report overall health
		if err := reporter.ReportOverallHealth(results); err != nil {
			h.logger.Error("Failed to report overall health", "error", err)
		}
	}
}
