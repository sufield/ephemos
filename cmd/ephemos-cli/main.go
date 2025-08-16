// Package main provides the Ephemos CLI tool for production SPIFFE/SPIRE service management
// and identity-based authentication administration.
//
// The ephemos CLI tool is a production utility for system administrators,
// DevOps engineers, and developers managing Ephemos-based services in
// development and production environments.
//
// Core functionality includes:
//   - Service registration with SPIRE server
//   - Configuration validation and management
//   - Identity verification and diagnostics
//   - SPIRE infrastructure health checks
//   - Certificate and trust bundle inspection
//   - Service selector management
//
// Usage:
//
//	ephemos register --config config.yaml --selector unix:user:1000
//	ephemos validate --config config.yaml
//	ephemos health --config config.yaml --verbose
//
// The tool integrates with SPIRE infrastructure to provide streamlined
// service identity management for microservices and distributed systems.
// It abstracts SPIFFE/SPIRE complexity while providing full administrative
// control over identity policies and service registration.
//
// This is a production CLI binary built from cmd/ephemos-cli according to
// Go project layout conventions for production command-line tools.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/sufield/ephemos/internal/cli"
)

// Build information - injected at compile time via ldflags/x_defs
var (
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
	buildUser = "unknown"
	buildHost = "unknown"
)

// Exit codes for different types of failures
const (
	exitOK       = 0
	exitUsage    = 2
	exitConfig   = 3
	exitRuntime  = 4
	exitInternal = 10
)

// main is the entry point for the Ephemos CLI tool.
// It sets up signal handling, executes the CLI with context, and handles errors with appropriate exit codes.
func main() {
	// Inject build information into the CLI package
	cli.Version = version
	cli.Commit = commit
	cli.BuildDate = buildDate
	cli.BuildUser = buildUser
	cli.BuildHost = buildHost

	// Create a context that cancels on SIGINT or SIGTERM
	ctx, stop := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Execute the CLI with context
	if err := cli.ExecuteContext(ctx); err != nil {
		// Redact sensitive information from error messages
		redactedError := cli.RedactError(err)
		code := classifyExitCode(err)
		
		// Cobra already prints usage errors, so only print for non-usage errors
		if code != exitUsage {
			fmt.Fprintf(os.Stderr, "Error: %s\n", redactedError)
		}
		os.Exit(code)
	}
}

// classifyExitCode maps error types to appropriate exit codes for CI/automation
func classifyExitCode(err error) int {
	switch {
	case errors.Is(err, cli.ErrUsage):
		return exitUsage
	case errors.Is(err, cli.ErrConfig):
		return exitConfig
	case errors.Is(err, cli.ErrAuth):
		return exitRuntime // Auth failures are runtime issues
	case errors.Is(err, cli.ErrRuntime):
		return exitRuntime
	case errors.Is(err, cli.ErrInternal):
		return exitInternal
	case errors.Is(err, context.Canceled):
		// Graceful shutdown via signal
		return exitOK
	case errors.Is(err, context.DeadlineExceeded):
		// Timeout
		return exitRuntime
	default:
		// Unknown error type
		return exitRuntime
	}
}
