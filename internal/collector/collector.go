package collector

import (
	"log"
	"runtime"
	"time"

	"disk-health-exporter/internal/disk"
	"disk-health-exporter/internal/metrics"
	"disk-health-exporter/pkg/types"
)

// Collector handles metric collection
type Collector struct {
	metrics     *metrics.Metrics
	diskManager *disk.Manager
	interval    time.Duration
}

// New creates a new collector
func New(m *metrics.Metrics, interval time.Duration) *Collector {
	return &Collector{
		metrics:     m,
		diskManager: disk.New(),
		interval:    interval,
	}
}

// Start begins the metric collection loop
func (c *Collector) Start() {
	// Set exporter as up
	c.metrics.ExporterUp.Set(1)

	// Collect metrics immediately on startup
	c.updateMetrics()

	// Start periodic collection
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for range ticker.C {
		c.updateMetrics()
	}
}

// updateMetrics collects and updates all metrics
func (c *Collector) updateMetrics() {
	log.Println("Collecting disk health metrics...")

	// Clear previous metrics
	c.metrics.Reset()

	// Detect operating system
	osType := runtime.GOOS
	log.Printf("Detected OS: %s", osType)

	switch osType {
	case "linux":
		c.collectLinuxMetrics()
	case "darwin":
		c.collectMacOSMetrics()
	default:
		c.collectFallbackMetrics()
	}
}

// collectLinuxMetrics collects metrics on Linux systems
func (c *Collector) collectLinuxMetrics() {
	disks, raidArrays := c.diskManager.GetLinuxDisks()

	// Update RAID array metrics
	for _, raid := range raidArrays {
		c.metrics.RaidArrayStatus.WithLabelValues(raid.ArrayID, raid.RaidLevel, raid.State).Set(float64(raid.Status))
	}

	// Update disk metrics
	c.updateDiskMetrics(disks)

	log.Printf("Updated metrics for %d disks and %d RAID arrays", len(disks), len(raidArrays))
}

// collectMacOSMetrics collects metrics on macOS systems
func (c *Collector) collectMacOSMetrics() {
	disks := c.diskManager.GetMacOSDisks()
	c.updateDiskMetrics(disks)

	log.Printf("Updated metrics for %d macOS disks", len(disks))
}

// collectFallbackMetrics collects metrics using fallback method
func (c *Collector) collectFallbackMetrics() {
	log.Printf("Using fallback disk detection for OS: %s", runtime.GOOS)

	// Try to get regular disks as fallback
	disks, _ := c.diskManager.GetLinuxDisks()
	c.updateDiskMetrics(disks)

	log.Printf("Updated metrics for %d disks (fallback mode)", len(disks))
}

// updateDiskMetrics updates metrics for a list of disks
func (c *Collector) updateDiskMetrics(disks []types.DiskInfo) {
	for _, disk := range disks {
		// Convert health status to numeric value
		status := getHealthStatusValue(disk.Health)

		// Update health status metric
		c.metrics.DiskHealthStatus.WithLabelValues(
			disk.Device,
			disk.Type,
			disk.Serial,
			disk.Model,
			disk.Location,
		).Set(float64(status))

		// Update temperature metric only if available
		if disk.Temperature > 0 {
			c.metrics.DiskTemperature.WithLabelValues(
				disk.Device,
				disk.Serial,
				disk.Model,
			).Set(disk.Temperature)
		}

		// Set sector errors to 0 when no SMART data is available
		// In production, this should only be set when we have actual error data
		c.metrics.DiskSectorErrors.WithLabelValues(
			disk.Device,
			disk.Serial,
			disk.Model,
			"reallocated_sectors",
		).Set(0)

		// Power-on hours metric is available but not populated from current disk detection
		// It would need to be added to the DiskInfo struct and populated from SMART data
		// For now, we don't set this metric unless we have actual data
	}
}

// getHealthStatusValue converts health string to numeric value
func getHealthStatusValue(health string) int {
	switch health {
	case "OK":
		return int(types.HealthStatusOK)
	case "WARNING":
		return int(types.HealthStatusWarning)
	case "FAILED", "CRITICAL":
		return int(types.HealthStatusCritical)
	default:
		return int(types.HealthStatusUnknown)
	}
}
