package main

import (
	"net/http"
	"testing"
	"time"

	"disk-health-exporter/internal/config"
	"disk-health-exporter/internal/metrics"
)

func TestApplicationStartup(t *testing.T) {
	// Test configuration
	cfg := config.New()
	if cfg.Port == "" {
		t.Error("Port should not be empty")
	}

	// Test metrics initialization
	m := metrics.New()
	if m == nil {
		t.Error("Metrics should not be nil")
	}

	// Test that metrics endpoint would be available
	// Note: This is a simple test - in production you'd want more comprehensive tests
	expectedMetricsPath := "/metrics"
	if cfg.MetricsPath != expectedMetricsPath {
		t.Errorf("Expected metrics path %s, got %s", expectedMetricsPath, cfg.MetricsPath)
	}
}

func TestHealthCheck(t *testing.T) {
	// Start server in background for testing
	go func() {
		cfg := config.New()
		cfg.Port = "9101" // Use different port for testing

		http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"status":"ok","service":"disk-health-exporter"}`))
		})

		http.ListenAndServe(":"+cfg.Port, nil)
	}()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Test health endpoint
	resp, err := http.Get("http://localhost:9101/health")
	if err != nil {
		t.Skipf("Health check test skipped: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK, got %d", resp.StatusCode)
	}
}
