// Package ports defines the interfaces (ports) for the Ephemos library.
package ports

import (
	"log/slog"
)

// SecureLoggerProvider provides secure logging capabilities with automatic redaction.
type SecureLoggerProvider interface {
	// CreateSecureLogger creates a logger that automatically redacts sensitive data
	CreateSecureLogger(handler slog.Handler) *slog.Logger
}
