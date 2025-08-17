package cli

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/spf13/cobra"
	"github.com/sufield/ephemos/internal/adapters/secondary/health"
	"github.com/sufield/ephemos/internal/core/ports"
	"github.com/sufield/ephemos/internal/core/services"
)

// Output templates for health check results
const healthOverallTemplate = `{{.Icon}} Overall Health: {{.Status}}
`

const healthComponentTemplate = `{{.Icon}} {{.Component}}: {{.Status}}{{if .ResponseTime}} ({{.ResponseTime}}){{end}}{{if .Message}} - {{.Message}}{{end}}
{{if .ShowDetails}}{{range $key, $value := .Details}}  {{$key}}: {{$value}}
{{end}}{{end}}`

const healthQuietTemplate = `{{.Icon}} {{.Component}}: {{.Status}}
`

const healthMonitorStartTemplate = `🔍 Starting health monitoring (interval: {{.Interval}}, timeout: {{.Timeout}})
Press Ctrl+C to stop...
`

var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check SPIRE infrastructure health",
	Long: `Check the health of SPIRE server and agent components using their built-in HTTP health endpoints.

This command leverages SPIRE's native health check capabilities (/live and /ready endpoints)
rather than implementing custom health checks from scratch. It supports both liveness
checks (process running) and readiness checks (ready to serve requests).

Examples:
  # Check health with configuration file
  ephemos health --config config.yaml

  # Check specific component
  ephemos health --server-address localhost:8080
  ephemos health --agent-address localhost:8081

  # Continuous monitoring
  ephemos health --monitor --interval 30s

  # Output as JSON
  ephemos health --format json`,
	RunE: runHealthCheck,
}

var healthCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Perform one-time health check",
	Long:  `Perform a one-time health check of SPIRE components and exit.`,
	RunE:  runHealthCheck,
}

var healthMonitorCmd = &cobra.Command{
	Use:   "monitor",
	Short: "Continuously monitor health",
	Long: `Continuously monitor SPIRE component health at regular intervals.
Use Ctrl+C to stop monitoring.`,
	RunE: runHealthMonitor,
}

func init() {
	// Health check flags
	healthCmd.Flags().String("config", "", "Path to configuration file")
	healthCmd.Flags().String("server-address", "", "SPIRE server health endpoint address (e.g., localhost:8080)")
	healthCmd.Flags().String("agent-address", "", "SPIRE agent health endpoint address (e.g., localhost:8081)")
	healthCmd.Flags().String("server-live-path", "/live", "SPIRE server liveness endpoint path")
	healthCmd.Flags().String("server-ready-path", "/ready", "SPIRE server readiness endpoint path")
	healthCmd.Flags().String("agent-live-path", "/live", "SPIRE agent liveness endpoint path")
	healthCmd.Flags().String("agent-ready-path", "/ready", "SPIRE agent readiness endpoint path")
	healthCmd.Flags().Bool("https", false, "Use HTTPS for health check requests")
	healthCmd.Flags().Duration("check-timeout", 10*time.Second, "Timeout for individual health checks")
	healthCmd.Flags().Bool("monitor", false, "Continuously monitor health")
	healthCmd.Flags().Duration("interval", 30*time.Second, "Monitoring interval")
	healthCmd.Flags().Bool("verbose", false, "Show detailed health information")

	// Sub-commands
	healthCmd.AddCommand(healthCheckCmd)
	healthCmd.AddCommand(healthMonitorCmd)

	// Monitor-specific flags
	healthMonitorCmd.Flags().Duration("interval", 30*time.Second, "Monitoring interval")
	healthMonitorCmd.Flags().String("config", "", "Path to configuration file")
	healthMonitorCmd.Flags().String("server-address", "", "SPIRE server health endpoint address")
	healthMonitorCmd.Flags().String("agent-address", "", "SPIRE agent health endpoint address")
	healthMonitorCmd.Flags().Duration("check-timeout", 10*time.Second, "Timeout for individual health checks")
	healthMonitorCmd.Flags().Bool("verbose", false, "Show detailed health information")

	// Check-specific flags (inherit from parent)
	healthCheckCmd.Flags().String("config", "", "Path to configuration file")
	healthCheckCmd.Flags().String("server-address", "", "SPIRE server health endpoint address")
	healthCheckCmd.Flags().String("agent-address", "", "SPIRE agent health endpoint address")
	healthCheckCmd.Flags().Duration("check-timeout", 10*time.Second, "Timeout for individual health checks")
	healthCheckCmd.Flags().Bool("verbose", false, "Show detailed health information")
}

func runHealthCheck(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	_ = ctx // Used below

	// Check if monitor flag is set
	monitor, _ := cmd.Flags().GetBool("monitor")
	if monitor {
		return runHealthMonitor(cmd, args)
	}

	config, err := buildHealthConfig(cmd)
	if err != nil {
		return fmt.Errorf("failed to build health config: %w", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelWarn, // Quiet by default for CLI
	}))

	// Create health monitor service
	monitor_service, err := services.NewHealthMonitorService(config, logger)
	if err != nil {
		return fmt.Errorf("failed to create health monitor: %w", err)
	}
	defer monitor_service.Close()

	// Register health checkers based on configuration
	if err := registerHealthCheckers(monitor_service, config); err != nil {
		return fmt.Errorf("failed to register health checkers: %w", err)
	}

	// Perform health check
	results, err := monitor_service.CheckAll(ctx)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	// Output results
	return outputHealthResults(cmd, results, monitor_service.GetOverallHealth())
}

func runHealthMonitor(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	config, err := buildHealthConfig(cmd)
	if err != nil {
		return fmt.Errorf("failed to build health config: %w", err)
	}

	// Enable monitoring
	config.Enabled = true

	// Get interval from flags
	interval, _ := cmd.Flags().GetDuration("interval")
	if interval > 0 {
		config.Interval = interval
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Create health monitor service
	monitor_service, err := services.NewHealthMonitorService(config, logger)
	if err != nil {
		return fmt.Errorf("failed to create health monitor: %w", err)
	}
	defer monitor_service.Close()

	// Register log reporter for monitoring output
	logReporter := health.NewLogHealthReporter(logger)
	if err := monitor_service.RegisterReporter(logReporter); err != nil {
		return fmt.Errorf("failed to register log reporter: %w", err)
	}

	// Register health checkers
	if err := registerHealthCheckers(monitor_service, config); err != nil {
		return fmt.Errorf("failed to register health checkers: %w", err)
	}

	fmt.Printf("🔍 Starting health monitoring (interval: %v, timeout: %v)\n",
		config.Interval, config.Timeout)
	fmt.Println("Press Ctrl+C to stop...")

	// Start monitoring
	if err := monitor_service.StartMonitoring(ctx); err != nil {
		return fmt.Errorf("failed to start monitoring: %w", err)
	}

	// Wait for context cancellation (Ctrl+C)
	<-ctx.Done()

	const stopTemplate = `
🛑 Health monitoring stopped`
	fmt.Println(stopTemplate)
	return nil
}

func buildHealthConfig(cmd *cobra.Command) (*ports.HealthConfig, error) {
	config := &ports.HealthConfig{
		Enabled: true,
	}

	// Get timeout
	if timeout, _ := cmd.Flags().GetDuration("check-timeout"); timeout > 0 {
		config.Timeout = timeout
	} else {
		config.Timeout = 10 * time.Second
	}

	// Get interval (for monitoring)
	if interval, _ := cmd.Flags().GetDuration("interval"); interval > 0 {
		config.Interval = interval
	} else {
		config.Interval = 30 * time.Second
	}

	// SPIRE server configuration
	if serverAddr, _ := cmd.Flags().GetString("server-address"); serverAddr != "" {
		useHTTPS, _ := cmd.Flags().GetBool("https")
		livePath, _ := cmd.Flags().GetString("server-live-path")
		readyPath, _ := cmd.Flags().GetString("server-ready-path")

		config.Server = &ports.SpireServerHealthConfig{
			Address:   serverAddr,
			LivePath:  livePath,
			ReadyPath: readyPath,
			UseHTTPS:  useHTTPS,
		}
	}

	// SPIRE agent configuration
	if agentAddr, _ := cmd.Flags().GetString("agent-address"); agentAddr != "" {
		useHTTPS, _ := cmd.Flags().GetBool("https")
		livePath, _ := cmd.Flags().GetString("agent-live-path")
		readyPath, _ := cmd.Flags().GetString("agent-ready-path")

		config.Agent = &ports.SpireAgentHealthConfig{
			Address:   agentAddr,
			LivePath:  livePath,
			ReadyPath: readyPath,
			UseHTTPS:  useHTTPS,
		}
	}

	// Try to load from config file if no addresses specified
	if config.Server == nil && config.Agent == nil {
		if configPath, _ := cmd.Flags().GetString("config"); configPath != "" {
			// TODO: Load health config from file
			// For now, set default addresses
		}

		// Set default addresses if none specified
		if config.Server == nil && config.Agent == nil {
			config.Server = &ports.SpireServerHealthConfig{
				Address:   "localhost:8080",
				LivePath:  "/live",
				ReadyPath: "/ready",
				UseHTTPS:  false,
			}
			config.Agent = &ports.SpireAgentHealthConfig{
				Address:   "localhost:8081",
				LivePath:  "/live",
				ReadyPath: "/ready",
				UseHTTPS:  false,
			}
		}
	}

	return config, nil
}

func registerHealthCheckers(monitor *services.HealthMonitorService, config *ports.HealthConfig) error {
	// Register SPIRE server health checker
	if config.Server != nil {
		serverChecker, err := health.NewSpireHealthClient("spire-server", config)
		if err != nil {
			return fmt.Errorf("failed to create server health checker: %w", err)
		}
		if err := monitor.RegisterChecker(serverChecker); err != nil {
			return fmt.Errorf("failed to register server health checker: %w", err)
		}
	}

	// Register SPIRE agent health checker
	if config.Agent != nil {
		agentChecker, err := health.NewSpireHealthClient("spire-agent", config)
		if err != nil {
			return fmt.Errorf("failed to create agent health checker: %w", err)
		}
		if err := monitor.RegisterChecker(agentChecker); err != nil {
			return fmt.Errorf("failed to register agent health checker: %w", err)
		}
	}

	return nil
}

func outputHealthResults(cmd *cobra.Command, results map[string]*ports.HealthResult, overallHealth ports.HealthStatus) error {
	format, _ := cmd.Flags().GetString("format")
	verbose, _ := cmd.Flags().GetBool("verbose")
	quiet, _ := cmd.Flags().GetBool("quiet")

	switch strings.ToLower(format) {
	case "json":
		return outputJSON(results, overallHealth)
	default:
		return outputText(cmd, results, overallHealth, verbose, quiet)
	}
}

func outputJSON(results map[string]*ports.HealthResult, overallHealth ports.HealthStatus) error {
	output := struct {
		OverallHealth string                         `json:"overall_health"`
		Components    map[string]*ports.HealthResult `json:"components"`
		Timestamp     time.Time                      `json:"timestamp"`
	}{
		OverallHealth: string(overallHealth),
		Components:    results,
		Timestamp:     time.Now(),
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

func outputText(cmd *cobra.Command, results map[string]*ports.HealthResult, overallHealth ports.HealthStatus, verbose, quiet bool) error {
	noEmoji, _ := cmd.Flags().GetBool("no-emoji")

	if !quiet {
		// Overall status using template
		overallData := struct {
			Icon   string
			Status string
		}{
			Icon:   getStatusIcon(overallHealth),
			Status: strings.ToUpper(string(overallHealth)),
		}

		tmplText := healthOverallTemplate
		if noEmoji {
			tmplText = replaceHealthEmojis(tmplText)
			overallData.Icon = replaceHealthEmojis(overallData.Icon)
		}

		tmpl := template.Must(template.New("overall").Parse(tmplText))
		if err := tmpl.Execute(os.Stdout, overallData); err != nil {
			return fmt.Errorf("failed to render overall health: %w", err)
		}
		fmt.Println() // Add spacing
	}

	// Component details using templates
	for component, result := range results {
		if result == nil {
			continue
		}

		componentData := struct {
			Icon         string
			Component    string
			Status       string
			ResponseTime time.Duration
			Message      string
			ShowDetails  bool
			Details      map[string]interface{}
		}{
			Icon:         getStatusIcon(result.Status),
			Component:    component,
			Status:       strings.ToUpper(string(result.Status)),
			ResponseTime: result.ResponseTime,
			Message:      result.Message,
			ShowDetails:  verbose && len(result.Details) > 0,
			Details:      result.Details,
		}

		var tmplText string
		if quiet {
			tmplText = healthQuietTemplate
		} else {
			tmplText = healthComponentTemplate
		}

		if noEmoji {
			tmplText = replaceHealthEmojis(tmplText)
			componentData.Icon = replaceHealthEmojis(componentData.Icon)
		}

		tmpl := template.Must(template.New("component").Parse(tmplText))
		if err := tmpl.Execute(os.Stdout, componentData); err != nil {
			return fmt.Errorf("failed to render component %s: %w", component, err)
		}
	}

	// Set exit code based on overall health
	if overallHealth != ports.HealthStatusHealthy {
		return fmt.Errorf("health check failed: overall status is %s", overallHealth)
	}

	return nil
}

func getStatusIcon(status ports.HealthStatus) string {
	switch status {
	case ports.HealthStatusHealthy:
		return "✅"
	case ports.HealthStatusUnhealthy:
		return "❌"
	case ports.HealthStatusUnknown:
		return "❓"
	default:
		return "⚠️"
	}
}

// replaceHealthEmojis replaces emojis with text equivalents for --no-emoji flag
func replaceHealthEmojis(text string) string {
	replacements := map[string]string{
		"✅":  "[HEALTHY]",
		"❌":  "[UNHEALTHY]",
		"❓":  "[UNKNOWN]",
		"⚠️": "[WARNING]",
		"🔍":  "[MONITOR]",
		"🛑":  "[STOP]",
	}

	result := text
	for emoji, replacement := range replacements {
		result = strings.ReplaceAll(result, emoji, replacement)
	}
	return result
}
