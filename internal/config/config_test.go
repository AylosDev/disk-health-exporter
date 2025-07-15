package config

import (
	"flag"
	"os"
	"strings"
	"testing"
	"time"
)

func TestConfigFromFlags(t *testing.T) {
	t.Helper()

	// Reset flag.CommandLine for testing
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	// Mock command line arguments
	os.Args = []string{"cmd", "-port", "8080", "-metrics-path", "/test-metrics", "-collect-interval", "45s", "-log-level", "debug"}

	config := New("test-version")

	tests := []struct {
		name     string
		got      interface{}
		expected interface{}
	}{
		{"port", config.Port, "8080"},
		{"metrics path", config.MetricsPath, "/test-metrics"},
		{"collect interval", config.CollectInterval, 45 * time.Second},
		{"log level", config.LogLevel, "debug"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, tt.got)
			}
		})
	}
}

func TestConfigFromEnvironmentFallback(t *testing.T) {
	t.Helper()

	// Reset flag.CommandLine for testing
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	// Clear any existing environment variables and set test ones
	envVars := []string{"PORT", "METRICS_PATH", "COLLECT_INTERVAL", "LOG_LEVEL"}
	for _, env := range envVars {
		os.Unsetenv(env)
	}

	os.Setenv("PORT", "7070")
	os.Setenv("METRICS_PATH", "/env-metrics")
	os.Setenv("COLLECT_INTERVAL", "90s")
	os.Setenv("LOG_LEVEL", "warn")

	// Cleanup after test
	defer func() {
		for _, env := range envVars {
			os.Unsetenv(env)
		}
	}()

	defer func() {
		// Clean up environment variables
		os.Unsetenv("PORT")
		os.Unsetenv("METRICS_PATH")
		os.Unsetenv("COLLECT_INTERVAL")
		os.Unsetenv("LOG_LEVEL")
	}()

	// Mock command line arguments with no flags
	os.Args = []string{"cmd"}

	config := New("test-version")

	if config.Port != "7070" {
		t.Errorf("Expected port 7070 from env, got %s", config.Port)
	}

	if config.MetricsPath != "/env-metrics" {
		t.Errorf("Expected metrics path /env-metrics from env, got %s", config.MetricsPath)
	}

	if config.CollectInterval != 90*time.Second {
		t.Errorf("Expected collect interval 90s from env, got %v", config.CollectInterval)
	}

	if config.LogLevel != "warn" {
		t.Errorf("Expected log level warn from env, got %s", config.LogLevel)
	}
}

func TestConfigDefaults(t *testing.T) {
	// Reset flag.CommandLine for testing
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	// Clear environment variables
	os.Unsetenv("PORT")
	os.Unsetenv("METRICS_PATH")
	os.Unsetenv("COLLECT_INTERVAL")
	os.Unsetenv("LOG_LEVEL")

	defer func() {
		// Clean up environment variables
		os.Unsetenv("PORT")
		os.Unsetenv("METRICS_PATH")
		os.Unsetenv("COLLECT_INTERVAL")
		os.Unsetenv("LOG_LEVEL")
	}()

	// Mock command line arguments with no flags
	os.Args = []string{"cmd"}

	config := New("test-version")

	if config.Port != "9100" {
		t.Errorf("Expected default port 9100, got %s", config.Port)
	}

	if config.MetricsPath != "/metrics" {
		t.Errorf("Expected default metrics path /metrics, got %s", config.MetricsPath)
	}

	if config.CollectInterval != 30*time.Second {
		t.Errorf("Expected default collect interval 30s, got %v", config.CollectInterval)
	}

	if config.LogLevel != "info" {
		t.Errorf("Expected default log level info, got %s", config.LogLevel)
	}
}

func TestFlagsPriorityOverEnvironment(t *testing.T) {
	// Reset flag.CommandLine for testing
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	// Set environment variables
	os.Setenv("PORT", "5000")
	os.Setenv("LOG_LEVEL", "error")

	defer func() {
		// Clean up environment variables
		os.Unsetenv("PORT")
		os.Unsetenv("LOG_LEVEL")
	}()

	// Mock command line arguments with flags that should override env vars
	os.Args = []string{"cmd", "-port", "6000", "-log-level", "debug"}

	config := New("test-version")

	if config.Port != "6000" {
		t.Errorf("Expected port 6000 from flag (not 5000 from env), got %s", config.Port)
	}

	if config.LogLevel != "debug" {
		t.Errorf("Expected log level debug from flag (not error from env), got %s", config.LogLevel)
	}
}

func TestGetEnvDuration(t *testing.T) {
	testCases := []struct {
		envValue string
		expected time.Duration
		name     string
	}{
		{"30s", 30 * time.Second, "duration string"},
		{"60", 60 * time.Second, "seconds as integer"},
		{"2m", 2 * time.Minute, "minutes"},
		{"1h", 1 * time.Hour, "hours"},
		{"invalid", 30 * time.Second, "invalid value falls back to default"},
		{"", 30 * time.Second, "empty value falls back to default"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			os.Setenv("TEST_DURATION", tc.envValue)
			defer os.Unsetenv("TEST_DURATION")

			result := getEnvDuration("TEST_DURATION", 30*time.Second)
			if result != tc.expected {
				t.Errorf("Expected %v, got %v for input '%s'", tc.expected, result, tc.envValue)
			}
		})
	}
}

func TestConfigValidation(t *testing.T) {
	t.Helper()

	tests := []struct {
		name        string
		config      Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config",
			config: Config{
				Port:            "9100",
				MetricsPath:     "/metrics",
				CollectInterval: 30 * time.Second,
				LogLevel:        "info",
			},
			expectError: false,
		},
		{
			name: "empty port",
			config: Config{
				Port:            "",
				MetricsPath:     "/metrics",
				CollectInterval: 30 * time.Second,
				LogLevel:        "info",
			},
			expectError: true,
			errorMsg:    "port cannot be empty",
		},
		{
			name: "invalid port - non-numeric",
			config: Config{
				Port:            "abc",
				MetricsPath:     "/metrics",
				CollectInterval: 30 * time.Second,
				LogLevel:        "info",
			},
			expectError: true,
			errorMsg:    "port must be a valid number",
		},
		{
			name: "invalid port - out of range",
			config: Config{
				Port:            "99999",
				MetricsPath:     "/metrics",
				CollectInterval: 30 * time.Second,
				LogLevel:        "info",
			},
			expectError: true,
			errorMsg:    "port must be between 1 and 65535",
		},
		{
			name: "empty metrics path",
			config: Config{
				Port:            "9100",
				MetricsPath:     "",
				CollectInterval: 30 * time.Second,
				LogLevel:        "info",
			},
			expectError: true,
			errorMsg:    "metrics path cannot be empty",
		},
		{
			name: "metrics path without leading slash",
			config: Config{
				Port:            "9100",
				MetricsPath:     "metrics",
				CollectInterval: 30 * time.Second,
				LogLevel:        "info",
			},
			expectError: true,
			errorMsg:    "metrics path must start with '/'",
		},
		{
			name: "zero collect interval",
			config: Config{
				Port:            "9100",
				MetricsPath:     "/metrics",
				CollectInterval: 0,
				LogLevel:        "info",
			},
			expectError: true,
			errorMsg:    "collect interval must be positive",
		},
		{
			name: "collect interval too small",
			config: Config{
				Port:            "9100",
				MetricsPath:     "/metrics",
				CollectInterval: 500 * time.Millisecond,
				LogLevel:        "info",
			},
			expectError: true,
			errorMsg:    "collect interval must be at least 1 second",
		},
		{
			name: "invalid log level",
			config: Config{
				Port:            "9100",
				MetricsPath:     "/metrics",
				CollectInterval: 30 * time.Second,
				LogLevel:        "invalid",
			},
			expectError: true,
			errorMsg:    "invalid log level",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error message to contain %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}
