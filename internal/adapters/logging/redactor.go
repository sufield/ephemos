// Package logging provides secure logging utilities with automatic redaction of sensitive data.
package logging

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
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
//
//nolint:gocritic // Required by slog.Handler interface
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
	return &RedactorHandler{handler: h.handler.WithAttrs(redactedAttrs)}
}

// WithGroup implements slog.Handler.
func (h *RedactorHandler) WithGroup(name string) slog.Handler {
	return &RedactorHandler{handler: h.handler.WithGroup(name)}
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

	// Redact SPIFFE IDs that might contain sensitive service names in development
	// Note: SPIFFE IDs are generally not sensitive, but we can redact in debug contexts
	// This is configurable based on policy

	return value
}

// NewSecureSlogLogger creates a new slog.Logger with automatic sensitive data redaction.
// This returns a standard *slog.Logger for backward compatibility.
func NewSecureSlogLogger(handler slog.Handler) *slog.Logger {
	return slog.New(NewRedactorHandler(handler))
}
