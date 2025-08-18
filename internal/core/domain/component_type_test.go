package domain

import "testing"

func TestComponentType_String(t *testing.T) {
	tests := []struct {
		component ComponentType
		expected  string
	}{
		{ComponentSpireServer, "spire-server"},
		{ComponentSpireAgent, "spire-agent"},
		{ComponentAgent, "agent"},
		{ComponentServer, "server"},
		{ComponentClient, "client"},
		{ComponentService, "service"},
		{ComponentUnknown, "unknown"},
		{ComponentType(999), "unknown"}, // Invalid component
	}

	for _, tt := range tests {
		if got := tt.component.String(); got != tt.expected {
			t.Errorf("ComponentType.String() = %v, want %v", got, tt.expected)
		}
	}
}

func TestParseComponentType(t *testing.T) {
	tests := []struct {
		input    string
		expected ComponentType
		wantErr  bool
	}{
		{"spire-server", ComponentSpireServer, false},
		{"spire-agent", ComponentSpireAgent, false},
		{"agent", ComponentAgent, false},
		{"server", ComponentServer, false},
		{"client", ComponentClient, false},
		{"service", ComponentService, false},
		{"invalid", ComponentUnknown, true},
		{"", ComponentUnknown, true},
		{"SPIRE-SERVER", ComponentUnknown, true}, // Case sensitive
	}

	for _, tt := range tests {
		got, err := ParseComponentType(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseComponentType(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}
		if got != tt.expected {
			t.Errorf("ParseComponentType(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestComponentType_IsValid(t *testing.T) {
	tests := []struct {
		component ComponentType
		expected  bool
	}{
		{ComponentSpireServer, true},
		{ComponentSpireAgent, true},
		{ComponentAgent, true},
		{ComponentServer, true},
		{ComponentClient, true},
		{ComponentService, true},
		{ComponentUnknown, false},
		{ComponentType(999), false},
	}

	for _, tt := range tests {
		if got := tt.component.IsValid(); got != tt.expected {
			t.Errorf("ComponentType.IsValid() = %v, want %v for component %v", got, tt.expected, tt.component)
		}
	}
}

func TestComponentType_IsSpireComponent(t *testing.T) {
	tests := []struct {
		component ComponentType
		expected  bool
	}{
		{ComponentSpireServer, true},
		{ComponentSpireAgent, true},
		{ComponentAgent, false},
		{ComponentServer, false},
		{ComponentClient, false},
		{ComponentService, false},
		{ComponentUnknown, false},
	}

	for _, tt := range tests {
		if got := tt.component.IsSpireComponent(); got != tt.expected {
			t.Errorf("ComponentType.IsSpireComponent() = %v, want %v for component %v", got, tt.expected, tt.component)
		}
	}
}

func TestComponentType_IsServerType(t *testing.T) {
	tests := []struct {
		component ComponentType
		expected  bool
	}{
		{ComponentSpireServer, true},
		{ComponentSpireAgent, false},
		{ComponentAgent, false},
		{ComponentServer, true},
		{ComponentClient, false},
		{ComponentService, true},
		{ComponentUnknown, false},
	}

	for _, tt := range tests {
		if got := tt.component.IsServerType(); got != tt.expected {
			t.Errorf("ComponentType.IsServerType() = %v, want %v for component %v", got, tt.expected, tt.component)
		}
	}
}

func TestComponentType_IsClientType(t *testing.T) {
	tests := []struct {
		component ComponentType
		expected  bool
	}{
		{ComponentSpireServer, false},
		{ComponentSpireAgent, true},
		{ComponentAgent, false},
		{ComponentServer, false},
		{ComponentClient, true},
		{ComponentService, false},
		{ComponentUnknown, false},
	}

	for _, tt := range tests {
		if got := tt.component.IsClientType(); got != tt.expected {
			t.Errorf("ComponentType.IsClientType() = %v, want %v for component %v", got, tt.expected, tt.component)
		}
	}
}