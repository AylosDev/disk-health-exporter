package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds the application configuration
type Config struct {
	Port            string
	MetricsPath     string
	CollectInterval time.Duration
	LogLevel        string
}

// New creates a new configuration with default values
func New() *Config {
	return &Config{
		Port:            getEnv("PORT", "9100"),
		MetricsPath:     getEnv("METRICS_PATH", "/metrics"),
		CollectInterval: getEnvDuration("COLLECT_INTERVAL", 30*time.Second),
		LogLevel:        getEnv("LOG_LEVEL", "info"),
	}
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvDuration gets a duration environment variable with a default value
func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
		// Try parsing as seconds
		if seconds, err := strconv.Atoi(value); err == nil {
			return time.Duration(seconds) * time.Second
		}
	}
	return defaultValue
}
