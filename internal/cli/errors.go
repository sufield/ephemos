package cli

import "errors"

// Sentinel errors for exit code classification
var (
	// ErrUsage indicates invalid command usage, flags, or arguments
	ErrUsage = errors.New("usage error")
	
	// ErrConfig indicates invalid or unsafe configuration
	ErrConfig = errors.New("configuration error")
	
	// ErrAuth indicates SPIFFE/SPIRE authentication failures
	ErrAuth = errors.New("authentication error")
	
	// ErrRuntime indicates runtime execution failures
	ErrRuntime = errors.New("runtime error")
	
	// ErrInternal indicates internal system errors
	ErrInternal = errors.New("internal error")
)