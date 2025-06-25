package config

import (
	"os"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := New()

	if cfg.Port != "9100" {
		t.Errorf("Expected default port 9100, got %s", cfg.Port)
	}

	if cfg.MetricsPath != "/metrics" {
		t.Errorf("Expected default metrics path /metrics, got %s", cfg.MetricsPath)
	}

	if cfg.CollectInterval != 30*time.Second {
		t.Errorf("Expected default collect interval 30s, got %s", cfg.CollectInterval)
	}

	if cfg.LogLevel != "info" {
		t.Errorf("Expected default log level info, got %s", cfg.LogLevel)
	}
}

func TestEnvironmentOverrides(t *testing.T) {
	// Set environment variables
	os.Setenv("PORT", "8080")
	os.Setenv("METRICS_PATH", "/custom")
	os.Setenv("LOG_LEVEL", "debug")
	defer func() {
		os.Unsetenv("PORT")
		os.Unsetenv("METRICS_PATH")
		os.Unsetenv("LOG_LEVEL")
	}()

	cfg := New()

	if cfg.Port != "8080" {
		t.Errorf("Expected port 8080 from env, got %s", cfg.Port)
	}

	if cfg.MetricsPath != "/custom" {
		t.Errorf("Expected metrics path /custom from env, got %s", cfg.MetricsPath)
	}

	if cfg.LogLevel != "debug" {
		t.Errorf("Expected log level debug from env, got %s", cfg.LogLevel)
	}
}

func TestCollectIntervalParsing(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected time.Duration
	}{
		{"Duration format", "45s", 45 * time.Second},
		{"Seconds format", "60", 60 * time.Second},
		{"Invalid format", "invalid", 30 * time.Second}, // Should fall back to default
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("COLLECT_INTERVAL", tt.envValue)
			defer os.Unsetenv("COLLECT_INTERVAL")

			cfg := New()
			if cfg.CollectInterval != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, cfg.CollectInterval)
			}
		})
	}
}
