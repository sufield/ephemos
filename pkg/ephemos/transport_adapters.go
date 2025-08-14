// Package ephemos provides transport adapter implementations.
package ephemos

import (
	"fmt"
	"net/http"
	"reflect"
	"sync"
)

// httpAdapterImpl implements HTTPAdapter.
type httpAdapterImpl struct {
	httpServer *http.Server
	mux        *http.ServeMux
	middleware []func(http.Handler) http.Handler
	services   map[string]interface{}
	mu         sync.RWMutex
}

// NewHTTPAdapter creates a new HTTP adapter.
func NewHTTPAdapter(addr string) HTTPAdapter {
	mux := http.NewServeMux()
	adapter := &httpAdapterImpl{
		mux:      mux,
		services: make(map[string]interface{}),
		httpServer: &http.Server{
			Addr:    addr,
			Handler: mux,
		},
	}
	return adapter
}

// Mount mounts a service implementation.
func (a *httpAdapterImpl) Mount(service interface{}) error {
	if service == nil {
		return fmt.Errorf("service cannot be nil")
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	// Store the service
	serviceType := reflect.TypeOf(service).String()
	a.services[serviceType] = service

	// In a real implementation, this would set up HTTP routes
	// based on the service interface methods
	// For now, we'll create a simple handler
	a.mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	return nil
}

// GetServer returns the underlying server.
func (a *httpAdapterImpl) GetServer() interface{} {
	return a.httpServer
}

// ConfigureMiddleware sets up HTTP middleware.
func (a *httpAdapterImpl) ConfigureMiddleware(middleware ...func(http.Handler) http.Handler) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.middleware = append(a.middleware, middleware...)

	// Apply middleware to the handler
	handler := http.Handler(a.mux)
	for i := len(a.middleware) - 1; i >= 0; i-- {
		handler = a.middleware[i](handler)
	}
	a.httpServer.Handler = handler
}

// GetHTTPServer returns the underlying HTTP server.
func (a *httpAdapterImpl) GetHTTPServer() *http.Server {
	return a.httpServer
}
