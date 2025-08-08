package ports

import (
	"testing"
)

func TestErrIdentityNotFound(t *testing.T) {
	// Test the standard error
	if ErrIdentityNotFound == nil {
		t.Error("ErrIdentityNotFound should not be nil")
	}

	expectedMsg := "identity not found"
	if ErrIdentityNotFound.Error() != expectedMsg {
		t.Errorf("ErrIdentityNotFound.Error() = %v, want %v", ErrIdentityNotFound.Error(), expectedMsg)
	}
}
