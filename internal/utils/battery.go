package utils

import (
	"strconv"

	"disk-health-exporter/internal/metrics"
	"disk-health-exporter/pkg/types"
)

// UpdateBatteryMetrics updates all battery metrics for a RAID controller
func UpdateBatteryMetrics(battery *types.RAIDBatteryInfo, m *metrics.Metrics) {
	if battery == nil {
		return
	}

	adapterIDStr := strconv.Itoa(battery.AdapterID)
	toolName := battery.ToolName
	if toolName == "" {
		toolName = "Unknown" // Fallback for missing tool name
	}
	labels := []string{adapterIDStr, battery.BatteryType, toolName}

	// Basic battery measurements
	if battery.Voltage > 0 {
		m.RaidBatteryVoltage.WithLabelValues(labels...).Set(float64(battery.Voltage))
	}

	if battery.Current >= 0 {
		m.RaidBatteryCurrent.WithLabelValues(labels...).Set(float64(battery.Current))
	}

	if battery.Temperature > 0 {
		m.RaidBatteryTemperature.WithLabelValues(labels...).Set(float64(battery.Temperature))
	}

	// Battery status as numeric value
	statusValue := GetBatteryStatusValue(battery.State)
	statusLabels := []string{adapterIDStr, battery.BatteryType, battery.State, toolName}
	m.RaidBatteryStatus.WithLabelValues(statusLabels...).Set(float64(statusValue))

	// Boolean indicators converted to 0/1
	m.RaidBatteryLearnCycleActive.WithLabelValues(labels...).Set(boolToFloat(battery.LearnCycleActive))
	m.RaidBatteryMissing.WithLabelValues(labels...).Set(boolToFloat(battery.BatteryMissing))
	m.RaidBatteryReplacementReq.WithLabelValues(labels...).Set(boolToFloat(battery.ReplacementRequired))
	m.RaidBatteryCapacityLow.WithLabelValues(labels...).Set(boolToFloat(battery.RemainingCapacityLow))

	// Energy and capacity metrics
	if battery.PackEnergy > 0 {
		m.RaidBatteryPackEnergy.WithLabelValues(labels...).Set(float64(battery.PackEnergy))
	}

	if battery.Capacitance > 0 {
		m.RaidBatteryCapacitance.WithLabelValues(labels...).Set(float64(battery.Capacitance))
	}

	if battery.BackupChargeTime >= 0 {
		m.RaidBatteryBackupChargeTime.WithLabelValues(labels...).Set(float64(battery.BackupChargeTime))
	}

	// Design specifications
	if battery.DesignCapacity > 0 {
		m.RaidBatteryDesignCapacity.WithLabelValues(labels...).Set(float64(battery.DesignCapacity))
	}

	if battery.DesignVoltage > 0 {
		m.RaidBatteryDesignVoltage.WithLabelValues(labels...).Set(float64(battery.DesignVoltage))
	}

	if battery.AutoLearnPeriod > 0 {
		m.RaidBatteryAutoLearnPeriod.WithLabelValues(labels...).Set(float64(battery.AutoLearnPeriod))
	}
}

// boolToFloat converts boolean to float64 for metrics
func boolToFloat(b bool) float64 {
	if b {
		return 1.0
	}
	return 0.0
}
