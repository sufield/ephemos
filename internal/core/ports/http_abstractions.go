// Package ports - HTTP abstractions that don't leak net/http infrastructure types.
// These types provide proper abstractions for HTTP operations in hexagonal architecture.
package ports

import (
	"context"
	"io"
)

// HTTPRequest abstracts an HTTP request without leaking net/http or net/url.
type HTTPRequest struct {
	Method  string
	URL     string              // Use string to avoid net/url dependency; parse in adapter if needed.
	Headers map[string][]string // Use []string for multi-value headers (more accurate than map[string]string).
	Body    io.Reader
}

// HTTPResponse abstracts an HTTP response without leaking net/http.
type HTTPResponse struct {
	StatusCode int
	Headers    map[string][]string
	Body       io.ReadCloser
}

// HTTPClientPort provides authenticated HTTP capabilities via abstractions.
// This interface replaces direct usage of *http.Client in port signatures.
type HTTPClientPort interface {
	// Do executes the request with automatic authentication.
	// The implementation should handle authentication credentials automatically.
	Do(ctx context.Context, req *HTTPRequest) (*HTTPResponse, error)

	// Close releases resources held by the HTTP client.
	// Must be safe to call multiple times.
	Close() error
}
