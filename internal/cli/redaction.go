package cli

import (
	"regexp"
	"strings"
)

// redactSensitiveInfo removes or masks sensitive information from error messages and output
func redactSensitiveInfo(message string) string {
	// Patterns for sensitive information
	patterns := []struct {
		pattern *regexp.Regexp
		replace string
	}{
		// JWT tokens (Bearer tokens)
		{regexp.MustCompile(`Bearer\s+[A-Za-z0-9\-._~+/]+=*`), "Bearer [REDACTED]"},
		
		// Authorization headers
		{regexp.MustCompile(`Authorization:\s*[^\s]+`), "Authorization: [REDACTED]"},
		
		// API keys (common patterns)
		{regexp.MustCompile(`[Aa]pi[_-]?[Kk]ey[\s:=]+[A-Za-z0-9\-._~+/]+=*`), "api_key=[REDACTED]"},
		
		// Tokens in URLs or query parameters
		{regexp.MustCompile(`[?&]token=[A-Za-z0-9\-._~+/]+=*`), "&token=[REDACTED]"},
		
		// SPIFFE/SPIRE socket paths with usernames
		{regexp.MustCompile(`/home/[^/\s]+`), "/home/[USER]"},
		{regexp.MustCompile(`/Users/[^/\s]+`), "/Users/[USER]"},
		
		// Certificate data (PEM blocks)
		{regexp.MustCompile(`-----BEGIN [A-Z\s]+ CERTIFICATE-----[^-]+-----END [A-Z\s]+ CERTIFICATE-----`), "[CERTIFICATE REDACTED]"},
		{regexp.MustCompile(`-----BEGIN [A-Z\s]+ PRIVATE KEY-----[^-]+-----END [A-Z\s]+ PRIVATE KEY-----`), "[PRIVATE KEY REDACTED]"},
		
		// Password-like patterns
		{regexp.MustCompile(`[Pp]assword[\s:=]+[^\s]+`), "password=[REDACTED]"},
		{regexp.MustCompile(`[Pp]asswd[\s:=]+[^\s]+`), "passwd=[REDACTED]"},
		
		// Common secret environment variable patterns
		{regexp.MustCompile(`[A-Z_]*SECRET[A-Z_]*=\S+`), "[SECRET REDACTED]"},
		{regexp.MustCompile(`[A-Z_]*TOKEN[A-Z_]*=\S+`), "[TOKEN REDACTED]"},
		{regexp.MustCompile(`[A-Z_]*KEY[A-Z_]*=\S+`), "[KEY REDACTED]"},
	}
	
	result := message
	for _, p := range patterns {
		result = p.pattern.ReplaceAllString(result, p.replace)
	}
	
	return result
}

// RedactError redacts sensitive information from error messages
func RedactError(err error) string {
	if err == nil {
		return ""
	}
	return redactSensitiveInfo(err.Error())
}

// RedactString redacts sensitive information from any string
func RedactString(s string) string {
	return redactSensitiveInfo(s)
}