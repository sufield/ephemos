package domain

import (
	"fmt"
	"time"
)

// HealthStatus represents the overall health status of the service.
// This is a domain value object that encapsulates health information
// in a structured, business-meaningful way.
type HealthStatus struct {
	// Overall indicates if the service is healthy overall
	Overall bool

	// Components contains health status for individual components
	Components map[string]ComponentHealth

	// LastUpdated indicates when this status was last updated
	LastUpdated time.Time

	// Message provides a human-readable status message
	Message string
}

// ComponentHealth represents the health status of an individual component.
type ComponentHealth struct {
	// Healthy indicates if this component is healthy
	Healthy bool

	// LastCheck indicates when this component was last checked
	LastCheck time.Time

	// ErrorMessage contains error details if unhealthy
	ErrorMessage string

	// Metadata contains additional component-specific information
	Metadata map[string]interface{}
}

// NewHealthStatus creates a new health status with default values.
func NewHealthStatus() *HealthStatus {
	return &HealthStatus{
		Overall:     true,
		Components:  make(map[string]ComponentHealth),
		LastUpdated: time.Now(),
		Message:     "Service is healthy",
	}
}

// AddComponent adds a component health status.
func (h *HealthStatus) AddComponent(name string, health ComponentHealth) {
	h.Components[name] = health
	h.LastUpdated = time.Now()

	// Update overall health based on component health
	h.updateOverallHealth()
}

// IsHealthy returns true if the service is overall healthy.
func (h *HealthStatus) IsHealthy() bool {
	return h.Overall
}

// GetComponentHealth returns the health status for a specific component.
func (h *HealthStatus) GetComponentHealth(name string) (ComponentHealth, bool) {
	health, exists := h.Components[name]
	return health, exists
}

// updateOverallHealth calculates overall health based on component health.
func (h *HealthStatus) updateOverallHealth() {
	h.Overall = true
	unhealthyCount := 0

	for _, component := range h.Components {
		if !component.Healthy {
			h.Overall = false
			unhealthyCount++
		}
	}

	// Update message based on health status
	if h.Overall {
		h.Message = "All components are healthy"
	} else {
		h.Message = fmt.Sprintf("%d component(s) are unhealthy", unhealthyCount)
	}
}

// RegistrationStatus represents the SPIRE registration status of the service.
// This is a domain value object that encapsulates registration state.
type RegistrationStatus struct {
	// Registered indicates if the service is registered with SPIRE
	Registered bool

	// SPIFFEID contains the service's SPIFFE ID if registered
	SPIFFEID string

	// Selectors contains the service selectors used for registration
	Selectors []string

	// RegisteredAt indicates when the service was registered
	RegisteredAt time.Time

	// ExpiresAt indicates when the registration expires
	ExpiresAt time.Time

	// ErrorMessage contains error details if registration failed
	ErrorMessage string
}

// NewRegistrationStatus creates a new registration status.
func NewRegistrationStatus() *RegistrationStatus {
	return &RegistrationStatus{
		Registered: false,
		Selectors:  make([]string, 0),
	}
}

// IsRegistered returns true if the service is registered with SPIRE.
func (r *RegistrationStatus) IsRegistered() bool {
	return r.Registered && r.ExpiresAt.After(time.Now())
}

// IsExpired returns true if the registration has expired.
func (r *RegistrationStatus) IsExpired() bool {
	return r.Registered && r.ExpiresAt.Before(time.Now())
}

// TimeUntilExpiry returns the duration until registration expires.
func (r *RegistrationStatus) TimeUntilExpiry() time.Duration {
	if !r.Registered {
		return 0
	}
	return time.Until(r.ExpiresAt)
}
