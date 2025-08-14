// Package logging provides secure logging adapter implementations.
package logging

import (
	"context"
	"log/slog"

	"github.com/sufield/ephemos/internal/core/ports"
)

// SecureLogger implements the ports.Logger interface with slog backend.
type SecureLogger struct {
	logger *slog.Logger
	attrs  []ports.LogAttribute
	group  string
}

// NewSecureLogger creates a new secure logger with automatic sensitive data redaction.
func NewSecureLogger(handler slog.Handler) *SecureLogger {
	return &SecureLogger{
		logger: slog.New(NewRedactorHandler(handler)),
	}
}

// Debug logs a debug level message.
func (l *SecureLogger) Debug(ctx context.Context, message string, attrs ...ports.LogAttribute) {
	l.log(ctx, slog.LevelDebug, message, attrs...)
}

// Info logs an info level message.
func (l *SecureLogger) Info(ctx context.Context, message string, attrs ...ports.LogAttribute) {
	l.log(ctx, slog.LevelInfo, message, attrs...)
}

// Warn logs a warning level message.
func (l *SecureLogger) Warn(ctx context.Context, message string, attrs ...ports.LogAttribute) {
	l.log(ctx, slog.LevelWarn, message, attrs...)
}

// Error logs an error level message.
func (l *SecureLogger) Error(ctx context.Context, message string, attrs ...ports.LogAttribute) {
	l.log(ctx, slog.LevelError, message, attrs...)
}

// WithAttrs returns a new logger with the given attributes added.
func (l *SecureLogger) WithAttrs(attrs ...ports.LogAttribute) ports.Logger {
	newAttrs := make([]ports.LogAttribute, len(l.attrs)+len(attrs))
	copy(newAttrs, l.attrs)
	copy(newAttrs[len(l.attrs):], attrs)

	return &SecureLogger{
		logger: l.logger,
		attrs:  newAttrs,
		group:  l.group,
	}
}

// WithGroup returns a new logger with the given group name.
func (l *SecureLogger) WithGroup(name string) ports.Logger {
	groupName := name
	if l.group != "" {
		groupName = l.group + "." + name
	}

	return &SecureLogger{
		logger: l.logger,
		attrs:  l.attrs,
		group:  groupName,
	}
}

// log performs the actual logging with level and attributes.
func (l *SecureLogger) log(ctx context.Context, level slog.Level, message string, attrs ...ports.LogAttribute) {
	// Convert ports.LogAttribute to slog.Attr
	slogAttrs := make([]slog.Attr, len(l.attrs)+len(attrs))

	// Add persistent attributes
	for i, attr := range l.attrs {
		slogAttrs[i] = slog.Any(attr.Key, attr.Value)
	}

	// Add current message attributes
	for i, attr := range attrs {
		slogAttrs[len(l.attrs)+i] = slog.Any(attr.Key, attr.Value)
	}

	// Log with group if specified
	logger := l.logger
	if l.group != "" {
		logger = logger.WithGroup(l.group)
	}

	logger.LogAttrs(ctx, level, message, slogAttrs...)
}

// SecureLoggerProvider implements ports.LoggerProvider.
type SecureLoggerProvider struct {
	handler slog.Handler
}

// NewSecureLoggerProvider creates a new secure logger provider.
func NewSecureLoggerProvider(handler slog.Handler) *SecureLoggerProvider {
	return &SecureLoggerProvider{
		handler: handler,
	}
}

// GetLogger returns a logger for the given context.
func (p *SecureLoggerProvider) GetLogger(ctx context.Context) ports.Logger {
	return NewSecureLogger(p.handler)
}
