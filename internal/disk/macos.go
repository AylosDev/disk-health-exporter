package disk

import (
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"strings"

	"disk-health-exporter/pkg/types"
)

// GetMacOSDisks gets disk information on macOS systems
func (m *Manager) GetMacOSDisks() []types.DiskInfo {
	var disks []types.DiskInfo

	log.Println("Detecting macOS disks...")

	if !commandExists("smartctl") {
		log.Println("smartctl not found, cannot detect macOS disks")
		return disks
	}

	// Primary method: Use smartctl --scan to find devices
	disks = m.scanForMacOSDevices()

	// Fallback method: Try common macOS device paths if scan found nothing
	if len(disks) == 0 {
		log.Println("No devices found via scan, trying direct detection...")
		disks = m.tryDirectMacOSDetection()
	}

	// Last resort: Create basic entries from diskutil if still no devices found
	if len(disks) == 0 {
		log.Println("No SMART data available, creating basic disk entries...")
		disks = m.createBasicMacOSEntries()
	}

	log.Printf("Found %d disks on macOS before filtering", len(disks))

	// Apply filtering based on configuration
	filteredDisks := m.filterDisks(disks)
	log.Printf("Found %d disks on macOS after filtering", len(filteredDisks))
	return filteredDisks
}

// scanForMacOSDevices scans for devices using smartctl --scan
func (m *Manager) scanForMacOSDevices() []types.DiskInfo {
	var disks []types.DiskInfo

	output, err := exec.Command("smartctl", "--scan").Output()
	if err != nil {
		log.Printf("Error scanning for devices: %v", err)
		return disks
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse smartctl scan format: "device -d devicetype # comment"
		device, deviceType := m.parseSmartCtlScanLine(line)
		if device == "" || deviceType == "" {
			continue
		}

		log.Printf("Found device from scan: %s (type: %s)", device, deviceType)

		// Get SMART data for this device
		diskInfo := m.getMacOSSmartCtlInfo(device, deviceType)
		if diskInfo.Device != "" {
			diskInfo.Type = fmt.Sprintf("macos-%s", deviceType)
			disks = append(disks, diskInfo)
		}
	}

	return disks
}

// parseSmartCtlScanLine parses a line from smartctl --scan output
func (m *Manager) parseSmartCtlScanLine(line string) (device, deviceType string) {
	// Format: "device -d devicetype # comment"
	// Split on " -d " to separate device from the rest
	parts := strings.SplitN(line, " -d ", 2)
	if len(parts) != 2 {
		return "", ""
	}

	device = strings.TrimSpace(parts[0])

	// Split the second part on space and # to get device type
	remaining := strings.TrimSpace(parts[1])
	fields := strings.Fields(remaining)
	if len(fields) > 0 {
		deviceType = fields[0]
	}

	return device, deviceType
}

// tryDirectMacOSDetection tries common macOS device paths with different protocols
func (m *Manager) tryDirectMacOSDetection() []types.DiskInfo {
	var disks []types.DiskInfo

	// Try common macOS device patterns
	commonDevices := []string{
		"/dev/disk0",
		"/dev/disk1",
		"/dev/disk2",
	}

	for _, device := range commonDevices {
		// Try different device types for macOS (order by likelihood)
		deviceTypes := []string{"nvme", "auto", "ata", "scsi"}

		for _, deviceType := range deviceTypes {
			diskInfo := m.getMacOSSmartCtlInfo(device, deviceType)
			if diskInfo.Device != "" {
				diskInfo.Type = fmt.Sprintf("macos-%s", deviceType)
				disks = append(disks, diskInfo)
				log.Printf("Successfully detected %s using %s protocol", device, deviceType)
				break // Found working device type, move to next device
			}
		}
	}

	return disks
}

// createBasicMacOSEntries creates basic disk entries when SMART data is unavailable
func (m *Manager) createBasicMacOSEntries() []types.DiskInfo {
	var disks []types.DiskInfo

	// Get basic disk info from diskutil
	output, err := exec.Command("diskutil", "list", "-physical").Output()
	if err != nil {
		log.Printf("Error running diskutil: %v", err)
		return disks
	}

	lines := strings.Split(string(output), "\n")
	diskCount := 0

	for _, line := range lines {
		// Look for physical disk entries
		if strings.Contains(line, "/dev/disk") && strings.Contains(line, "internal") {
			// Extract disk number using regex
			re := regexp.MustCompile(`(/dev/disk\d+)`)
			matches := re.FindStringSubmatch(line)
			if len(matches) > 1 {
				diskPath := matches[1]

				// Try to get additional info from diskutil info
				diskInfo := m.getDiskUtilInfo(diskPath)
				diskInfo.Type = "macos-basic"

				disks = append(disks, diskInfo)
				diskCount++
				log.Printf("Created basic entry for %s", diskPath)
			}
		}
	}

	log.Printf("Created %d basic disk entries", diskCount)
	return disks
}

// getDiskUtilInfo gets basic disk information using diskutil
func (m *Manager) getDiskUtilInfo(device string) types.DiskInfo {
	disk := types.DiskInfo{
		Device:   device,
		Location: "internal",
		Health:   "Unknown",
		Serial:   "unknown",
		Model:    "macOS Disk",
	}

	// Try to get more detailed info from diskutil
	output, err := exec.Command("diskutil", "info", device).Output()
	if err == nil {
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "Device / Media Name:") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					disk.Model = strings.TrimSpace(parts[1])
				}
			}
		}
	}

	return disk
}

// getMacOSSmartCtlInfo gets SMART information for a specific device on macOS
func (m *Manager) getMacOSSmartCtlInfo(device, deviceType string) types.DiskInfo {
	var disk types.DiskInfo

	// Build smartctl command with device type
	var cmd *exec.Cmd
	if deviceType == "auto" {
		cmd = exec.Command("smartctl", "-a", "-j", device)
	} else {
		cmd = exec.Command("smartctl", "-d", deviceType, "-a", "-j", device)
	}

	output, err := cmd.Output()

	// smartctl exit codes:
	// 0: OK
	// 2: Device open failed or device not supported
	// 4: Some SMART or other ATA command to the disk failed, or checksum error
	// Other codes indicate more serious issues

	if err != nil {
		// Check if it's an ExitError to get the exit code
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode := exitErr.ExitCode()

			// Exit code 4 is common on macOS and often still returns useful data
			if exitCode == 4 && len(output) > 0 {
				log.Printf("smartctl warning for %s (%s): exit code %d but data available", device, deviceType, exitCode)
			} else if exitCode == 2 {
				// Device not supported or can't be opened - don't spam logs for auto detection
				if deviceType != "auto" {
					log.Printf("Device %s not supported with %s protocol (exit code %d)", device, deviceType, exitCode)
				}
				return disk
			} else {
				// Other exit codes
				if deviceType != "auto" {
					log.Printf("smartctl error for %s (%s): exit code %d", device, deviceType, exitCode)
				}
				return disk
			}
		} else {
			// Non-exit error (like command not found)
			if deviceType != "auto" {
				log.Printf("Error running smartctl for %s (%s): %v", device, deviceType, err)
			}
			return disk
		}
	}

	// Try to parse JSON output even if there was an error (exit code 4 case)
	if len(output) == 0 {
		return disk
	}

	var smartData types.SmartCtlOutput
	if err := json.Unmarshal(output, &smartData); err != nil {
		log.Printf("Error parsing smartctl JSON for %s: %v", device, err)
		return disk
	}

	// Populate basic disk information
	disk.Device = device
	disk.Serial = smartData.SerialNumber
	disk.Model = smartData.ModelName
	disk.Location = fmt.Sprintf("%s (%s)", device, deviceType)

	// Handle temperature
	if smartData.Temperature.Current > 0 {
		disk.Temperature = float64(smartData.Temperature.Current)
	}

	// Determine health status
	if smartData.SmartStatus.Passed {
		disk.Health = "OK"
	} else {
		disk.Health = "FAILED"
	}

	// Set additional fields based on device type
	if deviceType == "nvme" {
		m.extractNVMeInfo(&disk, &smartData)
	} else {
		m.extractATAInfo(&disk, &smartData)
	}

	log.Printf("Successfully got SMART data for %s: %s %s (Health: %s, Temp: %.0f°C)",
		device, disk.Model, disk.Serial, disk.Health, disk.Temperature)

	return disk
}

// extractNVMeInfo extracts NVMe-specific information
func (m *Manager) extractNVMeInfo(disk *types.DiskInfo, smartData *types.SmartCtlOutput) {
	disk.Interface = "NVMe"
	disk.SmartEnabled = smartData.SmartSupport.Enabled
	disk.SmartHealthy = smartData.SmartStatus.Passed

	// Extract basic timing information
	if smartData.PowerOnTime.Hours > 0 {
		disk.PowerOnHours = int64(smartData.PowerOnTime.Hours)
	}

	if smartData.PowerCycleCount > 0 {
		disk.PowerCycles = int64(smartData.PowerCycleCount)
	}

	// Set capacity if available
	if smartData.UserCapacity.Bytes > 0 {
		disk.Capacity = smartData.UserCapacity.Bytes
	}

	// Extract NVMe-specific health data if available
	nvmeHealth := smartData.NvmeSmartHealthInformationLog

	// Use NVMe log data if available, fallback to basic data
	if nvmeHealth.PowerOnHours > 0 {
		disk.PowerOnHours = nvmeHealth.PowerOnHours
	}

	if nvmeHealth.PowerCycles > 0 {
		disk.PowerCycles = nvmeHealth.PowerCycles
	}

	// NVMe-specific health metrics
	if nvmeHealth.PercentageUsed > 0 {
		disk.PercentageUsed = nvmeHealth.PercentageUsed
	}

	if nvmeHealth.AvailableSpare > 0 {
		disk.AvailableSpare = nvmeHealth.AvailableSpare
	}

	if nvmeHealth.CriticalWarning > 0 {
		disk.CriticalWarning = nvmeHealth.CriticalWarning
	}

	if nvmeHealth.MediaErrors > 0 {
		disk.MediaErrors = nvmeHealth.MediaErrors
	}

	if nvmeHealth.NumErrLogEntries > 0 {
		disk.ErrorLogEntries = nvmeHealth.NumErrLogEntries
	}

	// Use NVMe temperature if available (often more accurate)
	if nvmeHealth.Temperature > 0 {
		disk.Temperature = float64(nvmeHealth.Temperature)
	}
}

// extractATAInfo extracts ATA/SATA-specific information
func (m *Manager) extractATAInfo(disk *types.DiskInfo, smartData *types.SmartCtlOutput) {
	disk.Interface = "ATA/SATA"
	disk.SmartEnabled = smartData.SmartSupport.Enabled
	disk.SmartHealthy = smartData.SmartStatus.Passed

	if smartData.PowerOnTime.Hours > 0 {
		disk.PowerOnHours = int64(smartData.PowerOnTime.Hours)
	}

	if smartData.PowerCycleCount > 0 {
		disk.PowerCycles = int64(smartData.PowerCycleCount)
	}

	// Set rotation rate for mechanical drives
	if smartData.RotationRate > 0 {
		disk.RPM = smartData.RotationRate
	}

	// Set capacity if available
	if smartData.UserCapacity.Bytes > 0 {
		disk.Capacity = smartData.UserCapacity.Bytes
	}

	// Set form factor if available
	if smartData.FormFactor.Name != "" {
		disk.FormFactor = smartData.FormFactor.Name
	}
}
