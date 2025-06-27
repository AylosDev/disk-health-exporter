package collector

import (
	"log"
	"runtime"
	"strconv"
	"strings"
	"time"

	"disk-health-exporter/internal/config"
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

	// Update tool availability metrics
	c.updateToolMetrics()

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
func (c *Collector) updateToolMetrics() {
	toolInfo := c.diskManager.GetToolInfo()

	// Update tool availability metrics
	c.metrics.ToolAvailable.WithLabelValues("smartctl", toolInfo.SmartCtlVersion).Set(boolToFloat(toolInfo.SmartCtl))
	c.metrics.ToolAvailable.WithLabelValues("megacli", toolInfo.MegaCLIVersion).Set(boolToFloat(toolInfo.MegaCLI))
	c.metrics.ToolAvailable.WithLabelValues("mdadm", "unknown").Set(boolToFloat(toolInfo.Mdadm))
	c.metrics.ToolAvailable.WithLabelValues("arcconf", "unknown").Set(boolToFloat(toolInfo.Arcconf))
	c.metrics.ToolAvailable.WithLabelValues("storcli", "unknown").Set(boolToFloat(toolInfo.Storcli))
	c.metrics.ToolAvailable.WithLabelValues("zpool", "unknown").Set(boolToFloat(toolInfo.Zpool))
	c.metrics.ToolAvailable.WithLabelValues("diskutil", "unknown").Set(boolToFloat(toolInfo.Diskutil))
	c.metrics.ToolAvailable.WithLabelValues("nvme", "unknown").Set(boolToFloat(toolInfo.Nvme))
	c.metrics.ToolAvailable.WithLabelValues("hdparm", "unknown").Set(boolToFloat(toolInfo.Hdparm))
	c.metrics.ToolAvailable.WithLabelValues("lsblk", "unknown").Set(boolToFloat(toolInfo.Lsblk))
}

// boolToFloat converts boolean to float64 for metrics
func boolToFloat(b bool) float64 {
	if b {
		return 1.0
	}
	return 0.0
}

// collectLinuxMetrics collects metrics on Linux systems
func (c *Collector) collectLinuxMetrics() {
	disks, raidArrays := c.diskManager.GetLinuxDisks()

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
			c.updateRaidBatteryMetrics(raid.Battery)
		}
	}

	// Update comprehensive disk metrics
	c.updateComprehensiveDiskMetrics(disks)

	log.Printf("Updated metrics for %d disks and %d RAID arrays", len(disks), len(raidArrays))
}

// collectMacOSMetrics collects metrics on macOS systems
func (c *Collector) collectMacOSMetrics() {
	disks := c.diskManager.GetMacOSDisks()
	c.updateComprehensiveDiskMetrics(disks)

	log.Printf("Updated metrics for %d macOS disks", len(disks))
}

// collectFallbackMetrics collects metrics using fallback method
func (c *Collector) collectFallbackMetrics() {
	log.Printf("Using fallback disk detection for OS: %s", runtime.GOOS)

	// Try to get regular disks as fallback
	disks, _ := c.diskManager.GetLinuxDisks()
	c.updateComprehensiveDiskMetrics(disks)

	log.Printf("Updated metrics for %d disks (fallback mode)", len(disks))
}

// updateDiskMetrics updates metrics for a list of disks (legacy method)
func (c *Collector) updateDiskMetrics(disks []types.DiskInfo) {
	c.updateComprehensiveDiskMetrics(disks)
}

// updateComprehensiveDiskMetrics updates comprehensive metrics for a list of disks
func (c *Collector) updateComprehensiveDiskMetrics(disks []types.DiskInfo) {
	for _, disk := range disks {
		// Convert health status to numeric value
		status := getHealthStatusValue(disk.Health)

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
	}
}

// updateRaidBatteryMetrics updates battery metrics for a RAID controller
func (c *Collector) updateRaidBatteryMetrics(battery *types.RAIDBatteryInfo) {
	adapterIdStr := strconv.Itoa(battery.AdapterID)
	labels := []string{adapterIdStr, battery.BatteryType, "MegaCLI"}

	// Basic battery measurements
	if battery.Voltage > 0 {
		c.metrics.RaidBatteryVoltage.WithLabelValues(labels...).Set(float64(battery.Voltage))
	}

	if battery.Current >= 0 {
		c.metrics.RaidBatteryCurrent.WithLabelValues(labels...).Set(float64(battery.Current))
	}

	if battery.Temperature > 0 {
		c.metrics.RaidBatteryTemperature.WithLabelValues(labels...).Set(float64(battery.Temperature))
	}

	// Battery status as numeric value
	statusValue := getBatteryStatusValue(battery.State)
	statusLabels := []string{adapterIdStr, battery.BatteryType, battery.State, "MegaCLI"}
	c.metrics.RaidBatteryStatus.WithLabelValues(statusLabels...).Set(float64(statusValue))

	// Boolean indicators converted to 0/1
	c.metrics.RaidBatteryLearnCycleActive.WithLabelValues(labels...).Set(boolToFloat(battery.LearnCycleActive))
	c.metrics.RaidBatteryMissing.WithLabelValues(labels...).Set(boolToFloat(battery.BatteryMissing))
	c.metrics.RaidBatteryReplacementReq.WithLabelValues(labels...).Set(boolToFloat(battery.ReplacementRequired))
	c.metrics.RaidBatteryCapacityLow.WithLabelValues(labels...).Set(boolToFloat(battery.RemainingCapacityLow))

	// Energy and capacity metrics
	if battery.PackEnergy > 0 {
		c.metrics.RaidBatteryPackEnergy.WithLabelValues(labels...).Set(float64(battery.PackEnergy))
	}

	if battery.Capacitance > 0 {
		c.metrics.RaidBatteryCapacitance.WithLabelValues(labels...).Set(float64(battery.Capacitance))
	}

	if battery.BackupChargeTime >= 0 {
		c.metrics.RaidBatteryBackupChargeTime.WithLabelValues(labels...).Set(float64(battery.BackupChargeTime))
	}

	// Design specifications
	if battery.DesignCapacity > 0 {
		c.metrics.RaidBatteryDesignCapacity.WithLabelValues(labels...).Set(float64(battery.DesignCapacity))
	}

	if battery.DesignVoltage > 0 {
		c.metrics.RaidBatteryDesignVoltage.WithLabelValues(labels...).Set(float64(battery.DesignVoltage))
	}

	if battery.AutoLearnPeriod > 0 {
		c.metrics.RaidBatteryAutoLearnPeriod.WithLabelValues(labels...).Set(float64(battery.AutoLearnPeriod))
	}
}

// getBatteryStatusValue converts battery status string to numeric value
func getBatteryStatusValue(status string) int {
	switch strings.ToLower(status) {
	case "optimal":
		return 1
	case "charging":
		return 1
	case "discharging":
		return 2
	case "warning":
		return 2
	case "low":
		return 2
	case "critical":
		return 3
	case "failed":
		return 3
	case "missing":
		return 3
	default:
		return 0 // unknown
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
