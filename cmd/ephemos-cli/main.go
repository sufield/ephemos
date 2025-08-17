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

// Exit codes - Cobra handles usage errors (exit 2) automatically
const (
	exitOK     = 0
	exitConfig = 3 // Configuration/business logic errors
	exitAuth   = 4 // Authentication errors
)

// main is the entry point for the Ephemos CLI tool.
// It sets up signal handling, executes the CLI with context, and handles errors with appropriate exit codes.
func main() {
	// Create a context that cancels on SIGINT or SIGTERM
	ctx, stop := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Execute the CLI with context - Cobra handles most error display
	if err := cli.ExecuteContext(ctx); err != nil {
		// Only handle business logic errors - Cobra handles usage/flag errors
		code := exitOK

		// Check for specific business logic errors
		switch {
		case errors.Is(err, cli.ErrConfig):
			code = exitConfig
		case errors.Is(err, cli.ErrAuth):
			code = exitAuth
		case errors.Is(err, context.Canceled):
			// Graceful shutdown via signal - no error message needed
			os.Exit(exitOK)
		default:
			// For any other error, Cobra has already printed it
			code = 1
		}

		// Redact sensitive information if we need to print
		if code != 0 && !errors.Is(err, context.Canceled) {
			// Only print if Cobra hasn't already (for our custom errors)
			if errors.Is(err, cli.ErrConfig) || errors.Is(err, cli.ErrAuth) {
				redactedError := cli.RedactError(err)
				fmt.Fprintf(os.Stderr, "Error: %s\n", redactedError)
			}
		}

		os.Exit(code)
	}
}
