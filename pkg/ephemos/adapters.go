package ephemos

import (
	"github.com/sufield/ephemos/internal/core/ports"
)

// configAdapter adapts public Configuration interface to internal *ports.Configuration.
type configAdapter struct {
	internal *ports.Configuration
}

func (c *configAdapter) Validate() error {
	if c.internal == nil {
		return ErrConfigInvalid
	}
	return c.internal.Validate()
}

func (c *configAdapter) IsProductionReady() error {
	if c.internal == nil {
		return ErrConfigInvalid
	}
	return c.internal.IsProductionReady()
}

// NewConfiguration creates a public Configuration from internal configuration.
// This is a bridge function for internal use.
func NewConfiguration(internal *ports.Configuration) Configuration {
	if internal == nil {
		return nil
	}
	return &configAdapter{internal: internal}
}

// GetInternalConfig extracts the internal configuration for factory use.
// This is a bridge function for internal use.
func GetInternalConfig(config Configuration) (*ports.Configuration, error) {
	if config == nil {
		return nil, ErrConfigInvalid
	}
	if adapter, ok := config.(*configAdapter); ok && adapter.internal != nil {
		return adapter.internal, nil
	}
	return nil, ErrConfigInvalid
}
