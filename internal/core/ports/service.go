// Package ports defines domain service interfaces using plain Go types.
// These interfaces are transport-agnostic and contain no protocol buffer dependencies.
package ports



// HealthStatus represents the health status of a service.
type HealthStatus struct {
	Service string
	Status  HealthStatusType
	Message string
}

// HealthStatusType represents different health states.
type HealthStatusType int

const (
	// HealthStatusUnknown indicates the health status is unknown.
	HealthStatusUnknown HealthStatusType = iota
	// HealthStatusServing indicates the service is healthy and serving requests.
	HealthStatusServing
	// HealthStatusNotServing indicates the service is not serving requests.
	HealthStatusNotServing
	// HealthStatusServiceUnknown indicates the requested service is unknown.
	HealthStatusServiceUnknown
)

