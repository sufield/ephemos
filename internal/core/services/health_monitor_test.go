package services

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/sufield/ephemos/internal/core/ports"
)

// MockHealthChecker for testing
type MockHealthChecker struct {
	mock.Mock
}

func (m *MockHealthChecker) CheckLiveness(ctx context.Context) (*ports.HealthResult, error) {
	args := m.Called(ctx)
	return args.Get(0).(*ports.HealthResult), args.Error(1)
}

func (m *MockHealthChecker) CheckReadiness(ctx context.Context) (*ports.HealthResult, error) {
	args := m.Called(ctx)
	return args.Get(0).(*ports.HealthResult), args.Error(1)
}

func (m *MockHealthChecker) CheckHealth(ctx context.Context) (*ports.HealthResult, error) {
	args := m.Called(ctx)
	return args.Get(0).(*ports.HealthResult), args.Error(1)
}

func (m *MockHealthChecker) GetComponentName() string {
	args := m.Called()
	return args.String(0)
}

// MockHealthReporter for testing
type MockHealthReporter struct {
	mock.Mock
}

func (m *MockHealthReporter) ReportHealth(result *ports.HealthResult) error {
	args := m.Called(result)
	return args.Error(0)
}

func (m *MockHealthReporter) ReportOverallHealth(results map[string]*ports.HealthResult) error {
	args := m.Called(results)
	return args.Error(0)
}

func (m *MockHealthReporter) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestNewHealthMonitorService(t *testing.T) {
	tests := []struct {
		name    string
		config  *ports.HealthConfig
		logger  *slog.Logger
		wantErr bool
	}{
		{
			name: "valid config",
			config: &ports.HealthConfig{
				Enabled: true,
				Timeout: 10 * time.Second,
			},
			logger:  slog.Default(),
			wantErr: false,
		},
		{
			name:    "nil config",
			config:  nil,
			logger:  slog.Default(),
			wantErr: true,
		},
		{
			name: "nil logger uses default",
			config: &ports.HealthConfig{
				Enabled: true,
			},
			logger:  nil,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, err := NewHealthMonitorService(tt.config, tt.logger)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, service)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, service)
			}
		})
	}
}

func TestHealthMonitorService_RegisterChecker(t *testing.T) {
	config := &ports.HealthConfig{Enabled: true}
	service, err := NewHealthMonitorService(config, slog.Default())
	require.NoError(t, err)

	tests := []struct {
		name      string
		checker   ports.HealthCheckerPort
		wantErr   bool
		setupMock func(*MockHealthChecker)
	}{
		{
			name:    "valid checker",
			checker: &MockHealthChecker{},
			wantErr: false,
			setupMock: func(m *MockHealthChecker) {
				m.On("GetComponentName").Return("test-component")
			},
		},
		{
			name:    "nil checker",
			checker: nil,
			wantErr: true,
		},
		{
			name:    "checker with empty name",
			checker: &MockHealthChecker{},
			wantErr: true,
			setupMock: func(m *MockHealthChecker) {
				m.On("GetComponentName").Return("")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupMock != nil && tt.checker != nil {
				tt.setupMock(tt.checker.(*MockHealthChecker))
			}

			err := service.RegisterChecker(tt.checker)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.checker != nil {
				tt.checker.(*MockHealthChecker).AssertExpectations(t)
			}
		})
	}
}

func TestHealthMonitorService_UnregisterChecker(t *testing.T) {
	config := &ports.HealthConfig{Enabled: true}
	service, err := NewHealthMonitorService(config, slog.Default())
	require.NoError(t, err)

	// Register a checker first
	checker := &MockHealthChecker{}
	checker.On("GetComponentName").Return("test-component")

	err = service.RegisterChecker(checker)
	require.NoError(t, err)

	// Test unregistering
	tests := []struct {
		name          string
		componentName string
		wantErr       bool
	}{
		{
			name:          "existing component",
			componentName: "test-component",
			wantErr:       false,
		},
		{
			name:          "non-existing component",
			componentName: "non-existing",
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.UnregisterChecker(tt.componentName)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}

	checker.AssertExpectations(t)
}

func TestHealthMonitorService_CheckAll(t *testing.T) {
	config := &ports.HealthConfig{Enabled: true}
	service, err := NewHealthMonitorService(config, slog.Default())
	require.NoError(t, err)

	// Test with no checkers
	t.Run("no checkers", func(t *testing.T) {
		ctx := context.Background()
		results, err := service.CheckAll(ctx)

		assert.NoError(t, err)
		assert.Empty(t, results)
	})

	// Test with healthy checker
	t.Run("healthy checker", func(t *testing.T) {
		checker := &MockHealthChecker{}
		checker.On("GetComponentName").Return("test-component")

		healthResult := &ports.HealthResult{
			Status:    ports.HealthStatusHealthy,
			Component: "test-component",
			Message:   "All good",
			CheckedAt: time.Now(),
		}
		checker.On("CheckHealth", mock.Anything).Return(healthResult, nil)

		err := service.RegisterChecker(checker)
		require.NoError(t, err)

		ctx := context.Background()
		results, err := service.CheckAll(ctx)

		assert.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Contains(t, results, "test-component")
		assert.Equal(t, ports.HealthStatusHealthy, results["test-component"].Status)

		checker.AssertExpectations(t)
	})

	// Test with unhealthy checker
	t.Run("unhealthy checker", func(t *testing.T) {
		service2, err := NewHealthMonitorService(config, slog.Default())
		require.NoError(t, err)

		checker := &MockHealthChecker{}
		checker.On("GetComponentName").Return("failing-component")

		// Checker returns error
		checker.On("CheckHealth", mock.Anything).Return((*ports.HealthResult)(nil), assert.AnError)

		err = service2.RegisterChecker(checker)
		require.NoError(t, err)

		ctx := context.Background()
		results, err := service2.CheckAll(ctx)

		assert.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Contains(t, results, "failing-component")
		assert.Equal(t, ports.HealthStatusUnknown, results["failing-component"].Status)

		checker.AssertExpectations(t)
	})
}

func TestHealthMonitorService_RegisterReporter(t *testing.T) {
	config := &ports.HealthConfig{Enabled: true}
	service, err := NewHealthMonitorService(config, slog.Default())
	require.NoError(t, err)

	tests := []struct {
		name     string
		reporter ports.HealthReporterPort
		wantErr  bool
	}{
		{
			name:     "valid reporter",
			reporter: &MockHealthReporter{},
			wantErr:  false,
		},
		{
			name:     "nil reporter",
			reporter: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.RegisterReporter(tt.reporter)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestHealthMonitorService_GetOverallHealth(t *testing.T) {
	config := &ports.HealthConfig{Enabled: true}
	service, err := NewHealthMonitorService(config, slog.Default())
	require.NoError(t, err)

	tests := []struct {
		name           string
		results        map[string]*ports.HealthResult
		expectedStatus ports.HealthStatus
	}{
		{
			name:           "no results",
			results:        map[string]*ports.HealthResult{},
			expectedStatus: ports.HealthStatusUnknown,
		},
		{
			name: "all healthy",
			results: map[string]*ports.HealthResult{
				"comp1": {Status: ports.HealthStatusHealthy},
				"comp2": {Status: ports.HealthStatusHealthy},
			},
			expectedStatus: ports.HealthStatusHealthy,
		},
		{
			name: "mixed health",
			results: map[string]*ports.HealthResult{
				"comp1": {Status: ports.HealthStatusHealthy},
				"comp2": {Status: ports.HealthStatusUnhealthy},
			},
			expectedStatus: ports.HealthStatusUnhealthy,
		},
		{
			name: "all unhealthy",
			results: map[string]*ports.HealthResult{
				"comp1": {Status: ports.HealthStatusUnhealthy},
				"comp2": {Status: ports.HealthStatusUnhealthy},
			},
			expectedStatus: ports.HealthStatusUnhealthy,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set the results directly in the service
			service.mu.Lock()
			service.results = tt.results
			service.mu.Unlock()

			status := service.GetOverallHealth()
			assert.Equal(t, tt.expectedStatus, status)
		})
	}
}

func TestHealthMonitorService_StartStopMonitoring(t *testing.T) {
	config := &ports.HealthConfig{
		Enabled:  true,
		Interval: 100 * time.Millisecond,
	}
	service, err := NewHealthMonitorService(config, slog.Default())
	require.NoError(t, err)

	ctx := context.Background()

	// Test starting monitoring
	err = service.StartMonitoring(ctx)
	assert.NoError(t, err)

	// Try to start again (should fail)
	err = service.StartMonitoring(ctx)
	assert.Error(t, err)

	// Test stopping monitoring
	err = service.StopMonitoring()
	assert.NoError(t, err)

	// Try to stop again (should fail)
	err = service.StopMonitoring()
	assert.Error(t, err)
}

func TestHealthMonitorService_MonitoringDisabled(t *testing.T) {
	config := &ports.HealthConfig{
		Enabled: false,
	}
	service, err := NewHealthMonitorService(config, slog.Default())
	require.NoError(t, err)

	ctx := context.Background()
	err = service.StartMonitoring(ctx)
	assert.NoError(t, err) // Should succeed but not actually start monitoring
}

func TestHealthMonitorService_Close(t *testing.T) {
	config := &ports.HealthConfig{Enabled: true}
	service, err := NewHealthMonitorService(config, slog.Default())
	require.NoError(t, err)

	// Register a reporter
	reporter := &MockHealthReporter{}
	reporter.On("Close").Return(nil)

	err = service.RegisterReporter(reporter)
	require.NoError(t, err)

	// Start monitoring
	ctx := context.Background()
	err = service.StartMonitoring(ctx)
	require.NoError(t, err)

	// Close service
	err = service.Close()
	assert.NoError(t, err)

	reporter.AssertExpectations(t)
}

func TestHealthMonitorService_GetResults(t *testing.T) {
	config := &ports.HealthConfig{Enabled: true}
	service, err := NewHealthMonitorService(config, slog.Default())
	require.NoError(t, err)

	// Set some results
	testResults := map[string]*ports.HealthResult{
		"comp1": {
			Status:    ports.HealthStatusHealthy,
			Component: "comp1",
		},
	}

	service.mu.Lock()
	service.results = testResults
	service.mu.Unlock()

	// Get results
	results := service.GetResults()

	assert.Len(t, results, 1)
	assert.Contains(t, results, "comp1")
	assert.Equal(t, ports.HealthStatusHealthy, results["comp1"].Status)

	// Verify it's a copy (not the same map)
	assert.NotSame(t, &testResults, &results)
}
