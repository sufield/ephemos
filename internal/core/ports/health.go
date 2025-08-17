// Package ports defines the health check interfaces for the Ephemos library.
// These interfaces follow the hexagonal architecture pattern and enable
// integration with SPIRE's built-in health endpoints.
package ports

import (
	"context"
	"time"
)

// HealthStatus represents the health status of a component
type HealthStatus string

const (
	// HealthStatusHealthy indicates the component is healthy and ready
	HealthStatusHealthy HealthStatus = "healthy"
	// HealthStatusUnhealthy indicates the component is not healthy
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	// HealthStatusUnknown indicates the health status cannot be determined
	HealthStatusUnknown HealthStatus = "unknown"
)

// HealthResult contains the result of a health check
type HealthResult struct {
	// Status is the overall health status
	Status HealthStatus `json:"status"`
	// Component is the name of the component being checked
	Component string `json:"component"`
	// Message provides additional details about the health status
	Message string `json:"message,omitempty"`
	// CheckedAt is when the health check was performed
	CheckedAt time.Time `json:"checked_at"`
	// ResponseTime is how long the health check took
	ResponseTime time.Duration `json:"response_time"`
	// Details contains component-specific health information
	Details map[string]interface{} `json:"details,omitempty"`
}

// HealthConfig configures health check behavior
type HealthConfig struct {
	// Enabled determines if health checks are active
	Enabled bool `json:"enabled"`
	// Timeout for individual health checks
	Timeout time.Duration `json:"timeout"`
	// Interval between periodic health checks
	Interval time.Duration `json:"interval"`
	// Server configuration for SPIRE server health checks
	Server *SpireServerHealthConfig `json:"server,omitempty"`
	// Agent configuration for SPIRE agent health checks
	Agent *SpireAgentHealthConfig `json:"agent,omitempty"`
}

// SpireServerHealthConfig configures SPIRE server health monitoring
type SpireServerHealthConfig struct {
	// Address of the SPIRE server health endpoint (e.g., "localhost:8080")
	Address string `json:"address"`
	// LivePath is the liveness check endpoint path (default: "/live")
	LivePath string `json:"live_path"`
	// ReadyPath is the readiness check endpoint path (default: "/ready")
	ReadyPath string `json:"ready_path"`
	// UseHTTPS enables HTTPS for health check requests
	UseHTTPS bool `json:"use_https"`
	// Headers are additional HTTP headers to include in health check requests
	Headers map[string]string `json:"headers,omitempty"`
}

// SpireAgentHealthConfig configures SPIRE agent health monitoring
type SpireAgentHealthConfig struct {
	// Address of the SPIRE agent health endpoint (e.g., "localhost:8080")
	Address string `json:"address"`
	// LivePath is the liveness check endpoint path (default: "/live")
	LivePath string `json:"live_path"`
	// ReadyPath is the readiness check endpoint path (default: "/ready")
	ReadyPath string `json:"ready_path"`
	// UseHTTPS enables HTTPS for health check requests
	UseHTTPS bool `json:"use_https"`
	// Headers are additional HTTP headers to include in health check requests
	Headers map[string]string `json:"headers,omitempty"`
}

// HealthCheckerPort defines the interface for performing health checks
type HealthCheckerPort interface {
	// CheckLiveness verifies if the component is alive/running
	CheckLiveness(ctx context.Context) (*HealthResult, error)
	// CheckReadiness verifies if the component is ready to handle requests
	CheckReadiness(ctx context.Context) (*HealthResult, error)
	// CheckHealth performs a comprehensive health check
	CheckHealth(ctx context.Context) (*HealthResult, error)
	// GetComponentName returns the name of the component being monitored
	GetComponentName() string
}

// HealthMonitorPort defines the interface for monitoring multiple components
type HealthMonitorPort interface {
	// RegisterChecker adds a health checker for monitoring
	RegisterChecker(checker HealthCheckerPort) error
	// UnregisterChecker removes a health checker from monitoring
	UnregisterChecker(componentName string) error
	// CheckAll performs health checks on all registered components
	CheckAll(ctx context.Context) (map[string]*HealthResult, error)
	// StartMonitoring begins periodic health monitoring
	StartMonitoring(ctx context.Context) error
	// StopMonitoring stops periodic health monitoring
	StopMonitoring() error
	// GetResults returns the latest health check results
	GetResults() map[string]*HealthResult
}

// HealthReporterPort defines the interface for reporting health status
type HealthReporterPort interface {
	// ReportHealth reports a health check result
	ReportHealth(result *HealthResult) error
	// ReportOverallHealth reports the overall system health
	ReportOverallHealth(results map[string]*HealthResult) error
	// Close cleans up reporter resources
	Close() error
}

// SpireHealthClientPort defines the interface for SPIRE-specific health checking
type SpireHealthClientPort interface {
	HealthCheckerPort
	// CheckServerHealth checks SPIRE server health via HTTP endpoints
	CheckServerHealth(ctx context.Context) (*HealthResult, error)
	// CheckAgentHealth checks SPIRE agent health via HTTP endpoints
	CheckAgentHealth(ctx context.Context) (*HealthResult, error)
	// UpdateConfig updates the health check configuration
	UpdateConfig(config *HealthConfig) error
}