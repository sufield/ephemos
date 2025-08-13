// Package logging provides secure logging adapter implementations.
package logging

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/sufield/ephemos/internal/core/ports"
)

// RedactedValue is the placeholder for redacted sensitive data.
const RedactedValue = "[REDACTED]"

// RedactorHandler wraps an slog.Handler to automatically redact sensitive fields.
type RedactorHandler struct {
	handler         slog.Handler
	sensitiveFields map[string]bool
}

// NewRedactorHandler creates a new handler that redacts sensitive fields.
func NewRedactorHandler(handler slog.Handler) *RedactorHandler {
	return &RedactorHandler{
		handler: handler,
		sensitiveFields: map[string]bool{
			"password":      true,
			"secret":        true,
			"token":         true,
			"key":           true,
			"private_key":   true,
			"privatekey":    true,
			"private-key":   true,
			"cert":          true,
			"certificate":   true,
			"credentials":   true,
			"auth":          true,
			"bearer":        true,
			"authorization": true,
		},
	}
}

// Enabled implements slog.Handler.
func (h *RedactorHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

// Handle implements slog.Handler with sensitive data redaction.
func (h *RedactorHandler) Handle(ctx context.Context, record slog.Record) error {
	// Create a new record with redacted attributes
	newRecord := slog.Record{
		Time:    record.Time,
		Level:   record.Level,
		Message: record.Message,
		PC:      record.PC,
	}

	record.Attrs(func(attr slog.Attr) bool {
		newRecord.AddAttrs(h.redactAttr(attr))
		return true
	})

	if err := h.handler.Handle(ctx, newRecord); err != nil {
		return fmt.Errorf("redactor handle failed: %w", err)
	}
	return nil
}

// WithAttrs implements slog.Handler.
func (h *RedactorHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	redactedAttrs := make([]slog.Attr, len(attrs))
	for i, attr := range attrs {
		redactedAttrs[i] = h.redactAttr(attr)
	}
	return &RedactorHandler{
		handler:         h.handler.WithAttrs(redactedAttrs),
		sensitiveFields: h.sensitiveFields,
	}
}

// WithGroup implements slog.Handler.
func (h *RedactorHandler) WithGroup(name string) slog.Handler {
	return &RedactorHandler{
		handler:         h.handler.WithGroup(name),
		sensitiveFields: h.sensitiveFields,
	}
}

// redactAttr redacts sensitive attributes recursively.
func (h *RedactorHandler) redactAttr(attr slog.Attr) slog.Attr {
	// Check if this field should be redacted
	if h.isSensitiveField(attr.Key) {
		return slog.Attr{
			Key:   attr.Key,
			Value: slog.StringValue(RedactedValue),
		}
	}

	// Handle group attributes recursively
	if attr.Value.Kind() == slog.KindGroup {
		group := attr.Value.Group()
		redactedAttrs := make([]slog.Attr, len(group))
		for i, groupAttr := range group {
			redactedAttrs[i] = h.redactAttr(groupAttr)
		}
		return slog.Attr{
			Key:   attr.Key,
			Value: slog.GroupValue(redactedAttrs...),
		}
	}

	// Handle string values that might contain sensitive patterns
	if attr.Value.Kind() == slog.KindString {
		return slog.Attr{
			Key:   attr.Key,
			Value: slog.StringValue(h.redactSensitiveStrings(attr.Value.String())),
		}
	}

	return attr
}

// isSensitiveField checks if a field name indicates sensitive data.
func (h *RedactorHandler) isSensitiveField(fieldName string) bool {
	lower := strings.ToLower(fieldName)

	// Direct match
	if h.sensitiveFields[lower] {
		return true
	}

	// Pattern matching for compound field names
	for sensitive := range h.sensitiveFields {
		if strings.Contains(lower, sensitive) {
			return true
		}
	}

	return false
}

// redactSensitiveStrings redacts sensitive patterns in string values.
func (h *RedactorHandler) redactSensitiveStrings(value string) string {
	// Redact certificate content (PEM format)
	if strings.Contains(value, "BEGIN CERTIFICATE") || strings.Contains(value, "BEGIN PRIVATE KEY") {
		return RedactedValue
	}

	// Redact JWT tokens (basic pattern matching)
	if strings.Count(value, ".") >= 2 && len(value) > 50 {
		// Looks like a JWT token
		return RedactedValue
	}

	return value
}

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

// NewSecureSlogLogger creates a new slog.Logger with automatic sensitive data redaction.
// This function is for backward compatibility with code that expects *slog.Logger directly.
func NewSecureSlogLogger(handler slog.Handler) *slog.Logger {
	return slog.New(NewRedactorHandler(handler))
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