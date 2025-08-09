package cli

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "ephemos",
	Short: "Identity-based authentication CLI for SPIFFE/SPIRE services",
	Long: `Identity-based authentication CLI for SPIFFE/SPIRE services.

Ephemos provides identity-based authentication for backend services using SPIFFE/SPIRE.
Use this CLI to register services, manage identities, and configure authentication policies.`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(registerCmd)
}
