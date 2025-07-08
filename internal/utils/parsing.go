package utils

import (
	"strconv"
	"strings"

	"disk-health-exporter/pkg/types"
)

// GetHealthStatusValue converts health string to numeric value
func GetHealthStatusValue(health string) int {
	health = strings.ToUpper(strings.TrimSpace(health))

	switch {
	case strings.Contains(health, "OK") || strings.Contains(health, "ONLINE") || strings.Contains(health, "OPTIMAL") ||
		strings.Contains(health, "SPUN UP") || strings.Contains(health, "HOTSPARE") || strings.Contains(health, "SPARE") ||
		strings.Contains(health, "UNCONFIGURED(GOOD)"):
		return int(types.HealthStatusOK)
	case strings.Contains(health, "WARNING") || strings.Contains(health, "REBUILDING") || strings.Contains(health, "SPUN DOWN"):
		return int(types.HealthStatusWarning)
	case strings.Contains(health, "CRITICAL") || strings.Contains(health, "FAILED") || strings.Contains(health, "OFFLINE") ||
		strings.Contains(health, "UNCONFIGURED(BAD)"):
		return int(types.HealthStatusCritical)
	default:
		return int(types.HealthStatusUnknown)
	}
}

// GetRaidStatusValue converts RAID state string to numeric value
func GetRaidStatusValue(state string) int {
	state = strings.ToUpper(strings.TrimSpace(state))

	switch {
	case strings.Contains(state, "OPTIMAL") || strings.Contains(state, "OPTL") || strings.Contains(state, "OK"):
		return 1
	case strings.Contains(state, "DEGRADED") || strings.Contains(state, "REBUILDING") || strings.Contains(state, "DGRD"):
		return 2
	case strings.Contains(state, "FAILED") || strings.Contains(state, "OFFLINE") || strings.Contains(state, "FAIL"):
		return 3
	default:
		return 0
	}
}

// ParseSizeToBytes converts human-readable size strings to bytes
func ParseSizeToBytes(sizeStr string) int64 {
	if sizeStr == "" {
		return 0
	}

	// Remove spaces and convert to uppercase
	sizeStr = strings.ToUpper(strings.ReplaceAll(sizeStr, " ", ""))

	// Extract numeric part and unit
	var numStr strings.Builder
	var unit string

	for i, r := range sizeStr {
		if r >= '0' && r <= '9' || r == '.' {
			numStr.WriteRune(r)
		} else {
			unit = sizeStr[i:]
			break
		}
	}

	// Parse the numeric value
	value, err := strconv.ParseFloat(numStr.String(), 64)
	if err != nil {
		return 0
	}

	// Convert based on unit
	switch unit {
	case "B", "":
		return int64(value)
	case "KB", "K":
		return int64(value * 1024)
	case "MB", "M":
		return int64(value * 1024 * 1024)
	case "GB", "G":
		return int64(value * 1024 * 1024 * 1024)
	case "TB", "T":
		return int64(value * 1024 * 1024 * 1024 * 1024)
	case "PB", "P":
		return int64(value * 1024 * 1024 * 1024 * 1024 * 1024)
	default:
		return int64(value)
	}
}

// GetSoftwareRAIDStatusValue converts software RAID state to numeric value
func GetSoftwareRAIDStatusValue(state string) int {
	state = strings.ToLower(strings.TrimSpace(state))

	switch state {
	case "clean", "active":
		return 1
	case "degraded", "recovering", "resyncing":
		return 2
	case "failed", "inactive":
		return 3
	default:
		return 0
	}
}

// GetBatteryStatusValue converts battery status string to numeric value
func GetBatteryStatusValue(status string) int {
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
