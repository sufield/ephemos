// Package health provides health checking implementations for SPIRE infrastructure.
// This package leverages SPIRE's built-in health endpoints rather than implementing
// health checks from scratch, following the recommendations in the SPIRE documentation.
package health

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/sufield/ephemos/internal/core/domain"
	"github.com/sufield/ephemos/internal/core/ports"
)

// HealthCapability provides access to health configuration values.
type HealthCapability interface {
	GetAgentAddress() string
	GetAgentLivePath() string
	GetAgentReadyPath() string
	GetAgentUseHTTPS() bool
	GetAgentHeaders() map[string]string
	GetServerAddress() string
	GetServerLivePath() string
	GetServerReadyPath() string
	GetServerUseHTTPS() bool
	GetServerHeaders() map[string]string
	GetTimeout() time.Duration
}

// configHealthCapability implements HealthCapability for ports.HealthConfig
type configHealthCapability struct {
	config *ports.HealthConfig
}

func (c *configHealthCapability) GetAgentAddress() string {
	if c.config.Agent == nil {
		return ""
	}
	return c.config.Agent.Address
}

func (c *configHealthCapability) GetAgentLivePath() string {
	if c.config.Agent == nil {
		return ""
	}
	return c.config.Agent.LivePath
}

func (c *configHealthCapability) GetAgentReadyPath() string {
	if c.config.Agent == nil {
		return ""
	}
	return c.config.Agent.ReadyPath
}

func (c *configHealthCapability) GetAgentUseHTTPS() bool {
	if c.config.Agent == nil {
		return false
	}
	return c.config.Agent.UseHTTPS
}

func (c *configHealthCapability) GetAgentHeaders() map[string]string {
	if c.config.Agent == nil {
		return nil
	}
	return c.config.Agent.Headers
}

func (c *configHealthCapability) GetServerAddress() string {
	if c.config.Server == nil {
		return ""
	}
	return c.config.Server.Address
}

func (c *configHealthCapability) GetServerLivePath() string {
	if c.config.Server == nil {
		return ""
	}
	return c.config.Server.LivePath
}

func (c *configHealthCapability) GetServerReadyPath() string {
	if c.config.Server == nil {
		return ""
	}
	return c.config.Server.ReadyPath
}

func (c *configHealthCapability) GetServerUseHTTPS() bool {
	if c.config.Server == nil {
		return false
	}
	return c.config.Server.UseHTTPS
}

func (c *configHealthCapability) GetServerHeaders() map[string]string {
	if c.config.Server == nil {
		return nil
	}
	return c.config.Server.Headers
}

func (c *configHealthCapability) GetTimeout() time.Duration {
	return c.config.Timeout
}

// SpireHealthClient implements health checking for SPIRE server and agent components
// using their built-in HTTP health endpoints (/live and /ready).
type SpireHealthClient struct {
	capability HealthCapability
	httpClient *http.Client
	component  domain.ComponentType
}

// NewSpireHealthClient creates a new SPIRE health checker client
func NewSpireHealthClient(component string, config *ports.HealthConfig) (*SpireHealthClient, error) {
	capability := &configHealthCapability{config: config}
	return NewSpireHealthClientWithCapability(component, capability)
}

// NewSpireHealthClientWithCapability creates a new SPIRE health checker client with injected capability
func NewSpireHealthClientWithCapability(component string, capability HealthCapability) (*SpireHealthClient, error) {
	if capability == nil {
		return nil, fmt.Errorf("health capability cannot be nil")
	}

	if strings.TrimSpace(component) == "" {
		return nil, fmt.Errorf("component name cannot be empty")
	}

	// Parse component type from string
	componentType, err := domain.ParseComponentType(component)
	if err != nil {
		return nil, fmt.Errorf("invalid component type: %w", err)
	}

	// Create HTTP client with appropriate timeout
	timeout := capability.GetTimeout()
	if timeout == 0 {
		timeout = 10 * time.Second // Default timeout
	}

	httpClient := &http.Client{
		Timeout: timeout,
		// Disable redirects for health checks to avoid false positives
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	return &SpireHealthClient{
		capability: capability,
		httpClient: httpClient,
		component:  componentType,
	}, nil
}

// GetComponentName returns the name of the component being monitored
func (c *SpireHealthClient) GetComponentName() string {
	return c.component.String()
}

// CheckLiveness verifies if the SPIRE component is alive/running
// This uses the /live endpoint which indicates if the process is running
func (c *SpireHealthClient) CheckLiveness(ctx context.Context) (*ports.HealthResult, error) {
	startTime := time.Now()

	var address, livePath string
	var headers map[string]string
	var useHTTPS bool

	// Determine configuration based on component type
	switch c.component {
	case domain.ComponentSpireServer, domain.ComponentServer:
		address = c.capability.GetServerAddress()
		if address == "" {
			return nil, fmt.Errorf("SPIRE server address not configured")
		}
		livePath = c.capability.GetServerLivePath()
		useHTTPS = c.capability.GetServerUseHTTPS()
		headers = c.capability.GetServerHeaders()
	case domain.ComponentSpireAgent, domain.ComponentAgent:
		address = c.capability.GetAgentAddress()
		if address == "" {
			return nil, fmt.Errorf("SPIRE agent address not configured")
		}
		livePath = c.capability.GetAgentLivePath()
		useHTTPS = c.capability.GetAgentUseHTTPS()
		headers = c.capability.GetAgentHeaders()
	default:
		return nil, fmt.Errorf("unsupported component type: %s", c.component.String())
	}

	// Set default path if not configured
	if livePath == "" {
		livePath = "/live"
	}

	// Build URL
	protocol := domain.ProtocolHTTP
	if useHTTPS {
		protocol = domain.ProtocolHTTPS
	}
	url := fmt.Sprintf("%s://%s%s", protocol.String(), address, livePath)

	// Perform health check
	result, err := c.performHealthCheck(ctx, url, "liveness", headers)
	result.ResponseTime = time.Since(startTime)

	return result, err
}

// CheckReadiness verifies if the SPIRE component is ready to handle requests
// This uses the /ready endpoint which indicates if the component can serve requests
func (c *SpireHealthClient) CheckReadiness(ctx context.Context) (*ports.HealthResult, error) {
	startTime := time.Now()

	var address, readyPath string
	var headers map[string]string
	var useHTTPS bool

	// Determine configuration based on component type
	switch c.component {
	case domain.ComponentSpireServer, domain.ComponentServer:
		address = c.capability.GetServerAddress()
		if address == "" {
			return nil, fmt.Errorf("SPIRE server address not configured")
		}
		readyPath = c.capability.GetServerReadyPath()
		useHTTPS = c.capability.GetServerUseHTTPS()
		headers = c.capability.GetServerHeaders()
	case domain.ComponentSpireAgent, domain.ComponentAgent:
		address = c.capability.GetAgentAddress()
		if address == "" {
			return nil, fmt.Errorf("SPIRE agent address not configured")
		}
		readyPath = c.capability.GetAgentReadyPath()
		useHTTPS = c.capability.GetAgentUseHTTPS()
		headers = c.capability.GetAgentHeaders()
	default:
		return nil, fmt.Errorf("unsupported component type: %s", c.component.String())
	}

	// Set default path if not configured
	if readyPath == "" {
		readyPath = "/ready"
	}

	// Build URL
	protocol := domain.ProtocolHTTP
	if useHTTPS {
		protocol = domain.ProtocolHTTPS
	}
	url := fmt.Sprintf("%s://%s%s", protocol.String(), address, readyPath)

	// Perform health check
	result, err := c.performHealthCheck(ctx, url, "readiness", headers)
	result.ResponseTime = time.Since(startTime)

	return result, err
}

// CheckHealth performs a comprehensive health check by checking both liveness and readiness
func (c *SpireHealthClient) CheckHealth(ctx context.Context) (*ports.HealthResult, error) {
	startTime := time.Now()

	// Check liveness first
	livenessResult, livenessErr := c.CheckLiveness(ctx)

	// Check readiness
	readinessResult, readinessErr := c.CheckReadiness(ctx)

	// Combine results
	result := &ports.HealthResult{
		Component:    c.component.String(),
		CheckedAt:    time.Now(),
		ResponseTime: time.Since(startTime),
		Details:      make(map[string]interface{}),
	}

	// Add liveness details
	if livenessErr != nil {
		result.Details["liveness_error"] = livenessErr.Error()
		result.Details["liveness_status"] = "error"
	} else {
		result.Details["liveness_status"] = string(livenessResult.Status)
		result.Details["liveness_response_time"] = livenessResult.ResponseTime.String()
	}

	// Add readiness details
	if readinessErr != nil {
		result.Details["readiness_error"] = readinessErr.Error()
		result.Details["readiness_status"] = "error"
	} else {
		result.Details["readiness_status"] = string(readinessResult.Status)
		result.Details["readiness_response_time"] = readinessResult.ResponseTime.String()
	}

	// Determine overall health status
	if livenessErr != nil || readinessErr != nil {
		result.Status = ports.HealthStatusUnhealthy
		result.Message = "Health check failed"

		var errors []string
		if livenessErr != nil {
			errors = append(errors, fmt.Sprintf("liveness: %v", livenessErr))
		}
		if readinessErr != nil {
			errors = append(errors, fmt.Sprintf("readiness: %v", readinessErr))
		}
		result.Message = fmt.Sprintf("Health check failed: %s", strings.Join(errors, ", "))
	} else if livenessResult.Status == ports.HealthStatusHealthy && readinessResult.Status == ports.HealthStatusHealthy {
		result.Status = ports.HealthStatusHealthy
		result.Message = "Component is healthy and ready"
	} else {
		result.Status = ports.HealthStatusUnhealthy
		result.Message = "Component is not fully healthy"
	}

	return result, nil
}

// CheckServerHealth checks SPIRE server health via HTTP endpoints
func (c *SpireHealthClient) CheckServerHealth(ctx context.Context) (*ports.HealthResult, error) {
	// Temporarily set component to server for this check
	originalComponent := c.component
	c.component = domain.ComponentSpireServer
	defer func() { c.component = originalComponent }()

	return c.CheckHealth(ctx)
}

// CheckAgentHealth checks SPIRE agent health via HTTP endpoints
func (c *SpireHealthClient) CheckAgentHealth(ctx context.Context) (*ports.HealthResult, error) {
	// Temporarily set component to agent for this check
	originalComponent := c.component
	c.component = domain.ComponentSpireAgent
	defer func() { c.component = originalComponent }()

	return c.CheckHealth(ctx)
}

// UpdateConfig updates the health check configuration
func (c *SpireHealthClient) UpdateConfig(config *ports.HealthConfig) error {
	if config == nil {
		return fmt.Errorf("health config cannot be nil")
	}

	c.config = config

	// Update HTTP client timeout if needed
	if config.Timeout > 0 {
		c.httpClient.Timeout = config.Timeout
	}

	return nil
}

// performHealthCheck performs the actual HTTP health check request
func (c *SpireHealthClient) performHealthCheck(ctx context.Context, url, checkType string, headers map[string]string) (*ports.HealthResult, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return &ports.HealthResult{
			Status:    ports.HealthStatusUnknown,
			Component: c.component.String(),
			Message:   fmt.Sprintf("Failed to create %s request: %v", checkType, err),
			CheckedAt: time.Now(),
			Details: map[string]interface{}{
				"error": err.Error(),
				"url":   url,
			},
		}, err
	}

	// Add custom headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Set user agent
	req.Header.Set("User-Agent", "ephemos-health-checker/1.0")

	// Perform request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return &ports.HealthResult{
			Status:    ports.HealthStatusUnhealthy,
			Component: c.component.String(),
			Message:   fmt.Sprintf("%s check failed: %v", checkType, err),
			CheckedAt: time.Now(),
			Details: map[string]interface{}{
				"error": err.Error(),
				"url":   url,
			},
		}, err
	}
	defer resp.Body.Close()

	// Read response body (limited to prevent abuse)
	bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
	bodyText := string(bodyBytes)

	result := &ports.HealthResult{
		Component: c.component.String(),
		CheckedAt: time.Now(),
		Details: map[string]interface{}{
			"url":            url,
			"status_code":    resp.StatusCode,
			"response_body":  bodyText,
			"content_length": resp.ContentLength,
		},
	}

	// SPIRE health endpoints return 200 for healthy, 503 for unhealthy
	switch resp.StatusCode {
	case http.StatusOK:
		result.Status = ports.HealthStatusHealthy
		result.Message = fmt.Sprintf("%s check passed", checkType)
	case http.StatusServiceUnavailable:
		result.Status = ports.HealthStatusUnhealthy
		result.Message = fmt.Sprintf("%s check failed: service unavailable", checkType)
	default:
		result.Status = ports.HealthStatusUnknown
		result.Message = fmt.Sprintf("%s check returned unexpected status: %d", checkType, resp.StatusCode)
	}

	return result, nil
}
