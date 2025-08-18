// Package main provides a configuration validation tool for Ephemos.
// This tool helps validate configuration settings before deployment to production.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/sufield/ephemos/internal/adapters/secondary/config"
	"github.com/sufield/ephemos/internal/core/ports"
)

// Version information - set by build
var (
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
)

// Global flags
var (
	configFile string
	envOnly    bool
	production bool
	verbose    bool
	noEmoji    bool
	format     string
	quiet      bool
	timeout    time.Duration
)

// Exit codes
const (
	ExitSuccess             = 0
	ExitUsageError          = 2
	ExitBasicValidation     = 3
	ExitProductionReadiness = 4
	ExitLoadError           = 5
)

var rootCmd = &cobra.Command{
	Use:   "config-validator",
	Short: "Validates Ephemos configuration for security and production readiness",
	Long: `Validates Ephemos configuration for security and production readiness.

This tool helps ensure your Ephemos configuration is secure and ready for production deployment.
It validates service names, trust domains, socket paths, and other security settings.`,
	Example: `  # Validate environment variables (most secure):
  config-validator --env-only --production

  # Validate config file with environment override:
  config-validator --config config/production.yaml --production

  # JSON output for CI:
  config-validator --env-only --format json --production

  # Verbose validation:
  config-validator --env-only --verbose`,
	RunE: runValidator,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Run: func(cmd *cobra.Command, args []string) {
		if format == "json" {
			versionInfo := map[string]string{
				"version":    version,
				"commit":     commit,
				"build_date": buildDate,
			}
			encoder := json.NewEncoder(os.Stdout)
			encoder.SetIndent("", "  ")
			encoder.Encode(versionInfo)
		} else {
			fmt.Printf("ephemos-config-validator version %s\n", version)
			fmt.Printf("  Commit: %s\n", commit)
			fmt.Printf("  Built:  %s\n", buildDate)
		}
	},
}

func init() {
	// Add version subcommand
	rootCmd.AddCommand(versionCmd)

	// Persistent flags (available to all commands)
	rootCmd.PersistentFlags().StringVar(&format, "format", "text", "Output format (text|json)")
	rootCmd.PersistentFlags().BoolVar(&noEmoji, "no-emoji", false, "Disable emoji in output")
	rootCmd.PersistentFlags().BoolVar(&quiet, "quiet", false, "Suppress success messages")
	rootCmd.PersistentFlags().DurationVar(&timeout, "timeout", 30*time.Second, "Timeout for operations")

	// Root command flags
	rootCmd.Flags().StringVar(&configFile, "config", "", "Path to configuration file")
	rootCmd.Flags().BoolVar(&envOnly, "env-only", false, "Validate environment variables only (most secure)")
	rootCmd.Flags().BoolVar(&production, "production", false, "Perform production readiness validation")
	rootCmd.Flags().BoolVar(&verbose, "verbose", false, "Verbose output")

	// Make flags mutually exclusive where appropriate
	rootCmd.MarkFlagsMutuallyExclusive("config", "env-only")
}

func runValidator(cmd *cobra.Command, args []string) error {
	// Create printer with proper writer injection
	printer := NewPrinter(os.Stdout, os.Stderr, !noEmoji, quiet)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Header for text mode
	if format == "text" && !quiet {
		printer.Plain("Ephemos Configuration Validator")
		printer.Plain("==============================")
		printer.Newline()
	}

	// Validate flag combination
	if configFile == "" && !envOnly {
		return fmt.Errorf("either --config or --env-only must be specified")
	}

	// Load configuration with proper precedence
	cfg, err := loadConfigurationCobra(ctx, printer, configFile, envOnly)
	if err != nil {
		handleLoadErrorCobra(printer, format, err)
		os.Exit(ExitLoadError)
	}

	// Create result object for JSON mode
	result := &Result{
		BasicValid:      true,
		ProductionValid: true,
		Configuration: &Config{
			ServiceName: cfg.Service.Name.Value(),
			TrustDomain: cfg.Service.Domain,
		},
	}

	if cfg.Agent != nil {
		result.Configuration.AgentSocket = cfg.Agent.SocketPath.Value()
	}

	// Display configuration if verbose
	if verbose && format == "text" {
		displayConfigurationCobra(printer, cfg)
		printer.Newline()
	}

	// Perform basic validation
	if format == "text" {
		printer.Info("Performing basic validation...")
	}

	if err := cfg.Validate(); err != nil {
		result.BasicValid = false
		result.Errors = append(result.Errors, err.Error())

		if format == "json" {
			return printer.PrintJSON(result)
		} else {
			printer.Errorf("Basic validation failed: %v", err)
			os.Exit(ExitBasicValidation)
		}
	}

	if format == "text" {
		printer.Success("Basic validation passed")
		result.Messages = append(result.Messages, "Basic validation passed")
	}

	// Perform production validation if requested
	if production {
		if format == "text" {
			printer.Newline()
			printer.Production("Performing production readiness validation...")
		}

		if err := cfg.IsProductionReady(); err != nil {
			result.ProductionValid = false
			tips := getProductionTips(err)
			result.Tips = tips

			if format == "json" {
				result.Errors = append(result.Errors, err.Error())
				return printer.PrintJSON(result)
			} else {
				printer.Errorf("Production validation failed: %v", err)
				printer.Newline()
				printer.Tip("Production readiness tips:")
				for _, tip := range tips {
					printer.Bullet(tip)
				}
				os.Exit(ExitProductionReadiness)
			}
		}

		if format == "text" {
			printer.Success("Production validation passed")
			result.Messages = append(result.Messages, "Production validation passed")
		}
	}

	// Show security recommendations
	if format == "text" && !quiet {
		printer.Newline()
		printer.Tip("Security recommendations:")
		recommendations := getSecurityRecommendations(envOnly)
		for _, rec := range recommendations {
			if len(rec) > 0 && rec[0] == ' ' {
				printer.Plain(rec)
			} else {
				printer.Bullet(rec)
			}
		}
	}

	// Final success message or JSON output
	if format == "text" && !quiet {
		printer.Newline()
		printer.Banner("Configuration validation completed successfully!")

		if production {
			printer.Success("Configuration is ready for production deployment")
		}
	} else if format == "json" {
		return printer.PrintJSON(result)
	}

	return nil
}

// loadConfigurationCobra loads configuration with proper precedence
func loadConfigurationCobra(ctx context.Context, printer *Printer, configFile string, envOnly bool) (*ports.Configuration, error) {
	switch {
	case configFile != "":
		printer.File(fmt.Sprintf("Loading configuration from file: %s", configFile))
		provider := config.NewFileProvider()
		cfg, err := provider.LoadConfiguration(ctx, configFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load configuration file: %w", err)
		}

		printer.Cycle("Merging with environment variables (environment overrides file)")
		if err := cfg.MergeWithEnvironment(); err != nil {
			return nil, fmt.Errorf("failed to merge with environment: %w", err)
		}

		printer.Success("Configuration loaded successfully")
		return cfg, nil

	case envOnly:
		printer.Lock("Loading configuration from environment variables (secure mode)")
		cfg, err := ports.LoadFromEnvironment()
		if err != nil {
			return nil, fmt.Errorf("failed to load configuration from environment: %w", err)
		}
		printer.Success("Configuration loaded successfully")
		return cfg, nil

	default:
		return nil, fmt.Errorf("either --config or --env-only must be specified")
	}
}

// displayConfigurationCobra displays configuration details
func displayConfigurationCobra(printer *Printer, cfg *ports.Configuration) {
	printer.Section("Configuration Details:")
	printer.Infof("   Service Name: %s", cfg.Service.Name)
	printer.Infof("   Trust Domain: %s", cfg.Service.Domain)

	if cfg.Agent != nil {
		printer.Infof("   Agent Socket: %s", cfg.Agent.SocketPath)
	}
}

// handleLoadErrorCobra handles configuration load errors
func handleLoadErrorCobra(printer *Printer, format string, err error) {
	if format == "json" {
		result := &Result{
			BasicValid: false,
			Errors:     []string{err.Error()},
		}
		printer.PrintJSON(result)
	} else {
		printer.Errorf("Failed to load configuration: %v", err)
	}
}

// Execute runs the CLI application
func Execute() error {
	return rootCmd.Execute()
}

func main() {
	if err := Execute(); err != nil {
		os.Exit(1)
	}
}
