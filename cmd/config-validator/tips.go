package main

import (
	stderrors "errors"

	"github.com/sufield/ephemos/internal/core/errors"
	"github.com/sufield/ephemos/internal/core/ports"
)

// getProductionTips returns production readiness tips based on validation errors
func getProductionTips(err error) []string {
	var tips []string

	// Handle wrapped errors using errors.Is
	if stderrors.Is(err, errors.ErrExampleTrustDomain) {
		tips = append(tips, "Set EPHEMOS_TRUST_DOMAIN to your production domain (e.g., 'prod.company.com')")
	}

	if stderrors.Is(err, errors.ErrLocalhostTrustDomain) {
		tips = append(tips, "Set EPHEMOS_TRUST_DOMAIN to a proper domain (not localhost)")
	}

	if stderrors.Is(err, errors.ErrDemoTrustDomain) {
		tips = append(tips, "Set EPHEMOS_TRUST_DOMAIN to a production domain (not demo/test)")
	}

	if stderrors.Is(err, errors.ErrExampleServiceName) {
		tips = append(tips, "Set EPHEMOS_SERVICE_NAME to your production service name (not example)")
	}

	if stderrors.Is(err, errors.ErrDemoServiceName) {
		tips = append(tips, "Set EPHEMOS_SERVICE_NAME to your production service name (not demo)")
	}

	if stderrors.Is(err, errors.ErrDebugEnabled) {
		tips = append(tips, "Set EPHEMOS_DEBUG_ENABLED=false for production")
	}

	if stderrors.Is(err, errors.ErrInsecureSkipVerify) {
		tips = append(tips, "Remove EPHEMOS_INSECURE_SKIP_VERIFY or set to false for production")
	}

	if stderrors.Is(err, errors.ErrVerboseLogging) {
		tips = append(tips, "Set EPHEMOS_LOG_LEVEL to 'info' or 'error' for production (not debug/trace)")
	}

	if stderrors.Is(err, errors.ErrWildcardClients) {
		tips = append(tips, "Use specific SPIFFE IDs instead of wildcards in EPHEMOS_AUTHORIZED_CLIENTS")
	}

	if stderrors.Is(err, errors.ErrInsecureSocketPath) {
		tips = append(tips, "Set EPHEMOS_SPIFFE_SOCKET to a secure path like '/run/spire/sockets/api.sock'")
	}

	// Handle any other ProductionValidationError
	var prodErr *errors.ProductionValidationError
	if stderrors.As(err, &prodErr) && len(tips) == 0 {
		// Extract tips from wrapped errors
		for _, e := range prodErr.Errors {
			subTips := getProductionTips(e)
			tips = append(tips, subTips...)
		}
	}

	return tips
}

// getSecurityRecommendations returns security recommendations
func getSecurityRecommendations(envOnly bool) []string {
	var recommendations []string

	if !envOnly {
		recommendations = append(recommendations, "Use environment variables for production (--env-only)")
		recommendations = append(recommendations, "Environment variables are more secure than config files")
	}

	recommendations = append(recommendations, "Required environment variables for production:")
	recommendations = append(recommendations, "  export "+ports.EnvServiceName+"=\"your-service-name\"")
	recommendations = append(recommendations, "  export "+ports.EnvTrustDomain+"=\"your.production.domain\"")
	recommendations = append(recommendations, "Optional security environment variables:")
	recommendations = append(recommendations, "  export "+ports.EnvAgentSocket+"=\"/run/sockets/agent.sock\"")
	recommendations = append(recommendations, "  export "+ports.EnvDebugEnabled+"=\"false\"")
	recommendations = append(recommendations, "For more details, see:")
	recommendations = append(recommendations, "  docs/security/CONFIGURATION_SECURITY.md")
	recommendations = append(recommendations, "  config/README.md")

	return recommendations
}