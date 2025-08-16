package arch

import (
	"strings"
	"sync"
	"testing"
	"time"
)

// Test adapter type extraction cases
func TestValidator_ExtractAdapterType(t *testing.T) {
	t.Parallel()
	
	v := NewValidator(true)
	cases := map[string]string{
		"github.com/x/y/internal/adapters/grpc.(*Server).Serve":       "grpc",
		"github.com/x/y/internal/adapters/grpc/client.(*C).Do":        "grpc",
		"github.com/x/y/internal/adapters/http.Handler.ServeHTTP":      "http",
		"github.com/x/y/internal/adapters/redis.(*Client).Get":        "redis",
		"github.com/x/y/internal/adapters/shared/util.(*Helper).Do":   "shared",
		"github.com/x/y/internal/adapters/database/sql.(*DB).Query":   "database",
		"github.com/x/y/internal/core/domain.(*Entity).Method":        "", // not adapter
		"github.com/x/y/other/package.Function":                       "", // not adapter
	}

	for input, expected := range cases {
		result := v.extractAdapterType(input)
		if result != expected {
			t.Errorf("extractAdapterType(%q) = %q, want %q", input, result, expected)
		}
	}
}

// Test adjacency detection logic
func TestValidator_CheckCallStackViolation(t *testing.T) {
	t.Parallel()
	
	v := NewValidator(true)
	
	tests := []struct {
		name      string
		stack     []string
		operation string
		wantError string
	}{
		{
			name: "domain directly calls adapter",
			stack: []string{
				"github.com/x/y/internal/core/domain.(*Entity).Method",
				"github.com/x/y/internal/adapters/grpc.(*Server).Handle",
			},
			operation: "entity processing",
			wantError: "domain github.com/x/y/internal/core/domain.(*Entity).Method directly calls adapter grpc during entity processing",
		},
		{
			name: "adapter directly calls different adapter",
			stack: []string{
				"github.com/x/y/internal/adapters/http.(*Handler).ServeHTTP",
				"github.com/x/y/internal/adapters/grpc.(*Client).Call",
			},
			operation: "request handling",
			wantError: "adapter http directly calls adapter grpc during request handling",
		},
		{
			name: "same adapter calls itself",
			stack: []string{
				"github.com/x/y/internal/adapters/grpc.(*Server).Handle",
				"github.com/x/y/internal/adapters/grpc.(*Client).Call",
			},
			operation: "grpc processing",
			wantError: "", // same adapter type, should be allowed
		},
		{
			name: "allowed cross-adapter call",
			stack: []string{
				"github.com/x/y/internal/adapters/http.(*Handler).ServeHTTP",
				"github.com/x/y/internal/adapters/shared.(*Util).Helper",
			},
			operation: "request processing",
			wantError: "", // will be allowed after setting up allowlist
		},
		{
			name: "indirect call through service",
			stack: []string{
				"github.com/x/y/internal/adapters/http.(*Handler).ServeHTTP",
				"github.com/x/y/internal/core/services.(*Service).Process",
				"github.com/x/y/internal/adapters/grpc.(*Client).Call",
			},
			operation: "request processing",
			wantError: "", // indirect, not adjacent
		},
		{
			name: "no violations",
			stack: []string{
				"github.com/x/y/internal/core/services.(*Service).Process",
				"github.com/x/y/internal/core/domain.(*Entity).Method",
			},
			operation: "business logic",
			wantError: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up allowlist for specific test
			if tt.name == "allowed cross-adapter call" {
				v.Allow("http", "shared")
			}
			
			result := v.checkCallStackViolation(tt.stack, tt.operation)
			if result != tt.wantError {
				t.Errorf("checkCallStackViolation() = %q, want %q", result, tt.wantError)
			}
		})
	}
}

// Test helper functions
func TestIsAdapterFunc(t *testing.T) {
	t.Parallel()
	
	tests := []struct {
		fn   string
		want bool
	}{
		{"github.com/x/y/internal/adapters/grpc.(*Server).Serve", true},
		{"github.com/x/y/internal/adapters/http/handler.ServeHTTP", true},
		{"github.com/x/y/internal/core/domain.(*Entity).Method", false},
		{"github.com/x/y/internal/core/services.(*Service).Process", false},
		{"github.com/x/y/pkg/ephemos.(*Client).Connect", false},
	}

	for _, tt := range tests {
		if got := isAdapterFunc(tt.fn); got != tt.want {
			t.Errorf("isAdapterFunc(%q) = %v, want %v", tt.fn, got, tt.want)
		}
	}
}

func TestIsDomainFunc(t *testing.T) {
	t.Parallel()
	
	tests := []struct {
		fn   string
		want bool
	}{
		{"github.com/x/y/internal/core/domain.(*Entity).Method", true},
		{"github.com/x/y/internal/core/domain/entities.(*User).Create", true},
		{"github.com/x/y/internal/core/services.(*Service).Process", false},
		{"github.com/x/y/internal/adapters/grpc.(*Server).Serve", false},
		{"github.com/x/y/pkg/ephemos.(*Client).Connect", false},
	}

	for _, tt := range tests {
		if got := isDomainFunc(tt.fn); got != tt.want {
			t.Errorf("isDomainFunc(%q) = %v, want %v", tt.fn, got, tt.want)
		}
	}
}

// Test thread safety with concurrent operations
func TestValidator_ConcurrentAccess(t *testing.T) {
	t.Parallel()
	
	v := NewValidator(true)
	
	// Number of goroutines for the race test
	const numGoroutines = 100
	const numOperations = 10
	
	var wg sync.WaitGroup
	
	// Simulate concurrent validation calls
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				// Toggle enabled state
				v.SetEnabled(j%2 == 0)
				
				// Check enabled state
				_ = v.IsEnabled()
				
				// Add violations
				v.addViolation("test violation")
				
				// Get violations
				_ = v.GetViolations()
				
				// Clear violations
				if j%5 == 0 {
					v.ClearViolations()
				}
				
				// Add allowlist entries
				v.Allow("test1", "test2")
			}
		}(i)
	}
	
	// Wait for all goroutines to complete
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		// Success - no deadlocks or races
	case <-time.After(5 * time.Second):
		t.Fatal("Test timed out - possible deadlock")
	}
}

// Test global API functions
func TestGlobalAPI(t *testing.T) {
	t.Parallel()
	
	// Save original state
	originalEnabled := IsGlobalValidationEnabled()
	originalViolations := GetGlobalViolations()
	defer func() {
		SetGlobalValidationEnabled(originalEnabled)
		ClearGlobalViolations()
		// Restore violations if any
		for _, v := range originalViolations {
			globalValidator.addViolation(v)
		}
	}()
	
	// Test enable/disable
	SetGlobalValidationEnabled(false)
	if IsGlobalValidationEnabled() {
		t.Error("Expected validation to be disabled")
	}
	
	SetGlobalValidationEnabled(true)
	if !IsGlobalValidationEnabled() {
		t.Error("Expected validation to be enabled")
	}
	
	// Clear any existing violations
	ClearGlobalViolations()
	
	// Test violations management
	globalValidator.addViolation("test violation 1")
	globalValidator.addViolation("test violation 2")
	
	violations := GetGlobalViolations()
	if len(violations) != 2 {
		t.Errorf("Expected 2 violations, got %d", len(violations))
	}
	
	expectedViolations := []string{"test violation 1", "test violation 2"}
	for i, expected := range expectedViolations {
		if violations[i] != expected {
			t.Errorf("Expected violation %d to be %q, got %q", i, expected, violations[i])
		}
	}
	
	// Test clear
	ClearGlobalViolations()
	violations = GetGlobalViolations()
	if len(violations) != 0 {
		t.Errorf("Expected 0 violations after clear, got %d", len(violations))
	}
	
	// Test allowlist
	AllowGlobalCrossing("http", "shared")
	if !globalValidator.allowed("http", "shared") {
		t.Error("Expected http->shared to be allowed")
	}
}

// Test validation with actual call stack
func TestValidator_ValidateCall(t *testing.T) {
	t.Parallel()
	
	v := NewValidator(true)
	
	// This will capture the actual call stack
	err := v.ValidateCall("test operation")
	
	// Should not error since this test doesn't violate boundaries
	if err != nil {
		// If it does error, check if it's expected based on where this test runs
		if !strings.Contains(err.Error(), "architectural boundary violation") {
			t.Errorf("Unexpected error: %v", err)
		}
	}
	
	// Test with validation disabled
	v.SetEnabled(false)
	err = v.ValidateCall("test operation")
	if err != nil {
		t.Errorf("Expected no error when validation disabled, got: %v", err)
	}
}

// Test allowlist functionality
func TestValidator_Allowlist(t *testing.T) {
	t.Parallel()
	
	v := NewValidator(true)
	
	// Initially not allowed
	if v.allowed("http", "shared") {
		t.Error("Expected http->shared to not be allowed initially")
	}
	
	// Add to allowlist
	v.Allow("http", "shared")
	if !v.allowed("http", "shared") {
		t.Error("Expected http->shared to be allowed after Allow()")
	}
	
	// Should not affect other combinations
	if v.allowed("grpc", "shared") {
		t.Error("Expected grpc->shared to still not be allowed")
	}
	
	if v.allowed("http", "grpc") {
		t.Error("Expected http->grpc to still not be allowed")
	}
}