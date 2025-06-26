package config

import (
	"flag"
	"os"
	"testing"
	"time"
)

func TestConfigFromFlags(t *testing.T) {
	// Reset flag.CommandLine for testing
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	// Mock command line arguments
	os.Args = []string{"cmd", "-port", "8080", "-metrics-path", "/test-metrics", "-collect-interval", "45s", "-log-level", "debug"}

	config := New()

	if config.Port != "8080" {
		t.Errorf("Expected port 8080, got %s", config.Port)
	}

	if config.MetricsPath != "/test-metrics" {
		t.Errorf("Expected metrics path /test-metrics, got %s", config.MetricsPath)
	}

	if config.CollectInterval != 45*time.Second {
		t.Errorf("Expected collect interval 45s, got %v", config.CollectInterval)
	}

	if config.LogLevel != "debug" {
		t.Errorf("Expected log level debug, got %s", config.LogLevel)
	}
}

func TestConfigFromEnvironmentFallback(t *testing.T) {
	// Reset flag.CommandLine for testing
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	// Clear any existing environment variables and set test ones
	os.Unsetenv("PORT")
	os.Unsetenv("METRICS_PATH")
	os.Unsetenv("COLLECT_INTERVAL")
	os.Unsetenv("LOG_LEVEL")

	os.Setenv("PORT", "7070")
	os.Setenv("METRICS_PATH", "/env-metrics")
	os.Setenv("COLLECT_INTERVAL", "90s")
	os.Setenv("LOG_LEVEL", "warn")

	defer func() {
		// Clean up environment variables
		os.Unsetenv("PORT")
		os.Unsetenv("METRICS_PATH")
		os.Unsetenv("COLLECT_INTERVAL")
		os.Unsetenv("LOG_LEVEL")
	}()

	// Mock command line arguments with no flags
	os.Args = []string{"cmd"}

	config := New()

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

	config := New()

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

	config := New()

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
