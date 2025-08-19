package domain

import "fmt"

// ComponentType is an enum for different types of components in the system.
type ComponentType int

const (
	ComponentUnknown ComponentType = iota
	ComponentSpireServer
	ComponentSpireAgent
	ComponentAgent
	ComponentServer
	ComponentClient
	ComponentService
)

var componentTypeStrings = map[ComponentType]string{
	ComponentSpireServer: "spire-server",
	ComponentSpireAgent:  "spire-agent",
	ComponentAgent:       "agent",
	ComponentServer:      "server",
	ComponentClient:      "client",
	ComponentService:     "service",
}

var stringToComponentType = map[string]ComponentType{
	"spire-server": ComponentSpireServer,
	"spire-agent":  ComponentSpireAgent,
	"agent":        ComponentAgent,
	"server":       ComponentServer,
	"client":       ComponentClient,
	"service":      ComponentService,
}

// String returns the string representation.
func (c ComponentType) String() string {
	if s, ok := componentTypeStrings[c]; ok {
		return s
	}
	return "unknown"
}

// ParseComponentType parses a string to ComponentType.
func ParseComponentType(s string) (ComponentType, error) {
	if comp, ok := stringToComponentType[s]; ok {
		return comp, nil
	}
	return ComponentUnknown, fmt.Errorf("invalid component type: %s", s)
}

// IsValid returns true if the component type is known/valid.
func (c ComponentType) IsValid() bool {
	_, ok := componentTypeStrings[c]
	return ok
}

// IsSpireComponent returns true if the component is a SPIRE component.
func (c ComponentType) IsSpireComponent() bool {
	switch c {
	case ComponentSpireServer, ComponentSpireAgent:
		return true
	default:
		return false
	}
}

// IsServerType returns true if the component is a server-type component.
func (c ComponentType) IsServerType() bool {
	switch c {
	case ComponentServer, ComponentSpireServer, ComponentService:
		return true
	default:
		return false
	}
}

// IsClientType returns true if the component is a client-type component.
func (c ComponentType) IsClientType() bool {
	switch c {
	case ComponentClient, ComponentSpireAgent:
		return true
	default:
		return false
	}
}
