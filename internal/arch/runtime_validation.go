// Package arch provides runtime architectural boundary validation.
// This complements the compile-time tests with runtime checks.
package arch

import (
	"fmt"
	"runtime"
	"strings"
)

// ArchValidator provides runtime validation of architectural boundaries.
type ArchValidator struct {
	violations []string
	enabled    bool
}

// NewValidator creates a new architectural validator.
// In production builds, validation can be disabled for performance.
func NewValidator(enabled bool) *ArchValidator {
	return &ArchValidator{
		enabled: enabled,
	}
}

// ValidateCall checks if the current call stack violates architectural boundaries.
// This is meant to be called at critical boundary points.
func (v *ArchValidator) ValidateCall(operation string) error {
	if !v.enabled {
		return nil
	}

	pc := make([]uintptr, 32)
	n := runtime.Callers(2, pc)
	frames := runtime.CallersFrames(pc[:n])

	var callStack []string
	for {
		frame, more := frames.Next()
		if !more {
			break
		}

		// Filter out runtime and testing frames
		if !strings.Contains(frame.File, "runtime/") &&
			!strings.Contains(frame.File, "testing/") {
			callStack = append(callStack, frame.Function)
		}
	}

	// Check for architectural violations in the call stack
	violation := v.checkCallStackViolation(callStack, operation)
	if violation != "" {
		v.violations = append(v.violations, violation)
		return fmt.Errorf("architectural boundary violation: %s", violation)
	}

	return nil
}

// checkCallStackViolation analyzes the call stack for boundary violations.
func (v *ArchValidator) checkCallStackViolation(callStack []string, operation string) string {
	// Check for direct adapter-to-adapter calls
	for i, caller := range callStack {
		if strings.Contains(caller, "/internal/adapters/") {
			for j := i + 1; j < len(callStack); j++ {
				callee := callStack[j]
				if strings.Contains(callee, "/internal/adapters/") {
					if v.isDifferentAdapterType(caller, callee) {
						return fmt.Sprintf("adapter %s directly calls adapter %s during %s",
							v.extractAdapterType(caller), v.extractAdapterType(callee), operation)
					}
				}
			}
		}
	}

	// Check for domain calling adapters directly
	for i, caller := range callStack {
		if strings.Contains(caller, "/internal/core/domain/") {
			for j := i + 1; j < len(callStack); j++ {
				callee := callStack[j]
				if strings.Contains(callee, "/internal/adapters/") {
					return fmt.Sprintf("domain %s directly calls adapter %s during %s",
						caller, v.extractAdapterType(callee), operation)
				}
			}
		}
	}

	return ""
}

// isDifferentAdapterType checks if two function names are from different adapter types.
func (v *ArchValidator) isDifferentAdapterType(caller, callee string) bool {
	callerType := v.extractAdapterType(caller)
	calleeType := v.extractAdapterType(callee)
	return callerType != calleeType && callerType != "" && calleeType != ""
}

// extractAdapterType extracts the adapter type from a function name.
func (v *ArchValidator) extractAdapterType(funcName string) string {
	if !strings.Contains(funcName, "/internal/adapters/") {
		return ""
	}

	// Extract path component after /internal/adapters/
	parts := strings.Split(funcName, "/internal/adapters/")
	if len(parts) < 2 {
		return ""
	}

	adapterPath := parts[1]
	pathParts := strings.Split(adapterPath, "/")
	if len(pathParts) > 0 {
		return pathParts[0] // primary, secondary, grpc, http, etc.
	}

	return ""
}

// GetViolations returns all recorded violations.
func (v *ArchValidator) GetViolations() []string {
	return append([]string(nil), v.violations...) // Return copy
}

// ClearViolations clears all recorded violations.
func (v *ArchValidator) ClearViolations() {
	v.violations = nil
}

// Global validator instance (can be disabled in production)
var globalValidator = NewValidator(true)

// SetGlobalValidationEnabled enables or disables global validation.
func SetGlobalValidationEnabled(enabled bool) {
	globalValidator.enabled = enabled
}

// ValidateBoundary is a convenience function for validating architectural boundaries.
// Call this at critical points where adapters interact with core or each other.
func ValidateBoundary(operation string) error {
	return globalValidator.ValidateCall(operation)
}

// GetGlobalViolations returns violations from the global validator.
func GetGlobalViolations() []string {
	return globalValidator.GetViolations()
}

// ClearGlobalViolations clears violations from the global validator.
func ClearGlobalViolations() {
	globalValidator.ClearViolations()
}
