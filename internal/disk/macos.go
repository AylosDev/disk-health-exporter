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

	// On macOS, smartctl works differently. We need to use the proper device format
	if !commandExists("smartctl") {
		log.Println("smartctl not found, cannot detect macOS disks")
		return disks
	}

	// Try to scan for NVMe devices first (common on modern Macs)
	output, err := exec.Command("smartctl", "--scan").Output()
	if err != nil {
		log.Printf("Error scanning for devices: %v", err)
	} else {
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.TrimSpace(line) == "" {
				continue
			}

			fields := strings.Fields(line)
			if len(fields) >= 2 {
				device := fields[0]
				deviceType := fields[1]

				log.Printf("Found device: %s (type: %s)", device, deviceType)

				// Get SMART data for this device
				diskInfo := m.getMacOSSmartCtlInfo(device, deviceType)
				if diskInfo.Device != "" {
					diskInfo.Type = "macos-smart"
					disks = append(disks, diskInfo)
				}
			}
		}
	}

	// If no devices found via scan, try direct approach for common macOS devices
	if len(disks) == 0 {

		// Try common macOS device patterns
		commonDevices := []string{
			"/dev/disk0",
			"/dev/disk1",
		}

		for _, device := range commonDevices {
			// Try different device types for macOS
			deviceTypes := []string{"auto", "ata", "nvme", "scsi"}

			for _, deviceType := range deviceTypes {
				diskInfo := m.getMacOSSmartCtlInfo(device, deviceType)
				if diskInfo.Device != "" {
					diskInfo.Type = fmt.Sprintf("macos-%s", deviceType)
					disks = append(disks, diskInfo)
					break // Found working device type, move to next device
				}
			}
		}
	}

	// If still no disks, create basic entries for available disks
	if len(disks) == 0 {
		log.Println("No SMART data available, creating basic disk entries...")

		// Get basic disk info from diskutil
		output, err := exec.Command("diskutil", "list").Output()
		if err == nil {
			lines := strings.Split(string(output), "\n")
			diskCount := 0

			for _, line := range lines {
				if strings.Contains(line, "/dev/disk") && strings.Contains(line, "internal, physical") {
					// Extract disk number
					re := regexp.MustCompile(`/dev/disk(\d+)`)
					matches := re.FindStringSubmatch(line)
					if len(matches) > 1 {
						diskPath := fmt.Sprintf("/dev/disk%s", matches[1])

						disk := types.DiskInfo{
							Device:   diskPath,
							Type:     "macos-basic",
							Location: "internal",
							Health:   "Unknown",
							Serial:   "unknown",
							Model:    "macOS Disk",
						}
						disks = append(disks, disk)
						diskCount++
					}
				}
			}

			log.Printf("Created %d basic disk entries", diskCount)
		}
	}

	log.Printf("Found %d disks on macOS before filtering", len(disks))

	// Apply filtering based on configuration
	filteredDisks := m.filterDisks(disks)
	log.Printf("Found %d disks on macOS after filtering", len(filteredDisks))
	return filteredDisks
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
	if err != nil {
		// Don't log error for auto detection attempts
		if deviceType != "auto" {
			log.Printf("Error getting smartctl info for %s (%s): %v", device, deviceType, err)
		}
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
	disk.Location = fmt.Sprintf("%s (%s)", device, deviceType)

	if smartData.SmartStatus.Passed {
		disk.Health = "OK"
	} else {
		disk.Health = "FAILED"
	}

	log.Printf("Successfully got SMART data for %s: %s %s (Health: %s)", device, disk.Model, disk.Serial, disk.Health)
	return disk
}
