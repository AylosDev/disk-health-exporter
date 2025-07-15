// Package main provides the disk health Prometheus exporter binary.
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"disk-health-exporter/internal/collector"
	"disk-health-exporter/internal/config"
	"disk-health-exporter/internal/metrics"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Build-time variables set via ldflags during compilation.
// These variables are populated by the build system using -X linker flags.
var (
	version   = "dev"     // Application version, set at build time
	commit    = "unknown" // Git commit hash, set at build time
	buildTime = "unknown" // Build timestamp, set at build time
	buildBy   = "unknown" // Builder identifier, set at build time
)

// getBuildInfo returns a formatted string with build and runtime information.
func getBuildInfo() string {
	return fmt.Sprintf("go-dev-deploys %s\nCommit %s\nBuild Time: %s\nBuild By %s\nGo Version %s\nPlatform %s/%s\n",
		version, commit, buildTime, buildBy, runtime.Version(), runtime.GOOS, runtime.GOARCH)
}

func main() {
	// Load configuration
	cfg := config.New(getBuildInfo())

	log.Println("Starting Disk Health Prometheus Exporter...")

	// Initialize metrics
	m := metrics.New()

	// Create collector with configuration
	c := collector.NewWithConfig(m, cfg.CollectInterval, cfg)

	// Set up context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start metrics collection in background
	go c.Start(ctx)

	// Set up HTTP server
	server := &http.Server{
		Addr:         ":" + cfg.Port,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Set up HTTP handlers
	setupHTTPHandlers(cfg)

	// Start HTTP server in a goroutine
	go func() {
		log.Printf("Starting HTTP server on port %s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Cancel context to stop collector
	cancel()

	// Give server 30 seconds to shutdown gracefully
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

// setupHTTPHandlers configures HTTP routes with proper error handling.
func setupHTTPHandlers(cfg *config.Config) {
	// Metrics endpoint
	http.Handle(cfg.MetricsPath, promhttp.Handler())

	// Root endpoint with basic info
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, err := fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head>
	<title>Disk Health Exporter</title>
	<meta charset="utf-8">
</head>
<body>
	<h1>Disk Health Prometheus Exporter</h1>
	<p><a href="%s">Metrics</a></p>
	<p>Version: %s (#%s)</p>
	<p>Collect Interval: %s</p>
</body>
</html>`, cfg.MetricsPath, version, commit, cfg.CollectInterval)

		if err != nil {
			log.Printf("Error writing response: %v", err)
		}
	})

	// Health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, err := fmt.Fprintf(w, `{"status":"ok","service":"disk-health-exporter"}`)

		if err != nil {
			log.Printf("Error writing health check response: %v", err)
		}
	})
}
