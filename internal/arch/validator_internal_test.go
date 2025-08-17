package arch

import (
	"sync"
	"testing"
)

// Internal tests for private methods using synthetic call stacks.
// Synthetic stacks use the same format runtime.CallersFrames gives in frame.Function.

func Test_extractAdapterType(t *testing.T) {
	t.Parallel()
	v := NewValidator(true)

	cases := []struct {
		fn   string
		want string
	}{
		{"github.com/sufield/ephemos/internal/adapters/grpc.(*Server).Serve", "grpc"},
		{"github.com/sufield/ephemos/internal/adapters/http.Handler.ServeHTTP", "http"},
		{"github.com/sufield/ephemos/internal/adapters/primary/api.(*H).Do", "primary"},
		{"github.com/sufield/ephemos/internal/adapters/secondary/spiffe.(*Provider).Get", "secondary"},
		{"github.com/sufield/ephemos/internal/adapters/grpc/client.(*C).Call", "grpc"},
		{"github.com/sufield/ephemos/internal/core/domain.Service.Do", ""},
		{"github.com/sufield/ephemos/pkg/ephemos.Client.Connect", ""},
		{"", ""},
	}
	for _, tc := range cases {
		t.Run(tc.fn, func(t *testing.T) {
			t.Parallel()
			got := v.extractAdapterType(tc.fn)
			if got != tc.want {
				t.Fatalf("extractAdapterType(%q) = %q, want %q", tc.fn, got, tc.want)
			}
		})
	}
}

func Test_checkCallStackViolation_directAdapterToAdapter(t *testing.T) {
	t.Parallel()
	v := NewValidator(true)

	// Most-recent first (top of stack), then its caller, etc.
	stack := []string{
		"github.com/sufield/ephemos/internal/adapters/http.(*H).Do",
		"github.com/sufield/ephemos/internal/adapters/grpc.(*S).Serve",
	}

	violation := v.checkCallStackViolation(stack, "test operation")
	if violation == "" {
		t.Fatalf("expected adapter->adapter violation, got none")
	}

	// Should mention both adapter types
	if !contains(violation, "http") || !contains(violation, "grpc") {
		t.Fatalf("violation should mention both adapter types, got: %q", violation)
	}
}

func Test_checkCallStackViolation_sameAdapterOkay(t *testing.T) {
	t.Parallel()
	v := NewValidator(true)

	stack := []string{
		"github.com/sufield/ephemos/internal/adapters/http.(*H).Do",
		"github.com/sufield/ephemos/internal/adapters/http.(*S).Serve",
	}
	if got := v.checkCallStackViolation(stack, "test operation"); got != "" {
		t.Fatalf("expected no violation for same adapter type, got %q", got)
	}
}

func Test_checkCallStackViolation_domainToAdapter(t *testing.T) {
	t.Parallel()
	v := NewValidator(true)

	stack := []string{
		"github.com/sufield/ephemos/internal/core/domain.Service.Method",
		"github.com/sufield/ephemos/internal/adapters/http.(*H).Do",
	}
	violation := v.checkCallStackViolation(stack, "test operation")
	if violation == "" {
		t.Fatalf("expected domain->adapter violation")
	}

	// Should mention domain and adapter
	if !contains(violation, "domain") || !contains(violation, "http") {
		t.Fatalf("violation should mention domain and adapter, got: %q", violation)
	}
}

func Test_checkCallStackViolation_indirectCallOkay(t *testing.T) {
	t.Parallel()
	v := NewValidator(true)

	// Indirect call through service layer - should not trigger violation
	stack := []string{
		"github.com/sufield/ephemos/internal/adapters/http.(*H).Do",
		"github.com/sufield/ephemos/internal/core/services.(*Service).Process",
		"github.com/sufield/ephemos/internal/adapters/grpc.(*Client).Call",
	}
	if got := v.checkCallStackViolation(stack, "test operation"); got != "" {
		t.Fatalf("expected no violation for indirect call through service, got %q", got)
	}
}

func Test_checkCallStackViolation_allowlistRespected(t *testing.T) {
	t.Parallel()
	v := NewValidator(true)

	// Add http->shared to allowlist
	v.Allow("http", "shared")

	stack := []string{
		"github.com/sufield/ephemos/internal/adapters/http.(*H).Do",
		"github.com/sufield/ephemos/internal/adapters/shared.(*Util).Helper",
	}
	if got := v.checkCallStackViolation(stack, "test operation"); got != "" {
		t.Fatalf("expected no violation for allowed crossing, got %q", got)
	}
}

func Test_checkCallStackViolation_allowlistNotBidirectional(t *testing.T) {
	t.Parallel()
	v := NewValidator(true)

	// Allow http->shared but not shared->http
	v.Allow("http", "shared")

	stack := []string{
		"github.com/sufield/ephemos/internal/adapters/shared.(*Util).Helper",
		"github.com/sufield/ephemos/internal/adapters/http.(*H).Do",
	}
	violation := v.checkCallStackViolation(stack, "test operation")
	if violation == "" {
		t.Fatalf("expected violation for reverse direction not in allowlist")
	}
}

func Test_allowed_threadSafety(t *testing.T) {
	t.Parallel()
	v := NewValidator(true)

	const workers = 10
	const iterations = 100
	var wg sync.WaitGroup
	wg.Add(workers)

	for i := 0; i < workers; i++ {
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				// Concurrent allowlist modifications
				v.Allow("worker", "target")
				_ = v.allowed("worker", "target")
				v.Allow("other", "destination")
				_ = v.allowed("other", "destination")
			}
		}(i)
	}
	wg.Wait()
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			indexOf(s, substr) >= 0)))
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
