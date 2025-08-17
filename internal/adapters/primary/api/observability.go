package api

import "context"

// OnAuthAttempt provides a lightweight hook for auth observability.
// This allows future integration with OpenTelemetry or other observability
// systems without API churn. By default, it's a no-op.
//
// Usage:
//
//	defer func() { OnAuthAttempt(ctx, err) }()
//
// Later, observability layers can assign:
//
//	OnAuthAttempt = otelRecordAuthAttempt
var OnAuthAttempt = func(ctx context.Context, err error) {}
