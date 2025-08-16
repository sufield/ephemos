package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/spf13/cobra"
)

// Global flags
var (
	globalQuiet   bool
	globalNoEmoji bool
	globalFormat  string
	globalTimeout time.Duration
)

var rootCmd = &cobra.Command{ //nolint:gochecknoglobals // Cobra command pattern
	Use:   "ephemos",
	Short: "Identity-based authentication CLI for SPIFFE/SPIRE services",
	Long: `Identity-based authentication CLI for SPIFFE/SPIRE services.

Ephemos provides identity-based authentication for backend services using SPIFFE/SPIRE.
Use this CLI to register services, manage identities, and configure authentication policies.

The CLI provides commands for service registration, configuration validation,
identity verification, and SPIRE infrastructure management.`,
	Version: Version,
}

// Execute runs the CLI without context (for backward compatibility).
func Execute() error {
	return ExecuteContext(context.Background())
}

// ExecuteContext runs the CLI with the provided context.
func ExecuteContext(ctx context.Context) error {
	// Apply global timeout if set
	if globalTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, globalTimeout)
		defer cancel()
	}

	rootCmd.SetContext(ctx)
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		return fmt.Errorf("failed to execute command: %w", err)
	}
	return nil
}

// GetOutputWriter returns the appropriate output writer based on global flags
func GetOutputWriter() io.Writer {
	return os.Stdout
}

// GetErrorWriter returns the appropriate error writer based on global flags
func GetErrorWriter() io.Writer {
	return os.Stderr
}

// IsQuiet returns true if quiet mode is enabled
func IsQuiet() bool {
	return globalQuiet
}

// IsEmojiDisabled returns true if emoji is disabled
func IsEmojiDisabled() bool {
	return globalNoEmoji
}

// GetFormat returns the output format
func GetFormat() string {
	return globalFormat
}

// GetTimeout returns the global timeout duration
func GetTimeout() time.Duration {
	return globalTimeout
}

func init() { //nolint:gochecknoinits // Cobra requires init for command setup
	// Persistent flags available to all commands
	rootCmd.PersistentFlags().BoolVar(&globalQuiet, "quiet", false, "Suppress non-essential output")
	rootCmd.PersistentFlags().BoolVar(&globalNoEmoji, "no-emoji", false, "Disable emoji in output")
	rootCmd.PersistentFlags().StringVar(&globalFormat, "format", "text", "Output format (text|json)")
	rootCmd.PersistentFlags().DurationVar(&globalTimeout, "timeout", 30*time.Second, "Global timeout for operations")

	// Add version flag at root level
	rootCmd.Flags().BoolP("version", "v", false, "Show version information")

	// Add subcommands
	rootCmd.AddCommand(registerCmd)
}
