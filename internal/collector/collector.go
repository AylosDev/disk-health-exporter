package collector

import (
	"log"
	"runtime"
	"time"

	"disk-health-exporter/internal/config"
	"disk-health-exporter/internal/disk"
	"disk-health-exporter/internal/metrics"
	"disk-health-exporter/internal/utils"
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

// NewWithConfig creates a new collector with configuration
func NewWithConfig(m *metrics.Metrics, interval time.Duration, cfg *config.Config) *Collector {
	return &Collector{
		metrics:     m,
		diskManager: disk.NewWithConfig(cfg.TargetDisks, cfg.IgnorePatterns),
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

// updateToolMetrics updates metrics about available tools
// boolToFloat converts boolean to float64 for metrics
func boolToFloat(b bool) float64 {
	if b {
		return 1.0
	}
	return 0.0
}

// collectLinuxMetrics collects metrics on Linux systems
func (c *Collector) collectLinuxMetrics() {
	disks, raidArrays := c.diskManager.GetDisks()

	// Update RAID array metrics with comprehensive data
	for _, raid := range raidArrays {
		c.metrics.RaidArrayStatus.WithLabelValues(
			raid.ArrayID,
			raid.RaidLevel,
			raid.State,
			raid.Type,
			raid.Controller,
		).Set(float64(raid.Status))

		// Additional RAID metrics
		if raid.Size > 0 {
			c.metrics.RaidArraySize.WithLabelValues(
				raid.ArrayID,
				raid.RaidLevel,
				raid.Type,
			).Set(float64(raid.Size))
		}

		if raid.UsedSize > 0 {
			c.metrics.RaidArrayUsedSize.WithLabelValues(
				raid.ArrayID,
				raid.RaidLevel,
				raid.Type,
			).Set(float64(raid.UsedSize))
		}

		if raid.NumDrives > 0 {
			c.metrics.RaidArrayNumDrives.WithLabelValues(
				raid.ArrayID,
				raid.RaidLevel,
				raid.Type,
			).Set(float64(raid.NumDrives))
		}

		if raid.NumActiveDrives > 0 {
			c.metrics.RaidArrayNumActiveDrives.WithLabelValues(
				raid.ArrayID,
				raid.RaidLevel,
				raid.Type,
			).Set(float64(raid.NumActiveDrives))
		}

		if raid.NumSpareDrives > 0 {
			c.metrics.RaidArrayNumSpareDrives.WithLabelValues(
				raid.ArrayID,
				raid.RaidLevel,
				raid.Type,
			).Set(float64(raid.NumSpareDrives))
		}

		if raid.NumFailedDrives > 0 {
			c.metrics.RaidArrayNumFailedDrives.WithLabelValues(
				raid.ArrayID,
				raid.RaidLevel,
				raid.Type,
			).Set(float64(raid.NumFailedDrives))
		}

		if raid.RebuildProgress > 0 {
			c.metrics.RaidArrayRebuildProgress.WithLabelValues(
				raid.ArrayID,
				raid.RaidLevel,
				raid.Type,
			).Set(float64(raid.RebuildProgress))
		}

		if raid.ScrubProgress > 0 {
			c.metrics.RaidArrayScrubProgress.WithLabelValues(
				raid.ArrayID,
				raid.RaidLevel,
				raid.Type,
			).Set(float64(raid.ScrubProgress))
		}

		// Update battery metrics if available
		if raid.Battery != nil {
			utils.UpdateBatteryMetrics(raid.Battery, c.metrics)
		}
	}

	// Update comprehensive disk metrics
	c.updateComprehensiveDiskMetrics(disks)

	log.Printf("Updated metrics for %d disks and %d RAID arrays", len(disks), len(raidArrays))
}

// collectMacOSMetrics collects metrics on macOS systems
func (c *Collector) collectMacOSMetrics() {
	disks, _ := c.diskManager.GetDisks()
	c.updateComprehensiveDiskMetrics(disks)

	log.Printf("Updated metrics for %d macOS disks", len(disks))
}

// collectFallbackMetrics collects metrics using fallback method
func (c *Collector) collectFallbackMetrics() {
	log.Printf("Using fallback disk detection for OS: %s", runtime.GOOS)

	// Try to get regular disks as fallback
	disks, _ := c.diskManager.GetDisks()
	c.updateComprehensiveDiskMetrics(disks)

	log.Printf("Updated metrics for %d disks (fallback mode)", len(disks))
}

// updateComprehensiveDiskMetrics updates comprehensive metrics for a list of disks
func (c *Collector) updateComprehensiveDiskMetrics(disks []types.DiskInfo) {
	for _, disk := range disks {
		// Convert health status to numeric value using the proper utility function
		status := utils.GetHealthStatusValue(disk.Health)

		// Basic health status metric with enhanced labels
		c.metrics.DiskHealthStatus.WithLabelValues(
			disk.Device,
			disk.Type,
			disk.Serial,
			disk.Model,
			disk.Location,
			disk.Interface,
		).Set(float64(status))

		// Temperature metrics
		if disk.Temperature > 0 {
			c.metrics.DiskTemperature.WithLabelValues(
				disk.Device,
				disk.Serial,
				disk.Model,
				disk.Interface,
			).Set(disk.Temperature)
		}

		if disk.DriveTemperatureMax > 0 {
			c.metrics.DiskTemperatureMax.WithLabelValues(
				disk.Device,
				disk.Serial,
				disk.Model,
			).Set(disk.DriveTemperatureMax)
		}

		if disk.DriveTemperatureMin > 0 {
			c.metrics.DiskTemperatureMin.WithLabelValues(
				disk.Device,
				disk.Serial,
				disk.Model,
			).Set(disk.DriveTemperatureMin)
		}

		// Power and lifecycle metrics
		if disk.PowerOnHours > 0 {
			c.metrics.DiskPowerOnHours.WithLabelValues(
				disk.Device,
				disk.Serial,
				disk.Model,
			).Set(float64(disk.PowerOnHours))
		}

		if disk.PowerCycles > 0 {
			c.metrics.DiskPowerCycles.WithLabelValues(
				disk.Device,
				disk.Serial,
				disk.Model,
			).Set(float64(disk.PowerCycles))
		}

		// Capacity and usage metrics
		if disk.Capacity > 0 {
			c.metrics.DiskCapacityBytes.WithLabelValues(
				disk.Device,
				disk.Serial,
				disk.Model,
				disk.Interface,
			).Set(float64(disk.Capacity))
		}

		// Filesystem usage metrics
		if disk.UsedBytes > 0 {
			c.metrics.DiskUsedBytes.WithLabelValues(
				disk.Device,
				disk.Serial,
				disk.Model,
				disk.Interface,
				disk.Mountpoint,
				disk.Filesystem,
			).Set(float64(disk.UsedBytes))
		}

		if disk.AvailableBytes > 0 {
			c.metrics.DiskAvailableBytes.WithLabelValues(
				disk.Device,
				disk.Serial,
				disk.Model,
				disk.Interface,
				disk.Mountpoint,
				disk.Filesystem,
			).Set(float64(disk.AvailableBytes))
		}

		if disk.UsagePercentage > 0 {
			c.metrics.DiskUsagePercentage.WithLabelValues(
				disk.Device,
				disk.Serial,
				disk.Model,
				disk.Interface,
				disk.Mountpoint,
				disk.Filesystem,
			).Set(disk.UsagePercentage)
		}

		// Error metrics
		if disk.ReallocatedSectors >= 0 {
			c.metrics.DiskReallocatedSectors.WithLabelValues(
				disk.Device,
				disk.Serial,
				disk.Model,
			).Set(float64(disk.ReallocatedSectors))

			// Also update legacy sector errors metric
			c.metrics.DiskSectorErrors.WithLabelValues(
				disk.Device,
				disk.Serial,
				disk.Model,
				"reallocated_sectors",
			).Set(float64(disk.ReallocatedSectors))
		}

		if disk.PendingSectors > 0 {
			c.metrics.DiskPendingSectors.WithLabelValues(
				disk.Device,
				disk.Serial,
				disk.Model,
			).Set(float64(disk.PendingSectors))

			c.metrics.DiskSectorErrors.WithLabelValues(
				disk.Device,
				disk.Serial,
				disk.Model,
				"pending_sectors",
			).Set(float64(disk.PendingSectors))
		}

		if disk.UncorrectableErrors > 0 {
			c.metrics.DiskUncorrectableErrors.WithLabelValues(
				disk.Device,
				disk.Serial,
				disk.Model,
			).Set(float64(disk.UncorrectableErrors))

			c.metrics.DiskSectorErrors.WithLabelValues(
				disk.Device,
				disk.Serial,
				disk.Model,
				"uncorrectable_errors",
			).Set(float64(disk.UncorrectableErrors))
		}

		// I/O metrics
		if disk.TotalLBAsWritten > 0 {
			c.metrics.DiskDataUnitsWritten.WithLabelValues(
				disk.Device,
				disk.Serial,
				disk.Model,
			).Set(float64(disk.TotalLBAsWritten))
		}

		if disk.TotalLBAsRead > 0 {
			c.metrics.DiskDataUnitsRead.WithLabelValues(
				disk.Device,
				disk.Serial,
				disk.Model,
			).Set(float64(disk.TotalLBAsRead))
		}

		// SMART status metrics
		c.metrics.DiskSmartEnabled.WithLabelValues(
			disk.Device,
			disk.Serial,
			disk.Model,
		).Set(boolToFloat(disk.SmartEnabled))

		c.metrics.DiskSmartHealthy.WithLabelValues(
			disk.Device,
			disk.Serial,
			disk.Model,
		).Set(boolToFloat(disk.SmartHealthy))

		// SSD/NVMe specific metrics
		if disk.WearLeveling > 0 {
			c.metrics.DiskWearLeveling.WithLabelValues(
				disk.Device,
				disk.Serial,
				disk.Model,
			).Set(float64(disk.WearLeveling))
		}

		if disk.PercentageUsed > 0 {
			c.metrics.DiskPercentageUsed.WithLabelValues(
				disk.Device,
				disk.Serial,
				disk.Model,
			).Set(float64(disk.PercentageUsed))
		}

		if disk.AvailableSpare > 0 {
			c.metrics.DiskAvailableSpare.WithLabelValues(
				disk.Device,
				disk.Serial,
				disk.Model,
			).Set(float64(disk.AvailableSpare))
		}

		if disk.CriticalWarning > 0 {
			c.metrics.DiskCriticalWarning.WithLabelValues(
				disk.Device,
				disk.Serial,
				disk.Model,
			).Set(float64(disk.CriticalWarning))
		}

		if disk.MediaErrors > 0 {
			c.metrics.DiskMediaErrors.WithLabelValues(
				disk.Device,
				disk.Serial,
				disk.Model,
			).Set(float64(disk.MediaErrors))
		}

		if disk.ErrorLogEntries > 0 {
			c.metrics.DiskErrorLogEntries.WithLabelValues(
				disk.Device,
				disk.Serial,
				disk.Model,
			).Set(float64(disk.ErrorLogEntries))
		}

		// RAID role and spare drive metrics
		if disk.RaidRole != "" {
			roleValue := getRaidRoleValue(disk.RaidRole)
			c.metrics.DiskRaidRole.WithLabelValues(
				disk.Device,
				disk.Serial,
				disk.Model,
			).Set(float64(roleValue))
		}

		// Spare drive status metrics
		isSpare := disk.RaidRole == "hot_spare" || disk.RaidRole == "spare" ||
			disk.RaidRole == "commissioned_spare" || disk.RaidRole == "emergency_spare" ||
			disk.IsCommissionedSpare || disk.IsEmergencySpare || disk.IsGlobalSpare

		c.metrics.DiskIsSpare.WithLabelValues(
			disk.Device,
			disk.Serial,
			disk.Model,
		).Set(boolToFloat(isSpare))

		c.metrics.DiskIsCommissionedSpare.WithLabelValues(
			disk.Device,
			disk.Serial,
			disk.Model,
		).Set(boolToFloat(disk.IsCommissionedSpare))

		c.metrics.DiskIsEmergencySpare.WithLabelValues(
			disk.Device,
			disk.Serial,
			disk.Model,
		).Set(boolToFloat(disk.IsEmergencySpare))

		c.metrics.DiskIsGlobalSpare.WithLabelValues(
			disk.Device,
			disk.Serial,
			disk.Model,
		).Set(boolToFloat(disk.IsGlobalSpare))
	}
}

// getRaidRoleValue converts RAID role string to numeric value
func getRaidRoleValue(role string) int {
	switch role {
	case "active":
		return int(types.RaidRoleActive)
	case "spare", "hot_spare", "commissioned_spare", "emergency_spare":
		return int(types.RaidRoleSpare)
	case "failed":
		return int(types.RaidRoleFailed)
	case "rebuilding":
		return int(types.RaidRoleRebuilding)
	case "unconfigured":
		return int(types.RaidRoleUnconfigured)
	default:
		return int(types.RaidRoleUnknown)
	}
}
