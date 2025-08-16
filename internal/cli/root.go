package cli

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/spf13/cobra"
)

// Build information - injected at compile time via ldflags/x_defs
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
	BuildUser = "unknown"
	BuildHost = "unknown"
)

var rootCmd = &cobra.Command{ //nolint:gochecknoglobals // Cobra command pattern
	Use:   "ephemos",
	Short: "Identity-based authentication CLI for SPIFFE/SPIRE services",
	Long: `Identity-based authentication CLI for SPIFFE/SPIRE services.

Ephemos provides identity-based authentication for backend services using SPIFFE/SPIRE.
Use this CLI to register services, manage identities, and configure authentication policies.

The CLI provides commands for service registration, configuration validation,
identity verification, and SPIRE infrastructure management.`,
	Version: getVersionString(),
}

// getVersionString returns formatted version information for Cobra's built-in version support
func getVersionString() string {
	return fmt.Sprintf("%s\nCommit: %s\nBuild Date: %s\nBuild User: %s\nBuild Host: %s\nGo Version: %s\nOS/Arch: %s/%s",
		Version, Commit, BuildDate, BuildUser, BuildHost, runtime.Version(), runtime.GOOS, runtime.GOARCH)
}

// Execute runs the CLI without context (for backward compatibility).
func Execute() error {
	return ExecuteContext(context.Background())
}

// ExecuteContext runs the CLI with the provided context.
func ExecuteContext(ctx context.Context) error {
	// Apply global timeout if set
	timeout, _ := rootCmd.PersistentFlags().GetDuration("timeout")
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	return rootCmd.ExecuteContext(ctx)
}

func init() { //nolint:gochecknoinits // Cobra requires init for command setup
	// Persistent flags available to all commands
	rootCmd.PersistentFlags().Bool("quiet", false, "Suppress non-essential output")
	rootCmd.PersistentFlags().Bool("no-emoji", false, "Disable emoji in output")
	rootCmd.PersistentFlags().String("format", "text", "Output format (text|json)")
	rootCmd.PersistentFlags().Duration("timeout", 30*time.Second, "Global timeout for operations")

	// Cobra's built-in version template with custom formatting
	rootCmd.SetVersionTemplate("{{.Version}}\n")

	// Add subcommands
	rootCmd.AddCommand(registerCmd)
}
