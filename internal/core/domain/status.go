package domain

import "fmt"

// Status is an enum for component or service status.
type Status int

const (
	StatusUnknown Status = iota
	StatusHealthy
	StatusUnhealthy
	StatusUp
	StatusDown
	StatusRunning
	StatusStopped
	StatusActive
	StatusInactive
	StatusReady
	StatusNotReady
	StatusEnabled
	StatusDisabled
)

var statusStrings = map[Status]string{
	StatusHealthy:   "healthy",
	StatusUnhealthy: "unhealthy",
	StatusUp:        "up",
	StatusDown:      "down",
	StatusRunning:   "running",
	StatusStopped:   "stopped",
	StatusActive:    "active",
	StatusInactive:  "inactive",
	StatusReady:     "ready",
	StatusNotReady:  "not_ready",
	StatusEnabled:   "enabled",
	StatusDisabled:  "disabled",
}

var stringToStatus = map[string]Status{
	"healthy":   StatusHealthy,
	"unhealthy": StatusUnhealthy,
	"up":        StatusUp,
	"down":      StatusDown,
	"running":   StatusRunning,
	"stopped":   StatusStopped,
	"active":    StatusActive,
	"inactive":  StatusInactive,
	"ready":     StatusReady,
	"not_ready": StatusNotReady,
	"enabled":   StatusEnabled,
	"disabled":  StatusDisabled,
}

// String returns the string representation.
func (s Status) String() string {
	if str, ok := statusStrings[s]; ok {
		return str
	}
	return "unknown"
}

// ParseStatus parses a string to Status.
func ParseStatus(s string) (Status, error) {
	if status, ok := stringToStatus[s]; ok {
		return status, nil
	}
	return StatusUnknown, fmt.Errorf("invalid status: %s", s)
}

// IsValid returns true if the status is known/valid.
func (s Status) IsValid() bool {
	_, ok := statusStrings[s]
	return ok
}

// IsHealthy returns true if the status indicates a healthy state.
func (s Status) IsHealthy() bool {
	switch s {
	case StatusHealthy, StatusUp, StatusRunning, StatusActive, StatusReady, StatusEnabled:
		return true
	default:
		return false
	}
}

// IsOperational returns true if the status indicates the component is operational.
func (s Status) IsOperational() bool {
	switch s {
	case StatusHealthy, StatusUp, StatusRunning, StatusActive, StatusReady:
		return true
	default:
		return false
	}
}

// IsErrorState returns true if the status indicates an error or unhealthy state.
func (s Status) IsErrorState() bool {
	switch s {
	case StatusUnhealthy, StatusDown, StatusStopped, StatusInactive, StatusNotReady:
		return true
	default:
		return false
	}
}
