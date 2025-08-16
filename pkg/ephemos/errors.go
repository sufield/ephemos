package ephemos

import "errors"

// Sentinel errors for stable programmatic error handling.
// These errors can be used with errors.Is() for reliable error detection.
var (
	// ErrNoAuth indicates that no SPIFFE authentication is available for the operation.
	// This typically occurs when the identity provider is not properly configured
	// or the service identity cannot be established.
	ErrNoAuth = errors.New("no SPIFFE authentication available")

	// ErrNoSPIFFEAuth is returned when a SPIFFE-authenticated HTTP client
	// cannot be constructed (e.g., no internal authenticated connection).
	ErrNoSPIFFEAuth = errors.New("no SPIFFE authentication available")

	// ErrInvalidIdentity indicates that the service identity is invalid or malformed.
	// This can occur due to invalid service names, trust domains, or SPIFFE IDs.
	ErrInvalidIdentity = errors.New("invalid service identity")

	// ErrConfigInvalid indicates that the provided configuration is invalid.
	// This includes missing required fields, invalid formats, or conflicting options.
	ErrConfigInvalid = errors.New("invalid configuration")

	// ErrConnectionFailed indicates that establishing a connection to the target service failed.
	// This can be due to network issues, authentication failures, or service unavailability.
	ErrConnectionFailed = errors.New("connection failed")

	// ErrServerClosed indicates that the server has been closed and cannot accept new requests.
	// Operations on a closed server will return this error.
	ErrServerClosed = errors.New("server closed")

	// ErrInvalidAddress indicates that the provided network address is invalid.
	// This includes malformed host:port combinations or unsupported address formats.
	ErrInvalidAddress = errors.New("invalid network address")

	// ErrTimeout indicates that an operation exceeded its configured timeout.
	// This can occur during connection establishment, authentication, or request processing.
	ErrTimeout = errors.New("operation timeout")
)
