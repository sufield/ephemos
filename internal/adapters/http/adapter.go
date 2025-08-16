// Package http provides HTTP transport adapters for domain services.
// This package maps plain Go domain interfaces to HTTP/REST implementations.
package http

import (
	"encoding/json"
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
	case ports.EchoService:
		return a.mountEchoService(serviceName, service)
	default:
		return a.mountGenericService(serviceName, impl)
	}
}

// mountEchoService creates HTTP handlers for EchoService.
func (a *Adapter) mountEchoService(serviceName string, service ports.EchoService) error {
	basePath := fmt.Sprintf("/%s", serviceName)

	// POST /{service}/echo
	a.mux.HandleFunc(fmt.Sprintf("%s/echo", basePath), func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			Message string `json:"message"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		result, err := service.Echo(r.Context(), req.Message)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		resp := struct {
			Message string `json:"message"`
		}{
			Message: result,
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		}
	})

	// POST /{service}/ping
	a.mux.HandleFunc(fmt.Sprintf("%s/ping", basePath), func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if err := service.Ping(r.Context()); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	})

	return nil
}

// mountGenericService creates HTTP handlers for unknown service types.
func (a *Adapter) mountGenericService(serviceName string, _ interface{}) error {
	// Use reflection to create handlers based on method signatures
	return fmt.Errorf("generic service mounting not yet implemented for %s", serviceName)
}

