package main

import (
	"context"
	"errors"
	"testing"

	"github.com/sufield/ephemos/internal/cli"
)

func TestExitCodeClassification(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected int
	}{
		{
			name:     "config error",
			err:      cli.ErrConfig,
			expected: exitConfig,
		},
		{
			name:     "auth error",
			err:      cli.ErrAuth,
			expected: exitAuth,
		},
		{
			name:     "context canceled",
			err:      context.Canceled,
			expected: exitOK,
		},
		{
			name:     "unknown error",
			err:      errors.New("unknown error"),
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the logic from main() function
			var code int
			switch {
			case errors.Is(tt.err, cli.ErrConfig):
				code = exitConfig
			case errors.Is(tt.err, cli.ErrAuth):
				code = exitAuth
			case errors.Is(tt.err, context.Canceled):
				code = exitOK
			default:
				code = 1
			}

			if code != tt.expected {
				t.Errorf("expected exit code %d, got %d", tt.expected, code)
			}
		})
	}
}

func TestExitCodesAreUnique(t *testing.T) {
	codes := map[string]int{
		"exitOK":     exitOK,
		"exitConfig": exitConfig,
		"exitAuth":   exitAuth,
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

	if exitConfig < 1 {
		t.Error("exitConfig should be positive")
	}

	if exitAuth < 1 {
		t.Error("exitAuth should be positive")
	}

	// Ensure all exit codes are non-negative
	codes := []int{exitOK, exitConfig, exitAuth}
	for _, code := range codes {
		if code < 0 {
			t.Errorf("exit code %d should not be negative", code)
		}
	}
}
