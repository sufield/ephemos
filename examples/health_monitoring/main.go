// Package main demonstrates how to use Ephemos health monitoring
// to check SPIRE infrastructure health using built-in HTTP endpoints.
//
// This example shows:
// - How to configure SPIRE health checks
// - Using the health monitoring service
// - Both one-time checks and continuous monitoring
// - Custom health reporters
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sufield/ephemos/internal/adapters/secondary/health"
	"github.com/sufield/ephemos/internal/core/ports"
	"github.com/sufield/ephemos/internal/core/services"
)

func main() {
	// Create logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Example 1: Simple one-time health check
	fmt.Println("=== Example 1: One-time Health Check ===")
	if err := runOneTimeHealthCheck(logger); err != nil {
		logger.Error("One-time health check failed", "error", err)
	}

	fmt.Println("\n=== Example 2: Continuous Health Monitoring ===")
	// Example 2: Continuous health monitoring
	if err := runContinuousMonitoring(logger); err != nil {
		logger.Error("Continuous monitoring failed", "error", err)
	}
}

func runOneTimeHealthCheck(logger *slog.Logger) error {
	// Configure health checks for SPIRE server and agent
	config := &ports.HealthConfig{
		Enabled: true,
		Timeout: 10 * time.Second,
		Server: &ports.SpireServerHealthConfig{
			Address:   "localhost:8080",
			LivePath:  "/live",
			ReadyPath: "/ready",
			UseHTTPS:  false,
		},
		Agent: &ports.SpireAgentHealthConfig{
			Address:   "localhost:8081",
			LivePath:  "/live",
			ReadyPath: "/ready",
			UseHTTPS:  false,
		},
	}

	// Create health monitor service
	monitor, err := services.NewHealthMonitorService(config, logger)
	if err != nil {
		return fmt.Errorf("failed to create health monitor: %w", err)
	}
	defer monitor.Close()

	// Create and register SPIRE server health checker
	serverChecker, err := health.NewSpireHealthClient("spire-server", config)
	if err != nil {
		return fmt.Errorf("failed to create server health checker: %w", err)
	}

	if err := monitor.RegisterChecker(serverChecker); err != nil {
		return fmt.Errorf("failed to register server health checker: %w", err)
	}

	// Create and register SPIRE agent health checker
	agentChecker, err := health.NewSpireHealthClient("spire-agent", config)
	if err != nil {
		return fmt.Errorf("failed to create agent health checker: %w", err)
	}

	if err := monitor.RegisterChecker(agentChecker); err != nil {
		return fmt.Errorf("failed to register agent health checker: %w", err)
	}

	// Register log reporter
	logReporter := health.NewLogHealthReporter(logger)
	if err := monitor.RegisterReporter(logReporter); err != nil {
		return fmt.Errorf("failed to register log reporter: %w", err)
	}

	// Perform health check
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	results, err := monitor.CheckAll(ctx)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	// Display results
	fmt.Printf("Health check completed. Overall status: %s\n", monitor.GetOverallHealth())
	for component, result := range results {
		fmt.Printf("  %s: %s (%v)\n", component, result.Status, result.ResponseTime)
		if result.Message != "" {
			fmt.Printf("    Message: %s\n", result.Message)
		}
	}

	return nil
}

func runContinuousMonitoring(logger *slog.Logger) error {
	// Configure continuous monitoring
	config := &ports.HealthConfig{
		Enabled:  true,
		Timeout:  10 * time.Second,
		Interval: 30 * time.Second, // Check every 30 seconds
		Server: &ports.SpireServerHealthConfig{
			Address:   "localhost:8080",
			LivePath:  "/live",
			ReadyPath: "/ready",
			UseHTTPS:  false,
		},
		Agent: &ports.SpireAgentHealthConfig{
			Address:   "localhost:8081",
			LivePath:  "/live",
			ReadyPath: "/ready",
			UseHTTPS:  false,
		},
	}

	// Create health monitor service
	monitor, err := services.NewHealthMonitorService(config, logger)
	if err != nil {
		return fmt.Errorf("failed to create health monitor: %w", err)
	}
	defer monitor.Close()

	// Register health checkers
	if err := registerHealthCheckers(monitor, config); err != nil {
		return fmt.Errorf("failed to register health checkers: %w", err)
	}

	// Register reporters
	logReporter := health.NewLogHealthReporter(logger)
	if err := monitor.RegisterReporter(logReporter); err != nil {
		return fmt.Errorf("failed to register log reporter: %w", err)
	}

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Start monitoring
	logger.Info("Starting continuous health monitoring", 
		"interval", config.Interval,
		"timeout", config.Timeout)

	if err := monitor.StartMonitoring(ctx); err != nil {
		return fmt.Errorf("failed to start monitoring: %w", err)
	}

	// Wait for signal
	<-sigCh
	logger.Info("Received shutdown signal, stopping health monitoring")

	// Stop monitoring
	if err := monitor.StopMonitoring(); err != nil {
		logger.Error("Failed to stop monitoring gracefully", "error", err)
	}

	return nil
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