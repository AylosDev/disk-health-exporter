package disk

import (
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"strings"

	"disk-health-exporter/pkg/types"
)

// GetLinuxDisks gets all disks on Linux systems using multiple tools
func (m *Manager) GetLinuxDisks() ([]types.DiskInfo, []types.RAIDInfo) {
	var allDisks []types.DiskInfo
	var allRAIDs []types.RAIDInfo

	log.Println("Starting Linux disk detection with multi-tool approach...")

	// 1. Check hardware RAID arrays first
	if m.tools.MegaCLI {
		log.Println("Detecting MegaCLI RAID arrays...")
		raidArrays := m.checkMegaCLI()
		raidDisks := m.getRaidDisks()
		allRAIDs = append(allRAIDs, raidArrays...)
		allDisks = append(allDisks, raidDisks...)
		log.Printf("Found %d MegaCLI RAID arrays and %d RAID disks", len(raidArrays), len(raidDisks))
	}

	// 2. Check other hardware RAID controllers
	if m.tools.Storcli {
		log.Println("Detecting StorCLI RAID arrays...")
		storcliRAIDs, storcliDisks := m.checkStorCLI()
		allRAIDs = append(allRAIDs, storcliRAIDs...)
		allDisks = append(allDisks, storcliDisks...)
		log.Printf("Found %d StorCLI RAID arrays and %d RAID disks", len(storcliRAIDs), len(storcliDisks))
	}

	if m.tools.Arcconf {
		log.Println("Detecting Arcconf RAID arrays...")
		arcconfRAIDs, arcconfDisks := m.checkArcconf()
		allRAIDs = append(allRAIDs, arcconfRAIDs...)
		allDisks = append(allDisks, arcconfDisks...)
		log.Printf("Found %d Arcconf RAID arrays and %d RAID disks", len(arcconfRAIDs), len(arcconfDisks))
	}

	// 3. Check software RAID
	if m.tools.Mdadm {
		log.Println("Detecting software RAID arrays...")
		softwareRAIDs := m.getSoftwareRAIDInfo()
		for _, swRaid := range softwareRAIDs {
			// Convert software RAID to standard RAID format
			raidInfo := types.RAIDInfo{
				ArrayID:         strings.TrimPrefix(swRaid.Device, "/dev/"),
				RaidLevel:       swRaid.Level,
				State:           swRaid.State,
				Status:          getSoftwareRAIDStatusValue(swRaid.State),
				Size:            swRaid.ArraySize,
				NumDrives:       swRaid.TotalDevices,
				NumActiveDrives: swRaid.RaidDevices,
				NumSpareDrives:  len(swRaid.SpareDevices),
				NumFailedDrives: len(swRaid.FailedDevices),
				RebuildProgress: int(swRaid.SyncProgress),
				Type:            "software",
				Controller:      "mdadm",
			}
			allRAIDs = append(allRAIDs, raidInfo)
		}
		log.Printf("Found %d software RAID arrays", len(softwareRAIDs))
	}

	// 4. Get regular disks using smartctl and other tools
	log.Println("Detecting regular disks...")
	regularDisks := m.getRegularDisksMultiTool()
	allDisks = append(allDisks, regularDisks...)
	log.Printf("Found %d regular disks", len(regularDisks))

	// 5. Try to enrich existing disk data with additional tools
	log.Println("Enriching disk data with additional tools...")
	m.enrichDiskData(allDisks)

	log.Printf("Total: %d disks, %d RAID arrays detected", len(allDisks), len(allRAIDs))
	return allDisks, allRAIDs
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

// checkStorCLI checks for RAID arrays using StorCLI (Broadcom)
func (m *Manager) checkStorCLI() ([]types.RAIDInfo, []types.DiskInfo) {
	var raidArrays []types.RAIDInfo
	var disks []types.DiskInfo

	// Check if storcli is available
	cmd := "storcli64"
	if !commandExists(cmd) {
		cmd = "storcli"
		if !commandExists(cmd) {
			return raidArrays, disks
		}
	}

	// Get RAID array information
	output, err := exec.Command(cmd, "/call", "show", "J").Output() // J flag for JSON output
	if err != nil {
		log.Printf("Error executing StorCLI: %v", err)
		return raidArrays, disks
	}

	// Parse StorCLI output (JSON format)
	// Note: StorCLI JSON parsing would be complex, this is a simplified version
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "VD") && strings.Contains(line, "RAID") {
			// This is a simplified parser - real implementation would parse JSON
			fields := strings.Fields(line)
			if len(fields) >= 3 {
				raid := types.RAIDInfo{
					ArrayID:    fields[0],
					RaidLevel:  fields[2],
					State:      "Optimal", // Default assumption
					Status:     1,
					Type:       "hardware",
					Controller: "StorCLI",
				}
				raidArrays = append(raidArrays, raid)
			}
		}
	}

	return raidArrays, disks
}

// checkArcconf checks for RAID arrays using Arcconf (Adaptec)
func (m *Manager) checkArcconf() ([]types.RAIDInfo, []types.DiskInfo) {
	var raidArrays []types.RAIDInfo
	var disks []types.DiskInfo

	if !commandExists("arcconf") {
		return raidArrays, disks
	}

	// Get RAID array information
	output, err := exec.Command("arcconf", "getconfig", "1").Output()
	if err != nil {
		log.Printf("Error executing Arcconf: %v", err)
		return raidArrays, disks
	}

	// Parse Arcconf output (simplified)
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "Logical device number") {
			// Simplified parser for Arcconf
			fields := strings.Fields(line)
			if len(fields) >= 4 {
				raid := types.RAIDInfo{
					ArrayID:    fields[3],
					Type:       "hardware",
					Controller: "Arcconf",
					Status:     1, // Default assumption
				}
				raidArrays = append(raidArrays, raid)
			}
		}
	}

	return raidArrays, disks
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

// getRegularDisksMultiTool gets regular disks using multiple detection methods
func (m *Manager) getRegularDisksMultiTool() []types.DiskInfo {
	var allDisks []types.DiskInfo
	seenDevices := make(map[string]bool)

	// Method 1: Use smartctl scan
	if m.tools.SmartCtl {
		smartDisks := m.getRegularDisksSmartCtl()
		for _, disk := range smartDisks {
			if !seenDevices[disk.Device] {
				allDisks = append(allDisks, disk)
				seenDevices[disk.Device] = true
			}
		}
	}

	// Method 2: Use lsblk for device discovery
	if m.tools.Lsblk {
		lsblkDisks := m.getDisksFromLsblk()
		for _, disk := range lsblkDisks {
			if !seenDevices[disk.Device] {
				// Try to get SMART data for this device
				if m.tools.SmartCtl {
					smartDisk := getSmartCtlInfo(disk.Device)
					if smartDisk.Device != "" {
						smartDisk.Type = "regular"
						allDisks = append(allDisks, smartDisk)
					} else {
						// Add basic disk info even without SMART
						disk.Type = "regular"
						allDisks = append(allDisks, disk)
					}
				} else {
					disk.Type = "regular"
					allDisks = append(allDisks, disk)
				}
				seenDevices[disk.Device] = true
			}
		}
	}

	// Method 3: Try NVMe-specific detection
	if m.tools.Nvme {
		nvmeDisks := m.getNVMeDisks()
		for _, disk := range nvmeDisks {
			if !seenDevices[disk.Device] {
				allDisks = append(allDisks, disk)
				seenDevices[disk.Device] = true
			}
		}
	}

	return allDisks
}

// getRegularDisksSmartCtl gets regular disks using smartctl (original method)
func (m *Manager) getRegularDisksSmartCtl() []types.DiskInfo {
	var disks []types.DiskInfo

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

// getDisksFromLsblk gets disk information using lsblk
func (m *Manager) getDisksFromLsblk() []types.DiskInfo {
	var disks []types.DiskInfo

	output, err := exec.Command("lsblk", "-d", "-o", "NAME,SIZE,MODEL,SERIAL,TRAN", "-n").Output()
	if err != nil {
		log.Printf("Error running lsblk: %v", err)
		return disks
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) >= 2 {
			disk := types.DiskInfo{
				Device: "/dev/" + fields[0],
				Type:   "regular",
			}

			// Parse additional fields if available
			if len(fields) >= 3 {
				disk.Model = fields[2]
			}
			if len(fields) >= 4 {
				disk.Serial = fields[3]
			}
			if len(fields) >= 5 {
				disk.Interface = fields[4]
			}

			disks = append(disks, disk)
		}
	}

	return disks
}

// getNVMeDisks gets NVMe-specific disk information
func (m *Manager) getNVMeDisks() []types.DiskInfo {
	var disks []types.DiskInfo

	// List NVMe devices
	output, err := exec.Command("nvme", "list").Output()
	if err != nil {
		log.Printf("Error running nvme list: %v", err)
		return disks
	}

	lines := strings.Split(string(output), "\n")
	for i, line := range lines {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue // Skip header and empty lines
		}

		fields := strings.Fields(line)
		if len(fields) >= 1 && strings.Contains(fields[0], "/dev/nvme") {
			device := fields[0]

			// Get SMART data for NVMe device
			smartDisk := getSmartCtlInfoWithType(device, "nvme")
			if smartDisk.Device != "" {
				smartDisk.Type = "nvme"
				disks = append(disks, smartDisk)
			} else {
				// Add basic NVMe info
				disk := types.DiskInfo{
					Device:    device,
					Type:      "nvme",
					Interface: "NVMe",
					Health:    "Unknown",
				}
				if len(fields) >= 3 {
					disk.Model = fields[2]
				}
				disks = append(disks, disk)
			}
		}
	}

	return disks
}

// enrichDiskData enriches disk data with information from additional tools
func (m *Manager) enrichDiskData(disks []types.DiskInfo) {
	for i := range disks {
		disk := &disks[i]

		// Try to get additional info with hdparm
		if m.tools.Hdparm && disk.Interface == "" {
			m.enrichWithHdparm(disk)
		}

		// Try to get filesystem information
		m.enrichWithFilesystemInfo(disk)
	}
}

// enrichWithHdparm adds information using hdparm
func (m *Manager) enrichWithHdparm(disk *types.DiskInfo) {
	output, err := exec.Command("hdparm", "-I", disk.Device).Output()
	if err != nil {
		return // hdparm failed, skip
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "Transport:") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				disk.Interface = strings.TrimSpace(parts[1])
			}
		}
	}
}

// enrichWithFilesystemInfo adds filesystem-related information
func (m *Manager) enrichWithFilesystemInfo(disk *types.DiskInfo) {
	// Check if disk has mounted filesystems
	output, err := exec.Command("lsblk", "-o", "NAME,FSTYPE,MOUNTPOINT", disk.Device).Output()
	if err != nil {
		return
	}

	// This is just an example - you could extend this to track filesystem usage
	_ = string(output) // Placeholder for future filesystem metrics
}
