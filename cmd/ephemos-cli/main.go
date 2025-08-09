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
	"fmt"
	"os"

	"github.com/sufield/ephemos/internal/cli"
)

// main is the entry point for the Ephemos CLI tool.
// It executes the root command and handles any errors that occur.
func main() {
	if err := cli.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
