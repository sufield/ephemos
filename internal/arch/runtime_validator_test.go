package arch

import (
	"sync"
	"testing"
)

// Test helper functions with proper function signatures
func TestIsAdapterFunc(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		fn   string
		want bool
	}{
		{"grpc adapter", "github.com/sufield/ephemos/internal/adapters/grpc.(*Server).Serve", true},
		{"http adapter with subpath", "github.com/sufield/ephemos/internal/adapters/http/handler.ServeHTTP", true},
		{"domain function", "github.com/sufield/ephemos/internal/core/domain.(*Entity).Method", false},
		{"service function", "github.com/sufield/ephemos/internal/core/services.(*Service).Process", false},
		{"public API", "github.com/sufield/ephemos/pkg/ephemos.(*Client).Connect", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := isAdapterFunc(tt.fn); got != tt.want {
				t.Fatalf("isAdapterFunc(%q) = %v, want %v", tt.fn, got, tt.want)
			}
		})
	}
}

func TestIsDomainFunc(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		fn   string
		want bool
	}{
		{"domain entity", "github.com/sufield/ephemos/internal/core/domain.(*Entity).Method", true},
		{"domain subpackage", "github.com/sufield/ephemos/internal/core/domain/entities.(*User).Create", true},
		{"service function", "github.com/sufield/ephemos/internal/core/services.(*Service).Process", false},
		{"adapter function", "github.com/sufield/ephemos/internal/adapters/grpc.(*Server).Serve", false},
		{"public API", "github.com/sufield/ephemos/pkg/ephemos.(*Client).Connect", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := isDomainFunc(tt.fn); got != tt.want {
				t.Fatalf("isDomainFunc(%q) = %v, want %v", tt.fn, got, tt.want)
			}
		})
	}
}

// Test ValidateCall behavior without asserting specific violations
func TestValidator_ValidateCall_NoPanic(t *testing.T) {
	t.Parallel()

	enabled := NewValidator(true)
	disabled := NewValidator(false)

	// Disabled validator should never fail
	if err := disabled.ValidateCall("test operation"); err != nil {
		t.Fatalf("disabled validator should never fail, got %v", err)
	}

	// Enabled may or may not flag something depending on stack; just ensure no panic
	_ = enabled.ValidateCall("test operation")
}

// Test global API with proper state management
func TestGlobalAPI_Behavior(t *testing.T) {
	t.Parallel()

	// Save and restore original state
	originalEnabled := IsGlobalValidationEnabled()
	defer func() {
		SetGlobalValidationEnabled(originalEnabled)
		ClearGlobalViolations()
	}()

	// Test enable/disable atomicity
	SetGlobalValidationEnabled(false)
	if IsGlobalValidationEnabled() {
		t.Fatalf("Expected validation to be disabled")
	}

	SetGlobalValidationEnabled(true)
	if !IsGlobalValidationEnabled() {
		t.Fatalf("Expected validation to be enabled")
	}

	// Test violations management
	ClearGlobalViolations()
	initialCount := len(GetGlobalViolations())
	if initialCount != 0 {
		t.Fatalf("Expected 0 violations after clear, got %d", initialCount)
	}

	// Test allowlist API
	AllowGlobalCrossing("http", "shared")
	// Can't easily test the internal state without exposing internals,
	// but this ensures the API doesn't panic
}

// Comprehensive race test for global validator
func TestGlobalValidator_NoDataRaces(t *testing.T) {
	t.Parallel()

	const workers = 8
	const iterations = 500
	var wg sync.WaitGroup
	wg.Add(workers)

	for i := 0; i < workers; i++ {
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				// Concurrent state changes
				if j%2 == 0 {
					SetGlobalValidationEnabled(true)
				} else {
					SetGlobalValidationEnabled(false)
				}

				// Concurrent validation calls
				_ = ValidateBoundary("race test operation")

				// Concurrent violation management
				_ = GetGlobalViolations()
				if j%5 == 0 {
					ClearGlobalViolations()
				}

				// Concurrent allowlist changes
				AllowGlobalCrossing("worker", "target")
			}
		}(i)
	}
	wg.Wait()
}

// Test individual validator instance race safety
func TestValidator_InstanceRaceSafety(t *testing.T) {
	t.Parallel()

	v := NewValidator(true)
	const workers = 10
	const iterations = 100
	var wg sync.WaitGroup
	wg.Add(workers)

	for i := 0; i < workers; i++ {
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				// Concurrent operations on the same validator
				v.SetEnabled(j%2 == 0)
				_ = v.IsEnabled()
				_ = v.ValidateCall("instance race test")
				_ = v.GetViolations()
				if j%10 == 0 {
					v.ClearViolations()
				}
				v.Allow("test", "target")
			}
		}(i)
	}
	wg.Wait()
}

// Benchmarks with proper allocation reporting
func BenchmarkValidationOverhead(b *testing.B) {
	v := NewValidator(true)
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = v.ValidateCall("benchmark operation")
	}
}

func BenchmarkValidationDisabled(b *testing.B) {
	v := NewValidator(false)
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = v.ValidateCall("benchmark operation")
	}
}

func BenchmarkGlobalValidation(b *testing.B) {
	// Save and restore state
	originalEnabled := IsGlobalValidationEnabled()
	defer SetGlobalValidationEnabled(originalEnabled)

	SetGlobalValidationEnabled(true)
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = ValidateBoundary("global benchmark")
	}
}

func BenchmarkAtomicEnabledCheck(b *testing.B) {
	v := NewValidator(true)
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = v.IsEnabled()
	}
}
