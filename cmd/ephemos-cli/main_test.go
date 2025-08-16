package main

import (
	"errors"
	"testing"

	"github.com/sufield/ephemos/internal/cli"
)

func TestClassifyExitCode(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected int
	}{
		{
			name:     "usage error",
			err:      cli.ErrUsage,
			expected: exitUsage,
		},
		{
			name:     "config error",
			err:      cli.ErrConfig,
			expected: exitConfig,
		},
		{
			name:     "auth error",
			err:      cli.ErrAuth,
			expected: exitRuntime,
		},
		{
			name:     "runtime error",
			err:      cli.ErrRuntime,
			expected: exitRuntime,
		},
		{
			name:     "internal error",
			err:      cli.ErrInternal,
			expected: exitInternal,
		},
		{
			name:     "unknown error",
			err:      errors.New("unknown error"),
			expected: exitRuntime,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := classifyExitCode(tt.err)
			if code != tt.expected {
				t.Errorf("expected exit code %d, got %d", tt.expected, code)
			}
		})
	}
}

func TestExitCodesAreUnique(t *testing.T) {
	codes := map[string]int{
		"exitOK":       exitOK,
		"exitUsage":    exitUsage,
		"exitConfig":   exitConfig,
		"exitRuntime":  exitRuntime,
		"exitInternal": exitInternal,
	}

	seen := make(map[int]string)
	for name, code := range codes {
		if existing, ok := seen[code]; ok {
			t.Errorf("duplicate exit code %d for %s and %s", code, name, existing)
		}
		seen[code] = name
	}
}

func TestExitCodeValues(t *testing.T) {
	// Verify specific values follow conventions
	if exitOK != 0 {
		t.Error("exitOK should be 0")
	}
	
	if exitUsage != 2 {
		t.Error("exitUsage should be 2 (following standard conventions)")
	}

	// Ensure all exit codes are positive
	codes := []int{exitOK, exitUsage, exitConfig, exitRuntime, exitInternal}
	for _, code := range codes {
		if code < 0 {
			t.Errorf("exit code %d should not be negative", code)
		}
	}
}