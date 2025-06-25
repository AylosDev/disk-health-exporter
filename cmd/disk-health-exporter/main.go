package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"disk-health-exporter/internal/collector"
	"disk-health-exporter/internal/config"
	"disk-health-exporter/internal/health"
	"disk-health-exporter/internal/metrics"
	"disk-health-exporter/internal/system"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Build-time variables (set via -ldflags)
var (
	version   = "dev"
	commit    = "unknown"
	buildTime = "unknown"
	buildBy   = "unknown"
)

func main() {
	log.Println("Starting Disk Health Prometheus Exporter...")

	// Perform one-time system detection
	info := system.New()
	sysInfo := info.Detect()

	// Load configuration
	cfg := config.New()

	// Initialize metrics
	m := metrics.New()

	// Create collector with system info
	c := collector.New(m, cfg.CollectInterval, sysInfo)

	// Create health service
	healthService := health.New(c, sysInfo)

	// Start metrics collection in background
	go c.Start()

	// Set up HTTP handlers
	setupHTTPHandlers(cfg, sysInfo, healthService)

	// Start HTTP server
	log.Printf("Starting HTTP server on port %s", cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, nil))
}

// setupHTTPHandlers configures HTTP routes
func setupHTTPHandlers(cfg *config.Config, sysInfo *system.SystemInfo, healthService *health.Service) {
	// Metrics endpoint
	http.Handle(cfg.MetricsPath, promhttp.Handler())
	ver := fmt.Sprintf("v%s (%s)", version, commit)

	// Root endpoint with basic info
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `
		<html>
		<head><title>Disk Health Exporter</title></head>
		<body>
		<h1>Disk Health Prometheus Exporter</h1>
		<p><a href="%s">Metrics</a></p>
		<p><a href="/health">Health Check</a></p>
		<p><a href="/health/json">Health JSON</a></p>
		<p>Version: %s</p>
		<p>Collect Interval: %s</p>
		<h3>System Information</h3>
		<p>Platform: %s</p>
		<p>SMART Support: %v</p>
		<p>RAID Support: %v</p>
		</body>
		</html>
		`, cfg.MetricsPath, ver, cfg.CollectInterval, sysInfo.Platform, sysInfo.CanMonitorSMART(), sysInfo.CanMonitorRAID())
	})

	// Basic health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok","service":"disk-health-exporter"}`)
	})

	// Detailed JSON health endpoint
	http.HandleFunc("/health/json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Get current health data
		healthData := healthService.GetHealthData()

		// Convert to JSON
		jsonData, err := json.MarshalIndent(healthData, "", "  ")
		if err != nil {
			http.Error(w, "Failed to generate JSON", http.StatusInternalServerError)
			return
		}

		w.Write(jsonData)
	})
}
