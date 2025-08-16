//go:build integration

package ephemos_test

import (
	"context"
	"testing"

	ephemos "github.com/sufield/ephemos/pkg/ephemos"
)

func Test_Integration_Constructors_smoke(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// These may fail without a real config/backing; assert behavior you expect.
	if _, err := ephemos.IdentityClientFromFile(ctx, "config.yaml"); err == nil {
		t.Fatalf("expected an error without real configuration")
	}
	if _, err := ephemos.IdentityServerFromFile(ctx, "config.yaml"); err == nil {
		t.Fatalf("expected an error without real configuration")
	}
}
