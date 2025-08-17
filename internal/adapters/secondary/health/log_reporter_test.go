package health

import (
	"bytes"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/sufield/ephemos/internal/core/ports"
)

func TestNewLogHealthReporter(t *testing.T) {
	// Test with provided logger
	logger := slog.Default()
	reporter := NewLogHealthReporter(logger)
	assert.NotNil(t, reporter)

	// Test with nil logger (should use default)
	reporter = NewLogHealthReporter(nil)
	assert.NotNil(t, reporter)
}

func TestLogHealthReporter_ReportHealth(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	
	reporter := NewLogHealthReporter(logger)

	tests := []struct {
		name           string
		result         *ports.HealthResult
		expectedLevel  string
		shouldContain  []string
	}{
		{
			name: "healthy component",
			result: &ports.HealthResult{
				Status:       ports.HealthStatusHealthy,
				Component:    "test-component",
				Message:      "All systems operational",
				CheckedAt:    time.Now(),
				ResponseTime: 100 * time.Millisecond,
			},
			expectedLevel: "INFO",
			shouldContain: []string{
				"Health check passed",
				"test-component",
				"healthy",
				"All systems operational",
			},
		},
		{
			name: "unhealthy component",
			result: &ports.HealthResult{
				Status:       ports.HealthStatusUnhealthy,
				Component:    "failing-component",
				Message:      "Service unavailable",
				CheckedAt:    time.Now(),
				ResponseTime: 500 * time.Millisecond,
			},
			expectedLevel: "WARN",
			shouldContain: []string{
				"Health check failed",
				"failing-component",
				"unhealthy",
				"Service unavailable",
			},
		},
		{
			name: "unknown status",
			result: &ports.HealthResult{
				Status:       ports.HealthStatusUnknown,
				Component:    "unknown-component",
				Message:      "Cannot determine status",
				CheckedAt:    time.Now(),
				ResponseTime: 0,
			},
			expectedLevel: "ERROR",
			shouldContain: []string{
				"Health check status unknown",
				"unknown-component",
				"unknown",
			},
		},
		{
			name: "component without message",
			result: &ports.HealthResult{
				Status:       ports.HealthStatusHealthy,
				Component:    "silent-component",
				CheckedAt:    time.Now(),
				ResponseTime: 50 * time.Millisecond,
			},
			expectedLevel: "INFO",
			shouldContain: []string{
				"Health check passed",
				"silent-component",
				"healthy",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			
			err := reporter.ReportHealth(tt.result)
			assert.NoError(t, err)

			output := buf.String()
			
			// Check log level
			assert.Contains(t, output, tt.expectedLevel)
			
			// Check all expected content
			for _, expected := range tt.shouldContain {
				assert.Contains(t, output, expected)
			}
		})
	}
}

func TestLogHealthReporter_ReportHealth_NilResult(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	reporter := NewLogHealthReporter(logger)

	err := reporter.ReportHealth(nil)
	assert.NoError(t, err)
	
	// Should not log anything for nil result
	assert.Empty(t, buf.String())
}

func TestLogHealthReporter_ReportOverallHealth(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	
	reporter := NewLogHealthReporter(logger)

	tests := []struct {
		name          string
		results       map[string]*ports.HealthResult
		expectedLevel string
		shouldContain []string
	}{
		{
			name:          "no results",
			results:       map[string]*ports.HealthResult{},
			expectedLevel: "INFO",
			shouldContain: []string{
				"No health check results available",
			},
		},
		{
			name: "all healthy",
			results: map[string]*ports.HealthResult{
				"comp1": {Status: ports.HealthStatusHealthy},
				"comp2": {Status: ports.HealthStatusHealthy},
			},
			expectedLevel: "INFO",
			shouldContain: []string{
				"Overall system health status",
				"overall_status=healthy",
				"total_components=2",
				"healthy=2",
				"unhealthy=0",
				"unknown=0",
			},
		},
		{
			name: "mixed health",
			results: map[string]*ports.HealthResult{
				"comp1": {Status: ports.HealthStatusHealthy},
				"comp2": {Status: ports.HealthStatusUnhealthy},
				"comp3": {Status: ports.HealthStatusUnknown},
			},
			expectedLevel: "WARN",
			shouldContain: []string{
				"System health degraded",
				"overall_status=unhealthy",
				"total_components=3",
				"healthy=1",
				"unhealthy=1",
				"unknown=1",
			},
		},
		{
			name: "all unhealthy",
			results: map[string]*ports.HealthResult{
				"comp1": {Status: ports.HealthStatusUnhealthy},
				"comp2": {Status: ports.HealthStatusUnhealthy},
			},
			expectedLevel: "WARN",
			shouldContain: []string{
				"System health degraded",
				"overall_status=unhealthy",
				"total_components=2",
				"healthy=0",
				"unhealthy=2",
				"unknown=0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			
			err := reporter.ReportOverallHealth(tt.results)
			assert.NoError(t, err)

			output := buf.String()
			
			// Check log level
			assert.Contains(t, output, tt.expectedLevel)
			
			// Check all expected content
			for _, expected := range tt.shouldContain {
				assert.Contains(t, output, expected)
			}
		})
	}
}

func TestLogHealthReporter_Close(t *testing.T) {
	reporter := NewLogHealthReporter(slog.Default())
	
	err := reporter.Close()
	assert.NoError(t, err)
}