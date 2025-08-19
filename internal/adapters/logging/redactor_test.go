package logging_test

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/sufield/ephemos/internal/adapters/logging"
)

func TestRedactorHandler_SensitiveFields(t *testing.T) {
	var buf bytes.Buffer
	baseHandler := slog.NewTextHandler(&buf, nil)
	redactorHandler := logging.NewRedactorHandler(baseHandler)
	logger := slog.New(redactorHandler)

	tests := []struct {
		name         string
		logFunc      func()
		shouldRedact bool
		fieldName    string
	}{
		{
			name: "password field redacted",
			logFunc: func() {
				logger.Info("User login", "password", "secret123")
			},
			shouldRedact: true,
			fieldName:    "password",
		},
		{
			name: "private_key field redacted",
			logFunc: func() {
				logger.Info("Key loaded", "private_key", "-----BEGIN PRIVATE KEY-----")
			},
			shouldRedact: true,
			fieldName:    "private_key",
		},
		{
			name: "certificate field redacted",
			logFunc: func() {
				logger.Info("Cert loaded", "certificate", "-----BEGIN CERTIFICATE-----")
			},
			shouldRedact: true,
			fieldName:    "certificate",
		},
		{
			name: "normal field not redacted",
			logFunc: func() {
				logger.Info("Service started", "port", "8080")
			},
			shouldRedact: false,
			fieldName:    "port",
		},
		{
			name: "compound sensitive field redacted",
			logFunc: func() {
				logger.Info("Auth data", "bearer_token", "eyJhbGciOiJSUzI1NiJ9...")
			},
			shouldRedact: true,
			fieldName:    "bearer_token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			tt.logFunc()

			output := buf.String()

			if tt.shouldRedact {
				if !strings.Contains(output, logging.RedactedValue) {
					t.Errorf("Expected [REDACTED] in output, got: %s", output)
				}
				if strings.Contains(output, "secret123") ||
					strings.Contains(output, "BEGIN PRIVATE KEY") ||
					strings.Contains(output, "BEGIN CERTIFICATE") ||
					strings.Contains(output, "eyJhbGciOiJSUzI1NiJ9") {
					t.Errorf("Sensitive data was not redacted: %s", output)
				}
			} else if strings.Contains(output, logging.RedactedValue) {
				t.Errorf("Non-sensitive field was incorrectly redacted: %s", output)
			}
		})
	}
}

func TestRedactorHandler_GroupAttributes(t *testing.T) {
	var buf bytes.Buffer
	baseHandler := slog.NewTextHandler(&buf, nil)
	redactorHandler := logging.NewRedactorHandler(baseHandler)
	logger := slog.New(redactorHandler)

	logger.Info("Service config",
		slog.Group("auth",
			slog.String("username", "admin"),
			slog.String("password", "secret123"),
		),
		slog.Group("server",
			slog.String("host", "localhost"),
			slog.Int("port", 8080),
		),
	)

	output := buf.String()

	// Password should be redacted
	if !strings.Contains(output, logging.RedactedValue) {
		t.Error("Expected password to be redacted in group")
	}

	// The text handler shows the group as redacted since it contains sensitive data
	// This is actually correct behavior - if a group contains sensitive data,
	// the whole group gets redacted for safety
	t.Logf("Group redaction output: %s", output)

	if !strings.Contains(output, "localhost") {
		t.Error("Host should not be redacted")
	}

	// Sensitive data should not appear
	if strings.Contains(output, "secret123") {
		t.Error("Password value should not appear in output")
	}
}

func TestRedactorHandler_CertificateContent(t *testing.T) {
	var buf bytes.Buffer
	baseHandler := slog.NewTextHandler(&buf, nil)
	redactorHandler := logging.NewRedactorHandler(baseHandler)
	logger := slog.New(redactorHandler)

	certContent := `-----BEGIN CERTIFICATE-----
MIIDETCCAfmgAwIBAgIRAK+RuNhJjJRQqQoA5X0l+bIwDQYJKoZIhvcNAQELBQAw
...certificate content...
-----END CERTIFICATE-----`

	logger.Info("Certificate loaded", "cert_data", certContent)

	output := buf.String()

	if !strings.Contains(output, logging.RedactedValue) {
		t.Error("Certificate content should be redacted")
	}

	if strings.Contains(output, "BEGIN CERTIFICATE") {
		t.Error("Certificate content should not appear in output")
	}
}


func TestSensitiveFieldPatternMatching(t *testing.T) {
	var buf bytes.Buffer
	baseHandler := slog.NewTextHandler(&buf, nil)
	redactorHandler := logging.NewRedactorHandler(baseHandler)
	logger := slog.New(redactorHandler)

	tests := []struct {
		fieldName    string
		shouldRedact bool
	}{
		{"user_password", true},        // Contains "password"
		{"secret_key", true},           // Contains "secret" and "key"
		{"api_token", true},            // Contains "token"
		{"database_credentials", true}, // Contains "credentials"
		{"service_name", false},        // Normal field
		{"user_id", false},             // Normal field
		{"connection_timeout", false},  // Normal field
	}

	for _, tt := range tests {
		t.Run(tt.fieldName, func(t *testing.T) {
			buf.Reset()
			logger.Info("Test message", tt.fieldName, "sensitive_value")

			output := buf.String()

			if tt.shouldRedact {
				if !strings.Contains(output, logging.RedactedValue) {
					t.Errorf("Field %s should be redacted", tt.fieldName)
				}
				if strings.Contains(output, "sensitive_value") {
					t.Errorf("Sensitive value should not appear for field %s", tt.fieldName)
				}
			} else if strings.Contains(output, logging.RedactedValue) {
				t.Errorf("Field %s should not be redacted", tt.fieldName)
			}
		})
	}
}
