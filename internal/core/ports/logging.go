// Package ports defines the interfaces (ports) for the Ephemos library.
package ports

import (
	"context"
)

// LogLevel represents the logging level for messages.
type LogLevel int

const (
	// LogLevelDebug represents debug level logging.
	LogLevelDebug LogLevel = iota
	// LogLevelInfo represents info level logging.
	LogLevelInfo
	// LogLevelWarn represents warn level logging.
	LogLevelWarn
	// LogLevelError represents error level logging.
	LogLevelError
)

// LogAttribute represents a key-value pair for structured logging.
type LogAttribute struct {
	Key   string
	Value interface{}
}

// Logger provides secure logging capabilities with automatic redaction.
type Logger interface {
	// Debug logs a debug level message.
	Debug(ctx context.Context, message string, attrs ...LogAttribute)
	// Info logs an info level message.
	Info(ctx context.Context, message string, attrs ...LogAttribute)
	// Warn logs a warning level message.
	Warn(ctx context.Context, message string, attrs ...LogAttribute)
	// Error logs an error level message.
	Error(ctx context.Context, message string, attrs ...LogAttribute)
	// WithAttrs returns a new logger with the given attributes added.
	WithAttrs(attrs ...LogAttribute) Logger
	// WithGroup returns a new logger with the given group name.
	WithGroup(name string) Logger
}

// LoggerProvider provides secure logging capabilities with automatic redaction.
type LoggerProvider interface {
	// GetLogger returns a logger for the given context.
	GetLogger(ctx context.Context) Logger
}
