// Package config provides configuration management for the disk health exporter.
// It supports configuration through command-line flags and environment variables.
package config

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds the application configuration
type Config struct {
	Version         string // Version of the application, set at build time
	Port            string
	MetricsPath     string
	CollectInterval time.Duration
	LogLevel        string
	TargetDisks     string   // Comma-separated list of specific disks to monitor (e.g., "/dev/sda,/dev/nvme0n1")
	IgnorePatterns  []string // Internal use: patterns to ignore (loop devices, etc.)
}

// New creates a new configuration from command-line flags and validates it.
func New(version string) *Config {
	var (
		port            = flag.String("port", getEnv("PORT", "9100"), "Port to listen on")
		metricsPath     = flag.String("metrics-path", getEnv("METRICS_PATH", "/metrics"), "Path to expose metrics")
		collectInterval = flag.Duration("collect-interval", getEnvDuration("COLLECT_INTERVAL", 30*time.Second), "Interval between disk health collections")
		logLevel        = flag.String("log-level", getEnv("LOG_LEVEL", "info"), "Log level (debug, info, warn, error)")
		targetDisks     = flag.String("target-disks", getEnv("TARGET_DISKS", ""), "Comma-separated list of specific disks to monitor (e.g., '/dev/sda,/dev/nvme0n1'). If empty, all detected disks are monitored.")
		showHelp        = flag.Bool("help", false, "Show help message")
		showVersion     = flag.Bool("version", false, "Show version information")
	)

	flag.Parse()

	if *showHelp {
		PrintUsage()
		os.Exit(0)
	}
	if *showVersion {
		PrintVersion(version)
		os.Exit(0)
	}

	// Internal ignore patterns for devices we should skip
	ignorePatterns := []string{
		"/dev/loop", // Loop devices
		"/dev/ram",  // RAM disks
		"/dev/dm-",  // Device mapper (handled by underlying devices)
	}

	cfg := &Config{
		Version:         version,
		Port:            *port,
		MetricsPath:     *metricsPath,
		CollectInterval: *collectInterval,
		LogLevel:        *logLevel,
		TargetDisks:     *targetDisks,
		IgnorePatterns:  ignorePatterns,
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "Configuration error: %v\n", err)
		os.Exit(1)
	}

	return cfg
}

// PrintUsage prints usage information to stdout.
func PrintUsage() {
	fmt.Printf("Disk Health Prometheus Exporter\n\n")
	fmt.Printf("Usage: %s [options]\n\n", os.Args[0])
	fmt.Printf("Options:\n")
	flag.PrintDefaults()
	fmt.Printf("\nEnvironment Variables (used as fallback if flags not provided):\n")
	fmt.Printf("  PORT             - Port to listen on (default: 9100)\n")
	fmt.Printf("  METRICS_PATH     - Path to expose metrics (default: /metrics)\n")
	fmt.Printf("  COLLECT_INTERVAL - Collection interval (default: 30s)\n")
	fmt.Printf("  LOG_LEVEL        - Log level (default: info)\n")
	fmt.Printf("  TARGET_DISKS     - Comma-separated list of disks to monitor\n")
	fmt.Printf("\nExamples:\n")
	fmt.Printf("  %s -port 8080 -collect-interval 60s\n", os.Args[0])
	fmt.Printf("  %s -metrics-path /health -log-level debug\n", os.Args[0])
	fmt.Printf("  %s -target-disks '/dev/sda,/dev/nvme0n1'\n", os.Args[0])
}

// PrintVersion prints version information to stdout.
func PrintVersion(version string) {
	fmt.Printf("Disk Health Prometheus Exporter\n")
	fmt.Print(version)
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvDuration gets a duration environment variable with a default value.
// It first tries to parse as a duration string (e.g., "30s", "5m"),
// then falls back to parsing as seconds if that fails.
func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	// Try parsing as duration first
	if duration, err := time.ParseDuration(value); err == nil {
		return duration
	}

	// Try parsing as seconds
	if seconds, err := strconv.Atoi(value); err == nil {
		return time.Duration(seconds) * time.Second
	}

	// If all parsing fails, return default
	return defaultValue
}

// Validate validates the configuration values and returns an error if any are invalid.
func (c *Config) Validate() error {
	if c.Port == "" {
		return fmt.Errorf("port cannot be empty")
	}

	// Validate port is numeric and in valid range
	if port, err := strconv.Atoi(c.Port); err != nil {
		return fmt.Errorf("port must be a valid number: %w", err)
	} else if port < 1 || port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535, got %d", port)
	}

	if c.MetricsPath == "" {
		return fmt.Errorf("metrics path cannot be empty")
	}

	if !strings.HasPrefix(c.MetricsPath, "/") {
		return fmt.Errorf("metrics path must start with '/', got %s", c.MetricsPath)
	}

	if c.CollectInterval <= 0 {
		return fmt.Errorf("collect interval must be positive, got %v", c.CollectInterval)
	}

	if c.CollectInterval < time.Second {
		return fmt.Errorf("collect interval must be at least 1 second, got %v", c.CollectInterval)
	}

	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}

	if !validLogLevels[strings.ToLower(c.LogLevel)] {
		return fmt.Errorf("invalid log level %s, must be one of: debug, info, warn, error", c.LogLevel)
	}

	return nil
}
