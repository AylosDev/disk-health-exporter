package disk

import (
	"encoding/json"
	"log"
	"os"
	"os/exec"
	"strings"

	"disk-health-exporter/pkg/types"
)

// Manager handles disk detection and monitoring
type Manager struct{}

// New creates a new disk manager
func New() *Manager {
	return &Manager{}
}

// commandExists checks if a command is available in PATH
func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

// getHealthStatusValue converts health string to numeric value
func getHealthStatusValue(health string) int {
	health = strings.ToUpper(strings.TrimSpace(health))

	switch {
	case strings.Contains(health, "OK") || strings.Contains(health, "ONLINE") || strings.Contains(health, "OPTIMAL"):
		return int(types.HealthStatusOK)
	case strings.Contains(health, "WARNING") || strings.Contains(health, "REBUILDING"):
		return int(types.HealthStatusWarning)
	case strings.Contains(health, "CRITICAL") || strings.Contains(health, "FAILED") || strings.Contains(health, "OFFLINE"):
		return int(types.HealthStatusCritical)
	default:
		return int(types.HealthStatusUnknown)
	}
}

// getRaidStatusValue converts RAID state string to numeric value
func getRaidStatusValue(state string) int {
	state = strings.ToUpper(strings.TrimSpace(state))

	switch {
	case strings.Contains(state, "OPTIMAL"):
		return 1
	case strings.Contains(state, "DEGRADED") || strings.Contains(state, "REBUILDING"):
		return 2
	case strings.Contains(state, "FAILED") || strings.Contains(state, "OFFLINE"):
		return 3
	default:
		return 0
	}
}

// getSmartCtlInfo gets SMART information for a device
func getSmartCtlInfo(device string) types.DiskInfo {
	var disk types.DiskInfo

	// Get JSON output from smartctl
	output, err := exec.Command("smartctl", "-a", "-j", device).Output()
	if err != nil {
		log.Printf("Error getting smartctl info for %s: %v", device, err)
		return disk
	}

	var smartData types.SmartCtlOutput
	if err := json.Unmarshal(output, &smartData); err != nil {
		log.Printf("Error parsing smartctl JSON for %s: %v", device, err)
		return disk
	}

	disk.Device = device
	disk.Serial = smartData.SerialNumber
	disk.Model = smartData.ModelName
	disk.Temperature = float64(smartData.Temperature.Current)

	if smartData.SmartStatus.Passed {
		disk.Health = "OK"
	} else {
		disk.Health = "FAILED"
	}

	return disk
}

// isPartOfSoftwareRAID checks if a device is part of a software RAID array
func isPartOfSoftwareRAID(device string) bool {
	// Check if device is part of mdadm RAID
	if commandExists("mdadm") {
		output, err := exec.Command("mdadm", "--examine", device).Output()
		if err == nil && strings.Contains(string(output), "Magic") {
			return true
		}
	}

	// Check /proc/mdstat for software RAID
	if data, err := os.ReadFile("/proc/mdstat"); err == nil {
		content := string(data)
		deviceName := strings.TrimPrefix(device, "/dev/")
		if strings.Contains(content, deviceName) {
			return true
		}
	}

	return false
}
