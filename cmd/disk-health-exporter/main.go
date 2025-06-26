package main

import (
	"fmt"
	"log"
	"net/http"

	"disk-health-exporter/internal/collector"
	"disk-health-exporter/internal/config"
	"disk-health-exporter/internal/metrics"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	// Load configuration
	cfg := config.New()

	log.Println("Starting Disk Health Prometheus Exporter...")

	// Initialize metrics
	m := metrics.New()

	// Create collector with configuration
	c := collector.NewWithConfig(m, cfg.CollectInterval, cfg)

	// Start metrics collection in background
	go c.Start()

	// Set up HTTP handlers
	setupHTTPHandlers(cfg)

	// Start HTTP server
	log.Printf("Starting HTTP server on port %s", cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, nil))
}

// setupHTTPHandlers configures HTTP routes
func setupHTTPHandlers(cfg *config.Config) {
	// Metrics endpoint
	http.Handle(cfg.MetricsPath, promhttp.Handler())

	// Root endpoint with basic info
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `
		<html>
		<head><title>Disk Health Exporter</title></head>
		<body>
		<h1>Disk Health Prometheus Exporter</h1>
		<p><a href="%s">Metrics</a></p>
		<p>Version: 1.0.0</p>
		<p>Collect Interval: %s</p>
		</body>
		</html>
		`, cfg.MetricsPath, cfg.CollectInterval)
	})

	// Health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok","service":"disk-health-exporter"}`)
	})
}
