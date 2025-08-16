// Package main provides a configuration validation tool for Ephemos.
// This tool helps validate configuration settings before deployment to production.
//
//nolint:forbidigo // CLI tool requires direct output to stdout
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/sufield/ephemos/internal/adapters/secondary/config"
	"github.com/sufield/ephemos/internal/core/ports"
)

// Version information
var (
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
)

// Exit codes
const (
	ExitSuccess            = 0
	ExitUsageError         = 2
	ExitBasicValidation    = 3
	ExitProductionReadiness = 4
	ExitLoadError          = 5
)

func main() {
	var (
		configFile = flag.String("config", "", "Path to configuration file")
		envOnly    = flag.Bool("env-only", false, "Validate environment variables only (most secure)")
		production = flag.Bool("production", false, "Perform production readiness validation")
		verbose    = flag.Bool("verbose", false, "Verbose output")
		showVersion = flag.Bool("version", false, "Show version information")
		noEmoji    = flag.Bool("no-emoji", false, "Disable emoji in output")
		format     = flag.String("format", "text", "Output format (text|json)")
		quiet      = flag.Bool("quiet", false, "Suppress success messages")
		timeout    = flag.Duration("timeout", 30*time.Second, "Timeout for operations")
	)

	setupUsage()
	flag.Parse()

	// Create printer with proper writer injection
	printer := NewPrinter(os.Stdout, os.Stderr, !*noEmoji, *quiet)

	// Handle version flag
	if *showVersion {
		printVersion(printer, *format)
		os.Exit(ExitSuccess)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	// Header for text mode
	if *format == "text" && !*quiet {
		printer.Plain("Ephemos Configuration Validator")
		printer.Plain("==============================")
		printer.Newline()
	}

	// Load configuration with proper precedence
	cfg, err := loadConfiguration(ctx, printer, *configFile, *envOnly)
	if err != nil {
		handleLoadError(printer, *format, err)
		os.Exit(ExitLoadError)
	}

	// Create result object for JSON mode
	result := &Result{
		BasicValid:      true,
		ProductionValid: true,
		Configuration: &Config{
			ServiceName: cfg.Service.Name,
			TrustDomain: cfg.Service.Domain,
		},
	}

	if cfg.Agent != nil {
		result.Configuration.AgentSocket = cfg.Agent.SocketPath
	}

	// Display configuration if verbose
	if *verbose && *format == "text" {
		displayConfiguration(printer, cfg)
		printer.Newline()
	}

	// Perform basic validation
	if *format == "text" {
		printer.Info("Performing basic validation...")
	}

	if err := cfg.Validate(); err != nil {
		result.BasicValid = false
		result.Errors = append(result.Errors, err.Error())

		if *format == "json" {
			printer.PrintJSON(result)
		} else {
			printer.Errorf("Basic validation failed: %v", err)
		}
		os.Exit(ExitBasicValidation)
	}

	if *format == "text" {
		printer.Success("Basic validation passed")
		result.Messages = append(result.Messages, "Basic validation passed")
	}

	// Perform production validation if requested
	if *production {
		if *format == "text" {
			printer.Newline()
			printer.Production("Performing production readiness validation...")
		}

		if err := cfg.IsProductionReady(); err != nil {
			result.ProductionValid = false
			tips := getProductionTips(err)
			result.Tips = tips

			if *format == "json" {
				result.Errors = append(result.Errors, err.Error())
				printer.PrintJSON(result)
			} else {
				printer.Errorf("Production validation failed: %v", err)
				printer.Newline()
				printer.Tip("Production readiness tips:")
				for _, tip := range tips {
					printer.Bullet(tip)
				}
			}
			os.Exit(ExitProductionReadiness)
		}

		if *format == "text" {
			printer.Success("Production validation passed")
			result.Messages = append(result.Messages, "Production validation passed")
		}
	}

	// Show security recommendations
	if *format == "text" && !*quiet {
		printer.Newline()
		printer.Tip("Security recommendations:")
		recommendations := getSecurityRecommendations(*envOnly)
		for _, rec := range recommendations {
			if rec[0] == ' ' {
				printer.Plain(rec)
			} else {
				printer.Bullet(rec)
			}
		}
	}

	// Final success message
	if *format == "text" && !*quiet {
		printer.Newline()
		printer.Banner("Configuration validation completed successfully!")

		if *production {
			printer.Success("Configuration is ready for production deployment")
		}
	} else if *format == "json" {
		printer.PrintJSON(result)
	}

	os.Exit(ExitSuccess)
}

// setupUsage configures the custom usage function
func setupUsage() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Validates Ephemos configuration for security and production readiness.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  # Validate environment variables (most secure):\n")
		fmt.Fprintf(os.Stderr, "  %s --env-only --production\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Validate config file with environment override:\n")
		fmt.Fprintf(os.Stderr, "  %s --config config/production.yaml --production\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # JSON output for CI:\n")
		fmt.Fprintf(os.Stderr, "  %s --env-only --format json --production\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Verbose validation:\n")
		fmt.Fprintf(os.Stderr, "  %s --env-only --verbose\n\n", os.Args[0])
	}
}

// loadConfiguration loads configuration with proper precedence
func loadConfiguration(ctx context.Context, printer *Printer, configFile string, envOnly bool) (*ports.Configuration, error) {
	// Precedence: config file (if specified) with env override, or env-only
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

// displayConfiguration displays configuration details
func displayConfiguration(printer *Printer, cfg *ports.Configuration) {
	printer.Section("Configuration Details:")
	printer.Infof("   Service Name: %s", cfg.Service.Name)
	printer.Infof("   Trust Domain: %s", cfg.Service.Domain)

	if cfg.Agent != nil {
		printer.Infof("   Agent Socket: %s", cfg.Agent.SocketPath)
	}
}

// handleLoadError handles configuration load errors
func handleLoadError(printer *Printer, format string, err error) {
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

// printVersion prints version information
func printVersion(printer *Printer, format string) {
	if format == "json" {
		versionInfo := map[string]string{
			"version":    version,
			"commit":     commit,
			"build_date": buildDate,
		}
		printer.PrintJSON(versionInfo)
	} else {
		printer.Plain(fmt.Sprintf("ephemos-config-validator version %s", version))
		printer.Plain(fmt.Sprintf("  Commit: %s", commit))
		printer.Plain(fmt.Sprintf("  Built:  %s", buildDate))
	}
}