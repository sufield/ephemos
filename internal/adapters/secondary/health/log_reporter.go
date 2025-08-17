// Package health provides health reporting implementations
package health

import (
	"log/slog"

	"github.com/sufield/ephemos/internal/core/ports"
)

// LogHealthReporter implements health reporting via structured logging
type LogHealthReporter struct {
	logger *slog.Logger
}

// NewLogHealthReporter creates a new logging health reporter
func NewLogHealthReporter(logger *slog.Logger) *LogHealthReporter {
	if logger == nil {
		logger = slog.Default()
	}

	return &LogHealthReporter{
		logger: logger,
	}
}

// ReportHealth reports a health check result via logging
func (r *LogHealthReporter) ReportHealth(result *ports.HealthResult) error {
	if result == nil {
		return nil
	}

	attrs := []slog.Attr{
		slog.String("component", result.Component),
		slog.String("status", string(result.Status)),
		slog.Duration("response_time", result.ResponseTime),
		slog.Time("checked_at", result.CheckedAt),
	}

	if result.Message != "" {
		attrs = append(attrs, slog.String("message", result.Message))
	}

	switch result.Status {
	case ports.HealthStatusHealthy:
		r.logger.LogAttrs(nil, slog.LevelInfo, "Health check passed", attrs...)
	case ports.HealthStatusUnhealthy:
		r.logger.LogAttrs(nil, slog.LevelWarn, "Health check failed", attrs...)
	case ports.HealthStatusUnknown:
		r.logger.LogAttrs(nil, slog.LevelError, "Health check status unknown", attrs...)
	default:
		r.logger.LogAttrs(nil, slog.LevelWarn, "Health check completed", attrs...)
	}

	return nil
}

// ReportOverallHealth reports the overall system health
func (r *LogHealthReporter) ReportOverallHealth(results map[string]*ports.HealthResult) error {
	if len(results) == 0 {
		r.logger.Info("No health check results available")
		return nil
	}

	healthyCount := 0
	unhealthyCount := 0
	unknownCount := 0

	for _, result := range results {
		switch result.Status {
		case ports.HealthStatusHealthy:
			healthyCount++
		case ports.HealthStatusUnhealthy:
			unhealthyCount++
		case ports.HealthStatusUnknown:
			unknownCount++
		}
	}

	totalCount := len(results)
	overallStatus := ports.HealthStatusHealthy
	if unhealthyCount > 0 || unknownCount > 0 {
		overallStatus = ports.HealthStatusUnhealthy
	}

	attrs := []slog.Attr{
		slog.String("overall_status", string(overallStatus)),
		slog.Int("total_components", totalCount),
		slog.Int("healthy", healthyCount),
		slog.Int("unhealthy", unhealthyCount),
		slog.Int("unknown", unknownCount),
	}

	level := slog.LevelInfo
	message := "Overall system health status"

	if overallStatus != ports.HealthStatusHealthy {
		level = slog.LevelWarn
		message = "System health degraded"
	}

	r.logger.LogAttrs(nil, level, message, attrs...)

	return nil
}

// Close cleans up the reporter (no-op for logging reporter)
func (r *LogHealthReporter) Close() error {
	return nil
}