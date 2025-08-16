// Package http provides HTTP transport adapters for domain services.
// This package maps plain Go domain interfaces to HTTP/REST implementations.
package http

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/sufield/ephemos/internal/core/ports"
)

// Adapter manages the mapping between domain services and HTTP handlers.
type Adapter struct {
	registry *ports.ServiceRegistry
	mux      *http.ServeMux
}

// NewAdapter creates a new HTTP adapter.
func NewAdapter(mux *http.ServeMux) *Adapter {
	return &Adapter{
		registry: ports.NewServiceRegistry(),
		mux:      mux,
	}
}

// Mount registers a domain service implementation with the HTTP server.
// It creates RESTful HTTP handlers based on the service methods.
func (a *Adapter) Mount(impl interface{}) error {
	serviceName := a.getServiceName(impl)

	// Register the service in our registry
	if err := a.registry.Register(serviceName, impl); err != nil {
		return fmt.Errorf("failed to register service: %w", err)
	}

	// Create HTTP handlers based on the interface type
	return a.createHTTPHandlers(serviceName, impl)
}

// getServiceName determines the service name from the implementation.
func (a *Adapter) getServiceName(impl interface{}) string {
	implType := reflect.TypeOf(impl)
	if implType.Kind() == reflect.Ptr {
		implType = implType.Elem()
	}
	return strings.ToLower(implType.Name())
}

// createHTTPHandlers creates HTTP handlers based on the service interface.
func (a *Adapter) createHTTPHandlers(serviceName string, impl interface{}) error {
	// Check what interfaces the implementation satisfies
	switch service := impl.(type) {
	default:
		return a.mountGenericService(serviceName, service)
	}
}

// mountGenericService creates HTTP handlers for unknown service types.
func (a *Adapter) mountGenericService(serviceName string, _ interface{}) error {
	// Use reflection to create handlers based on method signatures
	return fmt.Errorf("generic service mounting not yet implemented for %s", serviceName)
}

