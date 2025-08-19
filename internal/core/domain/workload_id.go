// Package domain contains workload ID validation with domain intent rather than primitive string checks.
package domain

import (
	"fmt"
	"strings"
)

// WorkloadID represents a workload identifier with domain validation.
// This value object expresses domain intent instead of asking for raw string data.
type WorkloadID string

// NewWorkloadID creates a validated workload ID from a raw string.
// This constructor guarantees validity at creation time.
func NewWorkloadID(rawID string) (WorkloadID, error) {
	id := WorkloadID(rawID)

	if id.IsEmpty() {
		return "", fmt.Errorf("workload ID cannot be empty")
	}

	if id.IsOnlyWhitespace() {
		return "", fmt.Errorf("workload ID cannot be only whitespace")
	}

	if id.HasInvalidCharacters() {
		return "", fmt.Errorf("workload ID contains invalid characters")
	}

	return id, nil
}

// NewWorkloadIDUnsafe creates a workload ID without validation (for trusted contexts).
func NewWorkloadIDUnsafe(rawID string) WorkloadID {
	return WorkloadID(rawID)
}

// IsEmpty returns true if the workload ID is empty.
// This expresses domain intent: "is this ID empty?" instead of "equals empty string?".
func (wid WorkloadID) IsEmpty() bool {
	return string(wid) == ""
}

// IsOnlyWhitespace returns true if the workload ID contains only whitespace.
// This expresses domain intent: "is this just whitespace?" instead of string trimming checks.
func (wid WorkloadID) IsOnlyWhitespace() bool {
	return strings.TrimSpace(string(wid)) == ""
}

// HasInvalidCharacters returns true if the workload ID contains invalid characters.
// This expresses domain intent: "has invalid chars?" instead of character-by-character checks.
func (wid WorkloadID) HasInvalidCharacters() bool {
	// For now, just check for control characters - extend as needed
	for _, r := range string(wid) {
		if r < 32 && r != '\t' { // Allow tab but not other control chars
			return true
		}
	}
	return false
}

// IsValid returns true if the workload ID passes all validation rules.
// This expresses overall domain intent: "is this a valid workload ID?".
func (wid WorkloadID) IsValid() bool {
	return !wid.IsEmpty() && !wid.IsOnlyWhitespace() && !wid.HasInvalidCharacters()
}

// String returns the string representation of the workload ID.
func (wid WorkloadID) String() string {
	return string(wid)
}

// Equals compares two workload IDs for equality.
// This expresses domain intent: "are these IDs the same?" instead of string comparison.
func (wid WorkloadID) Equals(other WorkloadID) bool {
	return string(wid) == string(other)
}