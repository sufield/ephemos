package buildinfo

import (
	"testing"
)

func TestGet(t *testing.T) {
	info := Get()
	
	// Test that we get the expected default values
	if info.Version == "" {
		t.Error("Version should not be empty")
	}
	
	if info.CommitHash == "" {
		t.Error("CommitHash should not be empty")
	}
	
	if info.BuildTime == "" {
		t.Error("BuildTime should not be empty")
	}
	
	if info.BuildUser == "" {
		t.Error("BuildUser should not be empty")
	}
	
	if info.BuildHost == "" {
		t.Error("BuildHost should not be empty")
	}
}

func TestDefaultValues(t *testing.T) {
	// Test that default values are reasonable
	if Version != "dev" {
		t.Errorf("Expected default Version to be 'dev', got %q", Version)
	}
	
	if CommitHash != "unknown" {
		t.Errorf("Expected default CommitHash to be 'unknown', got %q", CommitHash)
	}
	
	if BuildTime != "unknown" {
		t.Errorf("Expected default BuildTime to be 'unknown', got %q", BuildTime)
	}
}