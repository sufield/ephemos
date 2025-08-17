package cli

import "errors"

// Minimal sentinel errors - let Cobra handle usage/flag errors
var (
	// ErrConfig indicates invalid or unsafe configuration (business logic)
	ErrConfig = errors.New("configuration error")

	// ErrAuth indicates SPIFFE/SPIRE authentication failures
	ErrAuth = errors.New("authentication error")
)
