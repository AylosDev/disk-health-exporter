package disk

import (
	"encoding/json"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"disk-health-exporter/pkg/types"
)

// Manager handles disk detection and monitoring
type Manager struct {
	tools types.ToolInfo
}

// New creates a new disk manager
func New() *Manager {
	m := &Manager{}
	m.detectTools()
	return m
}

// detectTools detects available system tools
func (m *Manager) detectTools() {
	m.tools.SmartCtl = commandExists("smartctl")
	m.tools.MegaCLI = commandExists("megacli") || commandExists("MegaCli64")
	m.tools.Mdadm = commandExists("mdadm")
	m.tools.Arcconf = commandExists("arcconf")
	m.tools.Storcli = commandExists("storcli") || commandExists("storcli64")
	m.tools.Zpool = commandExists("zpool")
	m.tools.Diskutil = commandExists("diskutil")
	m.tools.Nvme = commandExists("nvme")
	m.tools.Hdparm = commandExists("hdparm")
	m.tools.Lsblk = commandExists("lsblk")

	// Get tool versions
	if m.tools.SmartCtl {
		if version, err := getToolVersion("smartctl", "--version"); err == nil {
			m.tools.SmartCtlVersion = version
		}
	}
	if m.tools.MegaCLI {
		cmd := "megacli"
		if commandExists("MegaCli64") {
			cmd = "MegaCli64"
		}
		if version, err := getToolVersion(cmd, "-v"); err == nil {
			m.tools.MegaCLIVersion = version
		}
	}

	log.Printf("Tool detection complete: smartctl=%v, megacli=%v, mdadm=%v, arcconf=%v, storcli=%v",
		m.tools.SmartCtl, m.tools.MegaCLI, m.tools.Mdadm, m.tools.Arcconf, m.tools.Storcli)
}

// GetToolInfo returns information about available tools
func (m *Manager) GetToolInfo() types.ToolInfo {
	return m.tools
}

// commandExists checks if a command is available in PATH
func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

// getToolVersion extracts version information from a tool
func getToolVersion(tool, versionFlag string) (string, error) {
	output, err := exec.Command(tool, versionFlag).Output()
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "version") || strings.Contains(line, "Version") {
			return strings.TrimSpace(line), nil
		}
	}

	// If no version line found, return first non-empty line
	if len(lines) > 0 && strings.TrimSpace(lines[0]) != "" {
		return strings.TrimSpace(lines[0]), nil
	}

	return "unknown", nil
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

// getSmartCtlInfo gets comprehensive SMART information for a device
func getSmartCtlInfo(device string) types.DiskInfo {
	return getSmartCtlInfoWithType(device, "auto")
}

// getSmartCtlInfoWithType gets SMART information for a device with specific type
func getSmartCtlInfoWithType(device, deviceType string) types.DiskInfo {
	var disk types.DiskInfo

	// Build smartctl command
	var cmd *exec.Cmd
	if deviceType == "auto" {
		cmd = exec.Command("smartctl", "-a", "-j", device)
	} else {
		cmd = exec.Command("smartctl", "-d", deviceType, "-a", "-j", device)
	}

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error getting smartctl info for %s (%s): %v", device, deviceType, err)
		return disk
	}

	var smartData types.SmartCtlOutput
	if err := json.Unmarshal(output, &smartData); err != nil {
		log.Printf("Error parsing smartctl JSON for %s: %v", device, err)
		return disk
	}

	// Basic information
	disk.Device = device
	disk.Serial = smartData.SerialNumber
	disk.Model = smartData.ModelName
	disk.Vendor = strings.Fields(smartData.ModelFamily)[0] // First word is usually vendor
	disk.Interface = smartData.Device.Protocol
	disk.Capacity = smartData.UserCapacity.Bytes
	disk.FormFactor = smartData.FormFactor.Name
	disk.RPM = smartData.RotationRate

	// SMART status
	disk.SmartEnabled = smartData.SmartSupport.Enabled
	disk.SmartHealthy = smartData.SmartStatus.Passed
	if smartData.SmartStatus.Passed {
		disk.Health = "OK"
	} else {
		disk.Health = "FAILED"
	}

	// Temperature information
	disk.Temperature = float64(smartData.Temperature.Current)
	disk.DriveTemperatureMax = float64(smartData.Temperature.Lifetime) // This would need parsing
	disk.DriveTemperatureMin = float64(smartData.Temperature.Lifetime) // This would need parsing

	// Power information
	disk.PowerOnHours = int64(smartData.PowerOnTime.Hours)
	disk.PowerCycles = int64(smartData.PowerCycleCount)

	// Handle different device types
	if strings.Contains(strings.ToLower(disk.Interface), "nvme") {
		extractNVMeMetrics(&disk, &smartData)
	} else {
		extractATAMetrics(&disk, &smartData)
	}

	return disk
}

// extractNVMeMetrics extracts NVMe-specific metrics
func extractNVMeMetrics(disk *types.DiskInfo, smartData *types.SmartCtlOutput) {
	nvme := &smartData.NvmeSmartHealthInformationLog

	disk.PercentageUsed = nvme.PercentageUsed
	disk.AvailableSpare = nvme.AvailableSpare
	disk.CriticalWarning = nvme.CriticalWarning
	disk.MediaErrors = nvme.MediaErrors
	disk.ErrorLogEntries = nvme.NumErrLogEntries
	disk.TotalLBAsWritten = nvme.DataUnitsWritten
	disk.TotalLBAsRead = nvme.DataUnitsRead
	disk.PowerOnHours = nvme.PowerOnHours
	disk.PowerCycles = nvme.PowerCycles

	// Additional temperature sensors for NVMe
	if nvme.TemperatureSensor1 > 0 {
		disk.DriveTemperatureMax = float64(nvme.TemperatureSensor1)
	}
	if nvme.TemperatureSensor2 > 0 && nvme.TemperatureSensor2 < int(disk.DriveTemperatureMax) {
		disk.DriveTemperatureMin = float64(nvme.TemperatureSensor2)
	}
}

// extractATAMetrics extracts ATA/SATA-specific metrics
func extractATAMetrics(disk *types.DiskInfo, smartData *types.SmartCtlOutput) {
	for _, attr := range smartData.AtaSmartAttributes.Table {
		switch attr.ID {
		case 5: // Reallocated Sector Count
			disk.ReallocatedSectors = attr.Raw.Value
		case 196: // Reallocation Event Count
			if disk.ReallocatedSectors == 0 {
				disk.ReallocatedSectors = attr.Raw.Value
			}
		case 197: // Current Pending Sector Count
			disk.PendingSectors = attr.Raw.Value
		case 198: // Uncorrectable Sector Count
			disk.UncorrectableErrors = attr.Raw.Value
		case 231: // SSD Life Left (some drives)
			disk.WearLeveling = 100 - attr.Value // Convert to wear percentage
		case 233: // Media Wearout Indicator (Intel SSDs)
			disk.WearLeveling = 100 - attr.Value
		case 241: // Total LBAs Written
			disk.TotalLBAsWritten = attr.Raw.Value
		case 242: // Total LBAs Read
			disk.TotalLBAsRead = attr.Raw.Value
		}
	}

	// Error log entries
	disk.ErrorLogEntries = int64(smartData.AtaSmartErrorLog.Summary.Count)
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

// getSoftwareRAIDInfo gets software RAID information using mdadm
func (m *Manager) getSoftwareRAIDInfo() []types.SoftwareRAIDInfo {
	var softwareRAIDs []types.SoftwareRAIDInfo

	if !m.tools.Mdadm {
		return softwareRAIDs
	}

	// Get list of RAID devices from /proc/mdstat
	mdstat, err := os.ReadFile("/proc/mdstat")
	if err != nil {
		log.Printf("Error reading /proc/mdstat: %v", err)
		return softwareRAIDs
	}

	lines := strings.Split(string(mdstat), "\n")
	var currentRAID types.SoftwareRAIDInfo
	var inRAIDBlock bool

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Check for RAID device line (e.g., "md0 : active raid1 sdb1[1] sda1[0]")
		if strings.HasPrefix(line, "md") && strings.Contains(line, " : ") {
			// Save previous RAID if exists
			if inRAIDBlock && currentRAID.Device != "" {
				softwareRAIDs = append(softwareRAIDs, currentRAID)
			}

			// Parse new RAID device
			currentRAID = types.SoftwareRAIDInfo{}
			inRAIDBlock = true

			parts := strings.Fields(line)
			if len(parts) >= 4 {
				currentRAID.Device = "/dev/" + parts[0]
				currentRAID.State = parts[2] // active, inactive, etc.
				currentRAID.Level = parts[3] // raid0, raid1, etc.

				// Extract device list
				for i := 4; i < len(parts); i++ {
					device := parts[i]
					// Remove bracketed information [0], [1], etc.
					deviceName := regexp.MustCompile(`\[[^\]]*\]`).ReplaceAllString(device, "")
					deviceName = "/dev/" + deviceName

					if strings.Contains(device, "(F)") {
						currentRAID.FailedDevices = append(currentRAID.FailedDevices, deviceName)
					} else if strings.Contains(device, "(S)") {
						currentRAID.SpareDevices = append(currentRAID.SpareDevices, deviceName)
					} else {
						currentRAID.ActiveDevices = append(currentRAID.ActiveDevices, deviceName)
					}
				}
			}
		} else if inRAIDBlock && strings.Contains(line, " blocks ") {
			// Parse size line (e.g., "1000000 blocks super 1.2 [2/2] [UU]")
			parts := strings.Fields(line)
			if len(parts) >= 1 {
				if size, err := strconv.ParseInt(parts[0], 10, 64); err == nil {
					currentRAID.ArraySize = size * 1024 // Convert from KB to bytes
				}
			}

			// Parse RAID configuration [2/2] [UU]
			if len(parts) >= 4 {
				raidConfigRe := regexp.MustCompile(`\[(\d+)/(\d+)\]`)
				matches := raidConfigRe.FindStringSubmatch(line)
				if len(matches) == 3 {
					if total, err := strconv.Atoi(matches[1]); err == nil {
						currentRAID.TotalDevices = total
					}
					if active, err := strconv.Atoi(matches[2]); err == nil {
						currentRAID.RaidDevices = active
					}
				}
			}
		} else if inRAIDBlock && (strings.Contains(line, "resync") || strings.Contains(line, "recover") || strings.Contains(line, "check")) {
			// Parse sync progress line
			currentRAID.SyncAction = extractSyncAction(line)
			currentRAID.SyncProgress = extractSyncProgress(line)
		} else if line == "" {
			// End of RAID block
			if inRAIDBlock && currentRAID.Device != "" {
				softwareRAIDs = append(softwareRAIDs, currentRAID)
				inRAIDBlock = false
			}
		}
	}

	// Add the last RAID if exists
	if inRAIDBlock && currentRAID.Device != "" {
		softwareRAIDs = append(softwareRAIDs, currentRAID)
	}

	// Get additional details for each RAID using mdadm
	for i := range softwareRAIDs {
		m.enrichSoftwareRAIDInfo(&softwareRAIDs[i])
	}

	return softwareRAIDs
}

// enrichSoftwareRAIDInfo adds detailed information using mdadm --detail
func (m *Manager) enrichSoftwareRAIDInfo(raid *types.SoftwareRAIDInfo) {
	output, err := exec.Command("mdadm", "--detail", raid.Device).Output()
	if err != nil {
		log.Printf("Error getting mdadm details for %s: %v", raid.Device, err)
		return
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.Contains(line, "UUID :") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				raid.UUID = strings.TrimSpace(parts[1])
			}
		} else if strings.Contains(line, "Update Time :") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				raid.UpdateTime = strings.TrimSpace(strings.Join(parts[1:], ":"))
			}
		} else if strings.Contains(line, "Persistence :") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				raid.Persistence = strings.TrimSpace(parts[1])
			}
		} else if strings.Contains(line, "Bitmap :") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				raid.Bitmap = strings.TrimSpace(parts[1])
			}
		}
	}
}

// extractSyncAction extracts the sync action from mdstat line
func extractSyncAction(line string) string {
	if strings.Contains(line, "resync") {
		return "resync"
	} else if strings.Contains(line, "recover") {
		return "recover"
	} else if strings.Contains(line, "check") {
		return "check"
	}
	return ""
}

// extractSyncProgress extracts sync progress percentage from mdstat line
func extractSyncProgress(line string) float64 {
	// Look for pattern like "resync = 45.2%"
	re := regexp.MustCompile(`(\d+\.?\d*)%`)
	matches := re.FindStringSubmatch(line)
	if len(matches) > 1 {
		if progress, err := strconv.ParseFloat(matches[1], 64); err == nil {
			return progress
		}
	}
	return 0.0
}

// getSoftwareRAIDStatusValue converts software RAID state to numeric value
func getSoftwareRAIDStatusValue(state string) int {
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
