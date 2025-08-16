// Package ports defines domain service interfaces using plain Go types.
// These interfaces are transport-agnostic and contain no protocol buffer dependencies.
package ports


// ServiceRegistry maps service interface types to their transport implementations.
// This allows the Mount[T] function to dynamically register services without compile-time coupling.
type ServiceRegistry struct {
	services map[string]ServiceDescriptor
}

// ServiceDescriptor contains metadata about a registered service.
type ServiceDescriptor struct {
	Name        string
	ServiceType string
	Impl        interface{}
	Methods     []MethodDescriptor
}

// MethodDescriptor describes a service method for transport mapping.
type MethodDescriptor struct {
	Name       string
	InputType  string
	OutputType string
}

// NewServiceRegistry creates a new service registry.
func NewServiceRegistry() *ServiceRegistry {
	return &ServiceRegistry{
		services: make(map[string]ServiceDescriptor),
	}
}

// Register adds a service implementation to the registry.
func (r *ServiceRegistry) Register(name string, impl interface{}) error {
	descriptor := ServiceDescriptor{
		Name:        name,
		ServiceType: getServiceType(impl),
		Impl:        impl,
		Methods:     extractMethods(impl),
	}

	r.services[name] = descriptor
	return nil
}

// GetService retrieves a service descriptor by name.
func (r *ServiceRegistry) GetService(name string) (ServiceDescriptor, bool) {
	desc, exists := r.services[name]
	return desc, exists
}

// ListServices returns all registered services.
func (r *ServiceRegistry) ListServices() []ServiceDescriptor {
	services := make([]ServiceDescriptor, 0, len(r.services))
	for _, desc := range r.services {
		services = append(services, desc)
	}
	return services
}

// Common domain service interfaces that users can implement.
// These use plain Go types and are completely transport-agnostic.


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

// Helper functions for service registration and reflection.

func getServiceType(_ interface{}) string {
	// Use reflection to determine the service type
	// This would be implemented using reflection to get the interface type
	return "unknown" // Placeholder
}

func extractMethods(_ interface{}) []MethodDescriptor {
	// Use reflection to extract method signatures
	// This would be implemented using reflection to analyze the interface
	return nil // Placeholder
}
