package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// Metrics holds all Prometheus metrics
type Metrics struct {
	// Existing metrics
	DiskHealthStatus *prometheus.GaugeVec
	DiskTemperature  *prometheus.GaugeVec
	RaidArrayStatus  *prometheus.GaugeVec
	DiskSectorErrors *prometheus.GaugeVec
	DiskPowerOnHours *prometheus.GaugeVec
	ExporterUp       prometheus.Gauge

	// New comprehensive metrics
	DiskCapacityBytes       *prometheus.GaugeVec
	DiskPowerCycles         *prometheus.GaugeVec
	DiskReallocatedSectors  *prometheus.GaugeVec
	DiskPendingSectors      *prometheus.GaugeVec
	DiskUncorrectableErrors *prometheus.GaugeVec
	DiskDataUnitsWritten    *prometheus.GaugeVec
	DiskDataUnitsRead       *prometheus.GaugeVec
	DiskTemperatureMax      *prometheus.GaugeVec
	DiskTemperatureMin      *prometheus.GaugeVec
	DiskSmartEnabled        *prometheus.GaugeVec
	DiskSmartHealthy        *prometheus.GaugeVec

	// SSD/NVMe specific metrics
	DiskWearLeveling    *prometheus.GaugeVec
	DiskPercentageUsed  *prometheus.GaugeVec
	DiskAvailableSpare  *prometheus.GaugeVec
	DiskCriticalWarning *prometheus.GaugeVec
	DiskMediaErrors     *prometheus.GaugeVec
	DiskErrorLogEntries *prometheus.GaugeVec

	// RAID specific metrics
	RaidArraySize            *prometheus.GaugeVec
	RaidArrayUsedSize        *prometheus.GaugeVec
	RaidArrayNumDrives       *prometheus.GaugeVec
	RaidArrayNumActiveDrives *prometheus.GaugeVec
	RaidArrayNumSpareDrives  *prometheus.GaugeVec
	RaidArrayNumFailedDrives *prometheus.GaugeVec
	RaidArrayRebuildProgress *prometheus.GaugeVec
	RaidArrayScrubProgress   *prometheus.GaugeVec

	// Software RAID metrics
	SoftwareRaidArrayStatus  *prometheus.GaugeVec
	SoftwareRaidSyncProgress *prometheus.GaugeVec
	SoftwareRaidArraySize    *prometheus.GaugeVec

	// RAID Battery metrics
	RaidBatteryVoltage          *prometheus.GaugeVec
	RaidBatteryCurrent          *prometheus.GaugeVec
	RaidBatteryTemperature      *prometheus.GaugeVec
	RaidBatteryStatus           *prometheus.GaugeVec
	RaidBatteryLearnCycleActive *prometheus.GaugeVec
	RaidBatteryMissing          *prometheus.GaugeVec
	RaidBatteryReplacementReq   *prometheus.GaugeVec
	RaidBatteryCapacityLow      *prometheus.GaugeVec
	RaidBatteryPackEnergy       *prometheus.GaugeVec
	RaidBatteryCapacitance      *prometheus.GaugeVec
	RaidBatteryBackupChargeTime *prometheus.GaugeVec
	RaidBatteryDesignCapacity   *prometheus.GaugeVec
	RaidBatteryDesignVoltage    *prometheus.GaugeVec
	RaidBatteryAutoLearnPeriod  *prometheus.GaugeVec
}

// New creates and registers all metrics
func New() *Metrics {
	m := &Metrics{
		// Existing metrics
		DiskHealthStatus: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "disk_health_status",
				Help: "Disk health status (0=unknown, 1=ok, 2=warning, 3=critical)",
			},
			[]string{"device", "type", "serial", "model", "location", "interface"},
		),
		DiskTemperature: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "disk_temperature_celsius",
				Help: "Disk temperature in Celsius",
			},
			[]string{"device", "serial", "model", "interface"},
		),
		RaidArrayStatus: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "raid_array_status",
				Help: "RAID array status (0=unknown, 1=ok, 2=degraded, 3=failed)",
			},
			[]string{"array_id", "raid_level", "state", "type", "controller"},
		),
		DiskSectorErrors: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "disk_sector_errors_total",
				Help: "Total number of disk sector errors",
			},
			[]string{"device", "serial", "model", "error_type"},
		),
		DiskPowerOnHours: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "disk_power_on_hours_total",
				Help: "Total power-on hours for the disk",
			},
			[]string{"device", "serial", "model"},
		),
		ExporterUp: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "disk_health_exporter_up",
				Help: "Whether the disk health exporter is up and running",
			},
		),

		// New comprehensive metrics
		DiskCapacityBytes: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "disk_capacity_bytes",
				Help: "Disk capacity in bytes",
			},
			[]string{"device", "serial", "model", "interface"},
		),
		DiskPowerCycles: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "disk_power_cycles_total",
				Help: "Total number of power cycles",
			},
			[]string{"device", "serial", "model"},
		),
		DiskReallocatedSectors: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "disk_reallocated_sectors_total",
				Help: "Total number of reallocated sectors",
			},
			[]string{"device", "serial", "model"},
		),
		DiskPendingSectors: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "disk_pending_sectors_total",
				Help: "Total number of pending sectors",
			},
			[]string{"device", "serial", "model"},
		),
		DiskUncorrectableErrors: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "disk_uncorrectable_errors_total",
				Help: "Total number of uncorrectable errors",
			},
			[]string{"device", "serial", "model"},
		),
		DiskDataUnitsWritten: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "disk_data_units_written_total",
				Help: "Total data units written",
			},
			[]string{"device", "serial", "model"},
		),
		DiskDataUnitsRead: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "disk_data_units_read_total",
				Help: "Total data units read",
			},
			[]string{"device", "serial", "model"},
		),
		DiskTemperatureMax: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "disk_temperature_max_celsius",
				Help: "Maximum recorded disk temperature in Celsius",
			},
			[]string{"device", "serial", "model"},
		),
		DiskTemperatureMin: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "disk_temperature_min_celsius",
				Help: "Minimum recorded disk temperature in Celsius",
			},
			[]string{"device", "serial", "model"},
		),
		DiskSmartEnabled: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "disk_smart_enabled",
				Help: "Whether SMART is enabled (1=enabled, 0=disabled)",
			},
			[]string{"device", "serial", "model"},
		),
		DiskSmartHealthy: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "disk_smart_healthy",
				Help: "SMART overall health assessment (1=healthy, 0=unhealthy)",
			},
			[]string{"device", "serial", "model"},
		),

		// SSD/NVMe specific metrics
		DiskWearLeveling: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "disk_wear_leveling_percentage",
				Help: "SSD wear leveling percentage (0-100)",
			},
			[]string{"device", "serial", "model"},
		),
		DiskPercentageUsed: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "disk_percentage_used",
				Help: "NVMe percentage used (0-100)",
			},
			[]string{"device", "serial", "model"},
		),
		DiskAvailableSpare: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "disk_available_spare_percentage",
				Help: "NVMe available spare percentage",
			},
			[]string{"device", "serial", "model"},
		),
		DiskCriticalWarning: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "disk_critical_warning",
				Help: "NVMe critical warning flags",
			},
			[]string{"device", "serial", "model"},
		),
		DiskMediaErrors: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "disk_media_errors_total",
				Help: "Total number of media errors",
			},
			[]string{"device", "serial", "model"},
		),
		DiskErrorLogEntries: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "disk_error_log_entries_total",
				Help: "Total number of error log entries",
			},
			[]string{"device", "serial", "model"},
		),

		// RAID specific metrics
		RaidArraySize: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "raid_array_size_bytes",
				Help: "RAID array size in bytes",
			},
			[]string{"array_id", "raid_level", "type"},
		),
		RaidArrayUsedSize: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "raid_array_used_size_bytes",
				Help: "RAID array used size in bytes",
			},
			[]string{"array_id", "raid_level", "type"},
		),
		RaidArrayNumDrives: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "raid_array_drives_total",
				Help: "Total number of drives in RAID array",
			},
			[]string{"array_id", "raid_level", "type"},
		),
		RaidArrayNumActiveDrives: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "raid_array_active_drives",
				Help: "Number of active drives in RAID array",
			},
			[]string{"array_id", "raid_level", "type"},
		),
		RaidArrayNumSpareDrives: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "raid_array_spare_drives",
				Help: "Number of spare drives in RAID array",
			},
			[]string{"array_id", "raid_level", "type"},
		),
		RaidArrayNumFailedDrives: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "raid_array_failed_drives",
				Help: "Number of failed drives in RAID array",
			},
			[]string{"array_id", "raid_level", "type"},
		),
		RaidArrayRebuildProgress: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "raid_array_rebuild_progress_percentage",
				Help: "RAID array rebuild progress percentage (0-100)",
			},
			[]string{"array_id", "raid_level", "type"},
		),
		RaidArrayScrubProgress: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "raid_array_scrub_progress_percentage",
				Help: "RAID array scrub progress percentage (0-100)",
			},
			[]string{"array_id", "raid_level", "type"},
		),

		// Software RAID metrics
		SoftwareRaidArrayStatus: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "software_raid_array_status",
				Help: "Software RAID array status (0=unknown, 1=clean, 2=degraded, 3=failed)",
			},
			[]string{"device", "level", "state"},
		),
		SoftwareRaidSyncProgress: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "software_raid_sync_progress_percentage",
				Help: "Software RAID sync progress percentage (0-100)",
			},
			[]string{"device", "level", "sync_action"},
		),
		SoftwareRaidArraySize: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "software_raid_array_size_bytes",
				Help: "Software RAID array size in bytes",
			},
			[]string{"device", "level"},
		),

		// RAID Battery metrics
		RaidBatteryVoltage: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "raid_battery_voltage_millivolts",
				Help: "RAID controller battery voltage in millivolts",
			},
			[]string{"adapter_id", "battery_type", "controller"},
		),
		RaidBatteryCurrent: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "raid_battery_current_milliamps",
				Help: "RAID controller battery current in milliamps",
			},
			[]string{"adapter_id", "battery_type", "controller"},
		),
		RaidBatteryTemperature: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "raid_battery_temperature_celsius",
				Help: "RAID controller battery temperature in Celsius",
			},
			[]string{"adapter_id", "battery_type", "controller"},
		),
		RaidBatteryStatus: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "raid_battery_status",
				Help: "RAID controller battery status (0=unknown, 1=optimal, 2=warning, 3=critical)",
			},
			[]string{"adapter_id", "battery_type", "state", "controller"},
		),
		RaidBatteryLearnCycleActive: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "raid_battery_learn_cycle_active",
				Help: "RAID controller battery learn cycle active (0=no, 1=yes)",
			},
			[]string{"adapter_id", "battery_type", "controller"},
		),
		RaidBatteryMissing: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "raid_battery_missing",
				Help: "RAID controller battery missing (0=no, 1=yes)",
			},
			[]string{"adapter_id", "battery_type", "controller"},
		),
		RaidBatteryReplacementReq: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "raid_battery_replacement_required",
				Help: "RAID controller battery replacement required (0=no, 1=yes)",
			},
			[]string{"adapter_id", "battery_type", "controller"},
		),
		RaidBatteryCapacityLow: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "raid_battery_capacity_low",
				Help: "RAID controller battery remaining capacity low (0=no, 1=yes)",
			},
			[]string{"adapter_id", "battery_type", "controller"},
		),
		RaidBatteryPackEnergy: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "raid_battery_pack_energy_joules",
				Help: "RAID controller battery pack energy in joules",
			},
			[]string{"adapter_id", "battery_type", "controller"},
		),
		RaidBatteryCapacitance: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "raid_battery_capacitance",
				Help: "RAID controller battery capacitance",
			},
			[]string{"adapter_id", "battery_type", "controller"},
		),
		RaidBatteryBackupChargeTime: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "raid_battery_backup_charge_time_hours",
				Help: "RAID controller battery backup charge time in hours",
			},
			[]string{"adapter_id", "battery_type", "controller"},
		),
		RaidBatteryDesignCapacity: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "raid_battery_design_capacity_joules",
				Help: "RAID controller battery design capacity in joules",
			},
			[]string{"adapter_id", "battery_type", "controller"},
		),
		RaidBatteryDesignVoltage: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "raid_battery_design_voltage_millivolts",
				Help: "RAID controller battery design voltage in millivolts",
			},
			[]string{"adapter_id", "battery_type", "controller"},
		),
		RaidBatteryAutoLearnPeriod: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "raid_battery_auto_learn_period_days",
				Help: "RAID controller battery auto learn period in days",
			},
			[]string{"adapter_id", "battery_type", "controller"},
		),
	}

	// Register all metrics
	prometheus.MustRegister(
		// Existing metrics
		m.DiskHealthStatus,
		m.DiskTemperature,
		m.RaidArrayStatus,
		m.DiskSectorErrors,
		m.DiskPowerOnHours,
		m.ExporterUp,

		// New comprehensive metrics
		m.DiskCapacityBytes,
		m.DiskPowerCycles,
		m.DiskReallocatedSectors,
		m.DiskPendingSectors,
		m.DiskUncorrectableErrors,
		m.DiskDataUnitsWritten,
		m.DiskDataUnitsRead,
		m.DiskTemperatureMax,
		m.DiskTemperatureMin,
		m.DiskSmartEnabled,
		m.DiskSmartHealthy,

		// SSD/NVMe specific metrics
		m.DiskWearLeveling,
		m.DiskPercentageUsed,
		m.DiskAvailableSpare,
		m.DiskCriticalWarning,
		m.DiskMediaErrors,
		m.DiskErrorLogEntries,

		// RAID specific metrics
		m.RaidArraySize,
		m.RaidArrayUsedSize,
		m.RaidArrayNumDrives,
		m.RaidArrayNumActiveDrives,
		m.RaidArrayNumSpareDrives,
		m.RaidArrayNumFailedDrives,
		m.RaidArrayRebuildProgress,
		m.RaidArrayScrubProgress,

		// Software RAID metrics
		m.SoftwareRaidArrayStatus,
		m.SoftwareRaidSyncProgress,
		m.SoftwareRaidArraySize,

		// RAID Battery metrics
		m.RaidBatteryVoltage,
		m.RaidBatteryCurrent,
		m.RaidBatteryTemperature,
		m.RaidBatteryStatus,
		m.RaidBatteryLearnCycleActive,
		m.RaidBatteryMissing,
		m.RaidBatteryReplacementReq,
		m.RaidBatteryCapacityLow,
		m.RaidBatteryPackEnergy,
		m.RaidBatteryCapacitance,
		m.RaidBatteryBackupChargeTime,
		m.RaidBatteryDesignCapacity,
		m.RaidBatteryDesignVoltage,
		m.RaidBatteryAutoLearnPeriod,
	)

	return m
}

// Reset clears all metrics
func (m *Metrics) Reset() {
	// Existing metrics
	m.DiskHealthStatus.Reset()
	m.DiskTemperature.Reset()
	m.RaidArrayStatus.Reset()
	m.DiskSectorErrors.Reset()
	m.DiskPowerOnHours.Reset()

	// New comprehensive metrics
	m.DiskCapacityBytes.Reset()
	m.DiskPowerCycles.Reset()
	m.DiskReallocatedSectors.Reset()
	m.DiskPendingSectors.Reset()
	m.DiskUncorrectableErrors.Reset()
	m.DiskDataUnitsWritten.Reset()
	m.DiskDataUnitsRead.Reset()
	m.DiskTemperatureMax.Reset()
	m.DiskTemperatureMin.Reset()
	m.DiskSmartEnabled.Reset()
	m.DiskSmartHealthy.Reset()

	// SSD/NVMe specific metrics
	m.DiskWearLeveling.Reset()
	m.DiskPercentageUsed.Reset()
	m.DiskAvailableSpare.Reset()
	m.DiskCriticalWarning.Reset()
	m.DiskMediaErrors.Reset()
	m.DiskErrorLogEntries.Reset()

	// RAID specific metrics
	m.RaidArraySize.Reset()
	m.RaidArrayUsedSize.Reset()
	m.RaidArrayNumDrives.Reset()
	m.RaidArrayNumActiveDrives.Reset()
	m.RaidArrayNumSpareDrives.Reset()
	m.RaidArrayNumFailedDrives.Reset()
	m.RaidArrayRebuildProgress.Reset()
	m.RaidArrayScrubProgress.Reset()

	// Software RAID metrics
	m.SoftwareRaidArrayStatus.Reset()
	m.SoftwareRaidSyncProgress.Reset()
	m.SoftwareRaidArraySize.Reset()

	// RAID Battery metrics
	m.RaidBatteryVoltage.Reset()
	m.RaidBatteryCurrent.Reset()
	m.RaidBatteryTemperature.Reset()
	m.RaidBatteryStatus.Reset()
	m.RaidBatteryLearnCycleActive.Reset()
	m.RaidBatteryMissing.Reset()
	m.RaidBatteryReplacementReq.Reset()
	m.RaidBatteryCapacityLow.Reset()
	m.RaidBatteryPackEnergy.Reset()
	m.RaidBatteryCapacitance.Reset()
	m.RaidBatteryBackupChargeTime.Reset()
	m.RaidBatteryDesignCapacity.Reset()
	m.RaidBatteryDesignVoltage.Reset()
	m.RaidBatteryAutoLearnPeriod.Reset()
}
