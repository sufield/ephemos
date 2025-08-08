// ephemos-cli is the command-line interface for the Ephemos identity-based authentication library.
//
// This CLI provides utilities for managing SPIFFE/SPIRE service registrations, identities,
// and authentication policies. It simplifies common operations like:
//   - Registering services with SPIRE server
//   - Managing service selectors and SPIFFE IDs
//   - Configuring trust domains and authentication policies
//
// Usage:
//   ephemos-cli register <service-name> [flags]
//   ephemos-cli --help
//
// For more information, see the Ephemos documentation.
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