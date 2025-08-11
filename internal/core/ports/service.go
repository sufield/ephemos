// Package ports defines domain service interfaces using plain Go types.
// These interfaces are transport-agnostic and contain no protocol buffer dependencies.
package ports

import (
	"context"
	"io"
)

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
	Name           string
	InputType      string
	OutputType     string
	IsStreaming    bool
	IsClientStream bool
	IsServerStream bool
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

// EchoService demonstrates a simple request-response service.
type EchoService interface {
	Echo(ctx context.Context, message string) (string, error)
	Ping(ctx context.Context) error
}

// StreamingService demonstrates streaming capabilities.
type StreamingService interface {
	ServerStream(ctx context.Context, input string, stream ServerStream[string]) error
	ClientStream(ctx context.Context, stream ClientStream[string]) (string, error)
	BidirectionalStream(ctx context.Context, stream BidiStream[string, string]) error
}

// ServerStream represents a server-side streaming interface.
type ServerStream[T any] interface {
	Send(T) error
	Context() context.Context
}

// ClientStream represents a client-side streaming interface.
type ClientStream[T any] interface {
	Recv() (T, error)
	Context() context.Context
}

// BidiStream represents a bidirectional streaming interface.
type BidiStream[Req, Resp any] interface {
	Send(Resp) error
	Recv() (Req, error)
	Context() context.Context
}

// FileService demonstrates binary data handling.
type FileService interface {
	Upload(ctx context.Context, filename string, data io.Reader) error
	Download(ctx context.Context, filename string) (io.Reader, error)
	List(ctx context.Context, prefix string) ([]string, error)
}

// HealthService provides standardized health checking.
type HealthService interface {
	Check(ctx context.Context, service string) (HealthStatus, error)
	Watch(ctx context.Context, service string, stream ServerStream[HealthStatus]) error
}

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
