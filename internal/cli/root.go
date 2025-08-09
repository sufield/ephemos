package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{ //nolint:gochecknoglobals // Cobra command pattern
	Use:   "ephemos",
	Short: "Identity-based authentication CLI for SPIFFE/SPIRE services",
	Long: `Identity-based authentication CLI for SPIFFE/SPIRE services.

Ephemos provides identity-based authentication for backend services using SPIFFE/SPIRE.
Use this CLI to register services, manage identities, and configure authentication policies.`,
}

// Execute runs the CLI.
func Execute() error {
	if err := rootCmd.Execute(); err != nil {
		return fmt.Errorf("failed to execute command: %w", err)
	}
	return nil
}

func init() { //nolint:gochecknoinits // Cobra requires init for command setup
	rootCmd.AddCommand(registerCmd)
}
