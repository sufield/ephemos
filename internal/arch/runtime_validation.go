//go:build !prod_arch_off

// Package arch provides thread-safe runtime architectural boundary validation.
// This complements the compile-time tests with runtime checks.
package arch

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
)

// Validator provides thread-safe runtime validation of architectural boundaries.
type Validator struct {
	mu         sync.Mutex
	violations []string
	enabled    atomic.Bool

	// allowlist: map[fromAdapter]set[toAdapter] for legitimate crossings
	allow map[string]map[string]struct{}
}

// NewValidator creates a new validator with the specified enabled state.
func NewValidator(enabled bool) *Validator {
	v := &Validator{allow: make(map[string]map[string]struct{})}
	v.enabled.Store(enabled)
	return v
}

// SetEnabled atomically sets the validation enabled state.
func (v *Validator) SetEnabled(b bool) {
	v.enabled.Store(b)
}

// IsEnabled atomically checks if validation is enabled.
func (v *Validator) IsEnabled() bool {
	return v.enabled.Load()
}

// addViolation safely adds a violation to the list.
func (v *Validator) addViolation(s string) {
	v.mu.Lock()
	v.violations = append(v.violations, s)
	v.mu.Unlock()
}

// ValidateCall checks the current call stack for architectural boundary violations.
func (v *Validator) ValidateCall(operation string) error {
	if !v.enabled.Load() {
		return nil
	}

	// Growable buffer to capture complete stack
	pc := make([]uintptr, 32)
	n := runtime.Callers(2, pc)
	for n == len(pc) {
		pc = make([]uintptr, len(pc)*2)
		n = runtime.Callers(2, pc)
	}

	frames := runtime.CallersFrames(pc[:n])
	var callStack []string
	for {
		frame, more := frames.Next()
		if !more {
			break
		}

		fn := frame.Function // import path + function
		// Filter noisy frames by package path rather than file path
		if strings.HasPrefix(fn, "runtime.") ||
			strings.HasPrefix(fn, "testing.") ||
			strings.Contains(fn, "/internal/arch") { // Skip our own frames
			continue
		}
		callStack = append(callStack, fn)
	}

	if violation := v.checkCallStackViolation(callStack, operation); violation != "" {
		v.addViolation(violation)
		return fmt.Errorf("architectural boundary violation: %s", violation)
	}
	return nil
}

// isAdapterFunc checks if a function is from an adapter package.
func isAdapterFunc(fn string) bool {
	return strings.Contains(fn, "/internal/adapters")
}

// isDomainFunc checks if a function is from a domain package.
func isDomainFunc(fn string) bool {
	return strings.Contains(fn, "/internal/core/domain")
}

// extractAdapterType correctly extracts the adapter type from a function name.
func (v *Validator) extractAdapterType(fn string) string {
	const key = "/internal/adapters/"
	i := strings.Index(fn, key)
	if i < 0 {
		return ""
	}
	rest := fn[i+len(key):] // e.g. "grpc/client.(*X).Meth" or "grpc.(*S).Serve"

	// First path segment after adapters
	if j := strings.IndexByte(rest, '/'); j >= 0 {
		rest = rest[:j] // "grpc"
	}
	// Drop anything after the first dot: ".(*Type)..." → "grpc"
	if k := strings.IndexByte(rest, '.'); k >= 0 {
		rest = rest[:k]
	}
	return rest
}

// checkCallStackViolation examines adjacent frames for direct boundary violations.
func (v *Validator) checkCallStackViolation(stack []string, op string) string {
	// Scan adjacent pairs: stack[0] (most recent) -> stack[1] (its caller) ...
	for i := 0; i+1 < len(stack); i++ {
		caller, callee := stack[i], stack[i+1]

		// Domain -> adapter direct call
		if isDomainFunc(caller) && isAdapterFunc(callee) {
			return fmt.Sprintf("domain %s directly calls adapter %s during %s",
				caller, v.extractAdapterType(callee), op)
		}

		// Adapter -> adapter direct call across types (unless allowed)
		if isAdapterFunc(caller) && isAdapterFunc(callee) {
			a := v.extractAdapterType(caller)
			b := v.extractAdapterType(callee)
			if a != "" && b != "" && a != b && !v.allowed(a, b) {
				return fmt.Sprintf("adapter %s directly calls adapter %s during %s", a, b, op)
			}
		}
	}
	return ""
}

// allowed checks if a cross-adapter call is explicitly permitted.
func (v *Validator) allowed(from, to string) bool {
	v.mu.Lock()
	defer v.mu.Unlock()
	if m, ok := v.allow[from]; ok {
		_, ok = m[to]
		return ok
	}
	return false
}

// Allow explicitly permits calls from one adapter type to another.
// Useful for shared/base adapters or legitimate crossings.
func (v *Validator) Allow(from, to string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.allow[from] == nil {
		v.allow[from] = make(map[string]struct{})
	}
	v.allow[from][to] = struct{}{}
}

// GetViolations returns a copy of all recorded violations.
func (v *Validator) GetViolations() []string {
	v.mu.Lock()
	cp := append([]string(nil), v.violations...)
	v.mu.Unlock()
	return cp
}

// ClearViolations removes all recorded violations.
func (v *Validator) ClearViolations() {
	v.mu.Lock()
	v.violations = nil
	v.mu.Unlock()
}

// Global validator instance
var globalValidator = NewValidator(true) //nolint:gochecknoglobals

// Global API functions for convenience
func SetGlobalValidationEnabled(enabled bool) {
	globalValidator.SetEnabled(enabled)
}

func IsGlobalValidationEnabled() bool {
	return globalValidator.IsEnabled()
}

func ValidateBoundary(operation string) error {
	return globalValidator.ValidateCall(operation)
}

func GetGlobalViolations() []string {
	return globalValidator.GetViolations()
}

func ClearGlobalViolations() {
	globalValidator.ClearViolations()
}

func AllowGlobalCrossing(from, to string) {
	globalValidator.Allow(from, to)
}
