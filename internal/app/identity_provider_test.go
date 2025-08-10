package ports_test

import (
	"testing"

	"github.com/sufield/ephemos/internal/core/ports"
)

func TestErrIdentityNotFound(t *testing.T) {
	// Test the standard error
	if ports.ErrIdentityNotFound == nil {
		t.Error("ErrIdentityNotFound should not be nil")
	}

	expectedMsg := "identity not found"
	if ports.ErrIdentityNotFound.Error() != expectedMsg {
		t.Errorf("ErrIdentityNotFound.Error() = %v, want %v", ports.ErrIdentityNotFound.Error(), expectedMsg)
	}
}
