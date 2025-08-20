package testing

import (
	"crypto/tls"
	"testing"
	"time"
)

// TLSBuilder wraps TLS configuration creation to enable testing
type TLSBuilder interface {
	ServerConfig() *tls.Config
	ClientConfig() *tls.Config
}

// FakeTLSBuilder provides a fake TLS builder for testing
type FakeTLSBuilder struct {
	serverConfig      *tls.Config
	clientConfig      *tls.Config
	rebuildTimestamps []time.Time
	callLog           []string
}

// NewFakeTLSBuilder creates a new fake TLS builder
func NewFakeTLSBuilder() *FakeTLSBuilder {
	return &FakeTLSBuilder{
		serverConfig:      &tls.Config{MinVersion: tls.VersionTLS12},
		clientConfig:      &tls.Config{MinVersion: tls.VersionTLS12},
		rebuildTimestamps: make([]time.Time, 0),
		callLog:           make([]string, 0),
	}
}

func (f *FakeTLSBuilder) ServerConfig() *tls.Config {
	f.callLog = append(f.callLog, "ServerConfig")
	f.rebuildTimestamps = append(f.rebuildTimestamps, time.Now())
	return f.serverConfig
}

func (f *FakeTLSBuilder) ClientConfig() *tls.Config {
	f.callLog = append(f.callLog, "ClientConfig")
	f.rebuildTimestamps = append(f.rebuildTimestamps, time.Now())
	return f.clientConfig
}

// Test helpers
func (f *FakeTLSBuilder) SetServerConfig(config *tls.Config) {
	f.serverConfig = config
}

func (f *FakeTLSBuilder) SetClientConfig(config *tls.Config) {
	f.clientConfig = config
}

func (f *FakeTLSBuilder) GetRebuildTimestamps() []time.Time {
	return f.rebuildTimestamps
}

func (f *FakeTLSBuilder) GetCallLog() []string {
	return f.callLog
}

func (f *FakeTLSBuilder) ClearCallLog() {
	f.callLog = make([]string, 0)
	f.rebuildTimestamps = make([]time.Time, 0)
}

// TriggerRotation simulates a certificate rotation event
func (f *FakeTLSBuilder) TriggerRotation() {
	// Record the rotation event
	f.rebuildTimestamps = append(f.rebuildTimestamps, time.Now())
}

func TestTLSBuilder_ServerConfig(t *testing.T) {
	builder := NewFakeTLSBuilder()

	config := builder.ServerConfig()

	if config == nil {
		t.Error("expected server config but got nil")
	}

	if config.MinVersion != tls.VersionTLS12 {
		t.Errorf("expected MinVersion %d, got %d", tls.VersionTLS12, config.MinVersion)
	}

	callLog := builder.GetCallLog()
	if len(callLog) != 1 || callLog[0] != "ServerConfig" {
		t.Errorf("expected call log [ServerConfig], got %v", callLog)
	}
}

func TestTLSBuilder_ClientConfig(t *testing.T) {
	builder := NewFakeTLSBuilder()

	config := builder.ClientConfig()

	if config == nil {
		t.Error("expected client config but got nil")
	}

	if config.MinVersion != tls.VersionTLS12 {
		t.Errorf("expected MinVersion %d, got %d", tls.VersionTLS12, config.MinVersion)
	}

	callLog := builder.GetCallLog()
	if len(callLog) != 1 || callLog[0] != "ClientConfig" {
		t.Errorf("expected call log [ClientConfig], got %v", callLog)
	}
}

func TestTLSBuilder_RebuildTiming(t *testing.T) {
	builder := NewFakeTLSBuilder()

	// Build configs multiple times
	_ = builder.ServerConfig()
	time.Sleep(10 * time.Millisecond) // Small delay to ensure different timestamps
	_ = builder.ClientConfig()
	time.Sleep(10 * time.Millisecond)
	_ = builder.ServerConfig()

	timestamps := builder.GetRebuildTimestamps()
	if len(timestamps) != 3 {
		t.Errorf("expected 3 rebuild timestamps, got %d", len(timestamps))
	}

	// Verify timestamps are in order and different
	for i := 1; i < len(timestamps); i++ {
		if !timestamps[i].After(timestamps[i-1]) {
			t.Errorf("timestamp %d should be after timestamp %d", i, i-1)
		}
	}
}

func TestTLSBuilder_RotationTiming(t *testing.T) {
	builder := NewFakeTLSBuilder()

	// Record initial state
	initialTimestamps := builder.GetRebuildTimestamps()
	if len(initialTimestamps) != 0 {
		t.Error("expected no initial rebuild timestamps")
	}

	// Trigger rotation
	builder.TriggerRotation()

	// Check that rotation was recorded
	rotationTimestamps := builder.GetRebuildTimestamps()
	if len(rotationTimestamps) != 1 {
		t.Errorf("expected 1 rotation timestamp, got %d", len(rotationTimestamps))
	}

	// Trigger multiple rotations
	time.Sleep(5 * time.Millisecond)
	builder.TriggerRotation()
	time.Sleep(5 * time.Millisecond)
	builder.TriggerRotation()

	finalTimestamps := builder.GetRebuildTimestamps()
	if len(finalTimestamps) != 3 {
		t.Errorf("expected 3 rotation timestamps, got %d", len(finalTimestamps))
	}

	// Verify all timestamps are different and in order
	for i := 1; i < len(finalTimestamps); i++ {
		if !finalTimestamps[i].After(finalTimestamps[i-1]) {
			t.Errorf("rotation timestamp %d should be after timestamp %d", i, i-1)
		}
	}
}

func TestTLSBuilder_CustomConfigs(t *testing.T) {
	builder := NewFakeTLSBuilder()

	// Set custom server config
	customServerConfig := &tls.Config{
		MinVersion: tls.VersionTLS13,
		MaxVersion: tls.VersionTLS13,
	}
	builder.SetServerConfig(customServerConfig)

	// Set custom client config
	customClientConfig := &tls.Config{
		MinVersion:         tls.VersionTLS13,
		MaxVersion:         tls.VersionTLS13,
		InsecureSkipVerify: true, // For testing only
	}
	builder.SetClientConfig(customClientConfig)

	// Test server config
	serverConfig := builder.ServerConfig()
	if serverConfig.MinVersion != tls.VersionTLS13 {
		t.Errorf("expected server MinVersion %d, got %d", tls.VersionTLS13, serverConfig.MinVersion)
	}

	// Test client config
	clientConfig := builder.ClientConfig()
	if clientConfig.MinVersion != tls.VersionTLS13 {
		t.Errorf("expected client MinVersion %d, got %d", tls.VersionTLS13, clientConfig.MinVersion)
	}
	if !clientConfig.InsecureSkipVerify {
		t.Error("expected client config to have InsecureSkipVerify=true")
	}
}

// RunTLSBuilderContractTests runs contract tests against any TLSBuilder implementation
func RunTLSBuilderContractTests(t *testing.T, builder TLSBuilder) {
	t.Run("ServerConfig", func(t *testing.T) {
		config := builder.ServerConfig()
		if config == nil {
			t.Error("ServerConfig() should not return nil")
		}
		
		// Contract: should have reasonable security settings
		if config.MinVersion < tls.VersionTLS12 {
			t.Errorf("ServerConfig() should enforce TLS 1.2 minimum, got %d", config.MinVersion)
		}
	})

	t.Run("ClientConfig", func(t *testing.T) {
		config := builder.ClientConfig()
		if config == nil {
			t.Error("ClientConfig() should not return nil")
		}
		
		// Contract: should have reasonable security settings
		if config.MinVersion < tls.VersionTLS12 {
			t.Errorf("ClientConfig() should enforce TLS 1.2 minimum, got %d", config.MinVersion)
		}
	})

	t.Run("RepeatedCalls", func(t *testing.T) {
		// Contract: should not panic on repeated calls
		config1 := builder.ServerConfig()
		config2 := builder.ServerConfig()
		
		if config1 == nil || config2 == nil {
			t.Error("repeated calls should not return nil")
		}
	})
}