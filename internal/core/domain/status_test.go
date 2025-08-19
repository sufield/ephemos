package domain

import "testing"

func TestStatus_String(t *testing.T) {
	tests := []struct {
		status   Status
		expected string
	}{
		{StatusHealthy, "healthy"},
		{StatusUnhealthy, "unhealthy"},
		{StatusUp, "up"},
		{StatusDown, "down"},
		{StatusRunning, "running"},
		{StatusStopped, "stopped"},
		{StatusActive, "active"},
		{StatusInactive, "inactive"},
		{StatusReady, "ready"},
		{StatusNotReady, "not_ready"},
		{StatusEnabled, "enabled"},
		{StatusDisabled, "disabled"},
		{StatusUnknown, "unknown"},
		{Status(999), "unknown"}, // Invalid status
	}

	for _, tt := range tests {
		if got := tt.status.String(); got != tt.expected {
			t.Errorf("Status.String() = %v, want %v", got, tt.expected)
		}
	}
}

func TestParseStatus(t *testing.T) {
	tests := []struct {
		input    string
		expected Status
		wantErr  bool
	}{
		{"healthy", StatusHealthy, false},
		{"unhealthy", StatusUnhealthy, false},
		{"up", StatusUp, false},
		{"down", StatusDown, false},
		{"running", StatusRunning, false},
		{"stopped", StatusStopped, false},
		{"active", StatusActive, false},
		{"inactive", StatusInactive, false},
		{"ready", StatusReady, false},
		{"not_ready", StatusNotReady, false},
		{"enabled", StatusEnabled, false},
		{"disabled", StatusDisabled, false},
		{"invalid", StatusUnknown, true},
		{"", StatusUnknown, true},
		{"HEALTHY", StatusUnknown, true}, // Case sensitive
	}

	for _, tt := range tests {
		got, err := ParseStatus(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseStatus(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}
		if got != tt.expected {
			t.Errorf("ParseStatus(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestStatus_IsValid(t *testing.T) {
	tests := []struct {
		status   Status
		expected bool
	}{
		{StatusHealthy, true},
		{StatusUnhealthy, true},
		{StatusUp, true},
		{StatusDown, true},
		{StatusRunning, true},
		{StatusStopped, true},
		{StatusActive, true},
		{StatusInactive, true},
		{StatusReady, true},
		{StatusNotReady, true},
		{StatusEnabled, true},
		{StatusDisabled, true},
		{StatusUnknown, false},
		{Status(999), false},
	}

	for _, tt := range tests {
		if got := tt.status.IsValid(); got != tt.expected {
			t.Errorf("Status.IsValid() = %v, want %v for status %v", got, tt.expected, tt.status)
		}
	}
}

func TestStatus_IsHealthy(t *testing.T) {
	tests := []struct {
		status   Status
		expected bool
	}{
		{StatusHealthy, true},
		{StatusUnhealthy, false},
		{StatusUp, true},
		{StatusDown, false},
		{StatusRunning, true},
		{StatusStopped, false},
		{StatusActive, true},
		{StatusInactive, false},
		{StatusReady, true},
		{StatusNotReady, false},
		{StatusEnabled, true},
		{StatusDisabled, false},
		{StatusUnknown, false},
	}

	for _, tt := range tests {
		if got := tt.status.IsHealthy(); got != tt.expected {
			t.Errorf("Status.IsHealthy() = %v, want %v for status %v", got, tt.expected, tt.status)
		}
	}
}

func TestStatus_IsOperational(t *testing.T) {
	tests := []struct {
		status   Status
		expected bool
	}{
		{StatusHealthy, true},
		{StatusUnhealthy, false},
		{StatusUp, true},
		{StatusDown, false},
		{StatusRunning, true},
		{StatusStopped, false},
		{StatusActive, true},
		{StatusInactive, false},
		{StatusReady, true},
		{StatusNotReady, false},
		{StatusEnabled, false}, // enabled != operational
		{StatusDisabled, false},
		{StatusUnknown, false},
	}

	for _, tt := range tests {
		if got := tt.status.IsOperational(); got != tt.expected {
			t.Errorf("Status.IsOperational() = %v, want %v for status %v", got, tt.expected, tt.status)
		}
	}
}

func TestStatus_IsErrorState(t *testing.T) {
	tests := []struct {
		status   Status
		expected bool
	}{
		{StatusHealthy, false},
		{StatusUnhealthy, true},
		{StatusUp, false},
		{StatusDown, true},
		{StatusRunning, false},
		{StatusStopped, true},
		{StatusActive, false},
		{StatusInactive, true},
		{StatusReady, false},
		{StatusNotReady, true},
		{StatusEnabled, false},
		{StatusDisabled, false}, // disabled != error state
		{StatusUnknown, false},
	}

	for _, tt := range tests {
		if got := tt.status.IsErrorState(); got != tt.expected {
			t.Errorf("Status.IsErrorState() = %v, want %v for status %v", got, tt.expected, tt.status)
		}
	}
}
