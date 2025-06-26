package main

import (
	"testing"

	"disk-health-exporter/internal/metrics"
)

func TestMetricsInitialization(t *testing.T) {
	// Test metrics initialization - this doesn't require config
	m := metrics.New()
	if m == nil {
		t.Error("Metrics should not be nil")
	}
}
