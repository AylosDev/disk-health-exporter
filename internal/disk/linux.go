package disk

import (
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"strings"

	"disk-health-exporter/pkg/types"
)

// GetLinuxDisks gets all disks on Linux systems
func (m *Manager) GetLinuxDisks() ([]types.DiskInfo, []types.RAIDInfo) {
	raidArrays := m.checkMegaCLI()
	raidDisks := m.getRaidDisks()
	regularDisks := m.getRegularDisks()

	allDisks := append(raidDisks, regularDisks...)
	return allDisks, raidArrays
}

// checkMegaCLI checks for RAID arrays using MegaCLI
func (m *Manager) checkMegaCLI() []types.RAIDInfo {
	var raidArrays []types.RAIDInfo

	// Check if megacli is available
	if !commandExists("megacli") && !commandExists("MegaCli64") {
		log.Println("MegaCLI not found, skipping RAID array check")
		return raidArrays
	}

	cmd := "megacli"
	if commandExists("MegaCli64") {
		cmd = "MegaCli64"
	}

	// Get RAID array information
	output, err := exec.Command(cmd, "-LDInfo", "-Lall", "-aALL", "-NoLog").Output()
	if err != nil {
		log.Printf("Error executing MegaCLI for array info: %v", err)
		return raidArrays
	}

	// Parse RAID array information
	lines := strings.Split(string(output), "\n")
	var currentArray types.RAIDInfo

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.Contains(line, "Virtual Drive:") {
			// Extract array ID
			re := regexp.MustCompile(`Virtual Drive: (\d+)`)
			matches := re.FindStringSubmatch(line)
			if len(matches) > 1 {
				currentArray.ArrayID = matches[1]
			}
		} else if strings.Contains(line, "RAID Level") {
			// Extract RAID level
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				currentArray.RaidLevel = strings.TrimSpace(parts[1])
			}
		} else if strings.Contains(line, "State") {
			// Extract state
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				state := strings.TrimSpace(parts[1])
				currentArray.State = state
				currentArray.Status = getRaidStatusValue(state)

				if currentArray.ArrayID != "" {
					raidArrays = append(raidArrays, currentArray)
					currentArray = types.RAIDInfo{} // Reset for next array
				}
			}
		}
	}

	return raidArrays
}

// getRaidDisks gets physical disks from RAID arrays
func (m *Manager) getRaidDisks() []types.DiskInfo {
	var disks []types.DiskInfo

	// Check if megacli is available
	if !commandExists("megacli") && !commandExists("MegaCli64") {
		return disks
	}

	cmd := "megacli"
	if commandExists("MegaCli64") {
		cmd = "MegaCli64"
	}

	// Get physical disk information
	output, err := exec.Command(cmd, "-PDList", "-aALL", "-NoLog").Output()
	if err != nil {
		log.Printf("Error executing MegaCLI for disk info: %v", err)
		return disks
	}

	// Parse physical disk information
	lines := strings.Split(string(output), "\n")
	var currentDisk types.DiskInfo

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.Contains(line, "Enclosure Device ID:") && strings.Contains(line, "Slot Number:") {
			// Extract enclosure and slot for location
			parts := strings.Fields(line)
			enclosure := ""
			slot := ""
			for i, part := range parts {
				if part == "ID:" && i+1 < len(parts) {
					enclosure = parts[i+1]
				}
				if part == "Number:" && i+1 < len(parts) {
					slot = parts[i+1]
				}
			}
			currentDisk.Location = fmt.Sprintf("Enc:%s Slot:%s", enclosure, slot)
		} else if strings.Contains(line, "Device Id:") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				currentDisk.Device = strings.TrimSpace(parts[1])
			}
		} else if strings.Contains(line, "Inquiry Data:") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				inquiry := strings.TrimSpace(parts[1])
				// Extract model from inquiry data
				fields := strings.Fields(inquiry)
				if len(fields) > 1 {
					currentDisk.Model = strings.Join(fields[1:], " ")
				}
			}
		} else if strings.Contains(line, "Firmware state:") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				state := strings.TrimSpace(parts[1])
				currentDisk.Health = state
				currentDisk.Type = "raid"

				if currentDisk.Device != "" {
					disks = append(disks, currentDisk)
					currentDisk = types.DiskInfo{} // Reset for next disk
				}
			}
		}
	}

	return disks
}

// getRegularDisks gets regular (non-RAID) disks using smartctl
func (m *Manager) getRegularDisks() []types.DiskInfo {
	var disks []types.DiskInfo

	// Check if smartctl is available
	if !commandExists("smartctl") {
		log.Println("smartctl not found, skipping regular disk check")
		return disks
	}

	// Get list of available devices
	output, err := exec.Command("smartctl", "--scan").Output()
	if err != nil {
		log.Printf("Error scanning for devices: %v", err)
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
			// Skip if this device is part of a RAID array
			if !isPartOfSoftwareRAID(device) {
				diskInfo := getSmartCtlInfo(device)
				if diskInfo.Device != "" {
					diskInfo.Type = "regular"
					disks = append(disks, diskInfo)
				}
			}
		}
	}

	return disks
}
