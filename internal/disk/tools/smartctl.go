package tools

import (
	"encoding/json"
	"log"
	"os/exec"
	"strings"

	"disk-health-exporter/internal/utils"
	"disk-health-exporter/pkg/types"
)

// SmartCtlTool represents the smartctl CLI tool
type SmartCtlTool struct{}

// NewSmartCtlTool creates a new SmartCtlTool instance
func NewSmartCtlTool() *SmartCtlTool {
	return &SmartCtlTool{}
}

// IsAvailable checks if smartctl is available on the system
func (s *SmartCtlTool) IsAvailable() bool {
	return utils.CommandExists("smartctl")
}

// GetVersion returns the smartctl version
func (s *SmartCtlTool) GetVersion() string {
	if !s.IsAvailable() {
		return ""
	}

	version, err := utils.GetToolVersion("smartctl", "--version")
	if err != nil {
		return "unknown"
	}
	return version
}

// GetName returns the tool name
func (s *SmartCtlTool) GetName() string {
	return "smartctl"
}

// GetDisks returns disk information detected by smartctl
func (s *SmartCtlTool) GetDisks() []types.DiskInfo {
	var disks []types.DiskInfo

	if !s.IsAvailable() {
		return disks
	}

	log.Printf("Detecting disks using smartctl...")

	// Get list of available devices
	output, err := exec.Command("smartctl", "--scan").Output()
	if err != nil {
		log.Printf("Error scanning for devices with smartctl: %v", err)
		return disks
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 1 {
			continue
		}

		device := fields[0]

		// Check various device types (sd*, nvme*, etc.)
		if strings.Contains(device, "sd") || strings.Contains(device, "nvme") ||
			strings.Contains(device, "hd") || strings.Contains(device, "vd") {

			diskInfo := s.getSmartCtlInfo(device)
			if diskInfo.Device != "" {
				diskInfo.Type = "regular"
				disks = append(disks, diskInfo)
			}
		}
	}

	log.Printf("Found %d disks using smartctl", len(disks))
	return disks
}

// GetSmartCtlInfo gets comprehensive SMART information for a device
func (s *SmartCtlTool) GetSmartCtlInfo(device string) types.DiskInfo {
	return s.getSmartCtlInfoWithType(device, "auto")
}

// getSmartCtlInfo gets comprehensive SMART information for a device
func (s *SmartCtlTool) getSmartCtlInfo(device string) types.DiskInfo {
	return s.getSmartCtlInfoWithType(device, "auto")
}

// GetSmartCtlInfoWithType gets SMART information for a device with specific type
func (s *SmartCtlTool) GetSmartCtlInfoWithType(device, deviceType string) types.DiskInfo {
	return s.getSmartCtlInfoWithType(device, deviceType)
}

// getSmartCtlInfoWithType gets SMART information for a device with specific type
func (s *SmartCtlTool) getSmartCtlInfoWithType(device, deviceType string) types.DiskInfo {
	var diskInfo types.DiskInfo

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
		return diskInfo
	}

	var smartData types.SmartCtlOutput
	if err := json.Unmarshal(output, &smartData); err != nil {
		log.Printf("Error parsing smartctl JSON for %s: %v", device, err)
		return diskInfo
	}

	// Basic information
	diskInfo.Device = device
	diskInfo.Serial = smartData.SerialNumber
	diskInfo.Model = smartData.ModelName
	if len(strings.Fields(smartData.ModelFamily)) > 0 {
		diskInfo.Vendor = strings.Fields(smartData.ModelFamily)[0] // First word is usually vendor
	}
	diskInfo.Interface = smartData.Device.Protocol
	diskInfo.Capacity = smartData.UserCapacity.Bytes
	diskInfo.FormFactor = smartData.FormFactor.Name
	diskInfo.RPM = smartData.RotationRate

	// SMART status
	diskInfo.SmartEnabled = smartData.SmartSupport.Enabled
	diskInfo.SmartHealthy = smartData.SmartStatus.Passed
	if smartData.SmartStatus.Passed {
		diskInfo.Health = "OK"
	} else {
		diskInfo.Health = "FAILED"
	}

	// Temperature information
	diskInfo.Temperature = float64(smartData.Temperature.Current)

	// Power information
	diskInfo.PowerOnHours = int64(smartData.PowerOnTime.Hours)
	diskInfo.PowerCycles = int64(smartData.PowerCycleCount)

	// Handle different device types
	if strings.Contains(strings.ToLower(diskInfo.Interface), "nvme") {
		s.extractNVMeMetrics(&diskInfo, &smartData)
	} else {
		s.extractATAMetrics(&diskInfo, &smartData)
	}

	return diskInfo
}

// extractNVMeMetrics extracts NVMe-specific metrics
func (s *SmartCtlTool) extractNVMeMetrics(diskInfo *types.DiskInfo, smartData *types.SmartCtlOutput) {
	nvme := &smartData.NvmeSmartHealthInformationLog

	diskInfo.PercentageUsed = nvme.PercentageUsed
	diskInfo.AvailableSpare = nvme.AvailableSpare
	diskInfo.CriticalWarning = nvme.CriticalWarning
	diskInfo.MediaErrors = nvme.MediaErrors
	diskInfo.ErrorLogEntries = nvme.NumErrLogEntries
	diskInfo.TotalLBAsWritten = nvme.DataUnitsWritten
	diskInfo.TotalLBAsRead = nvme.DataUnitsRead
	diskInfo.PowerOnHours = nvme.PowerOnHours
	diskInfo.PowerCycles = nvme.PowerCycles

	// Additional temperature sensors for NVMe
	if nvme.TemperatureSensor1 > 0 {
		diskInfo.DriveTemperatureMax = float64(nvme.TemperatureSensor1)
	}
	if nvme.TemperatureSensor2 > 0 && nvme.TemperatureSensor2 < int(diskInfo.DriveTemperatureMax) {
		diskInfo.DriveTemperatureMin = float64(nvme.TemperatureSensor2)
	}
}

// extractATAMetrics extracts ATA/SATA-specific metrics
func (s *SmartCtlTool) extractATAMetrics(diskInfo *types.DiskInfo, smartData *types.SmartCtlOutput) {
	for _, attr := range smartData.AtaSmartAttributes.Table {
		switch attr.ID {
		case 5: // Reallocated Sector Count
			diskInfo.ReallocatedSectors = attr.Raw.Value
		case 196: // Reallocation Event Count
			if diskInfo.ReallocatedSectors == 0 {
				diskInfo.ReallocatedSectors = attr.Raw.Value
			}
		case 197: // Current Pending Sector Count
			diskInfo.PendingSectors = attr.Raw.Value
		case 198: // Uncorrectable Sector Count
			diskInfo.UncorrectableErrors = attr.Raw.Value
		case 231: // SSD Life Left (some drives)
			diskInfo.WearLeveling = 100 - attr.Value // Convert to wear percentage
		case 233: // Media Wearout Indicator (Intel SSDs)
			diskInfo.WearLeveling = 100 - attr.Value
		case 241: // Total LBAs Written
			diskInfo.TotalLBAsWritten = attr.Raw.Value
		case 242: // Total LBAs Read
			diskInfo.TotalLBAsRead = attr.Raw.Value
		}
	}

	// Error log entries
	diskInfo.ErrorLogEntries = int64(smartData.AtaSmartErrorLog.Summary.Count)
}
