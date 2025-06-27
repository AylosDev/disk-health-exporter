package disk

import (
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"disk-health-exporter/pkg/types"
)

var sizeRegex = regexp.MustCompile(`(?i)^\s*([\d.]+)\s*([a-zA-Z]+)?\s*$`)

// GetLinuxDisks gets all disks on Linux systems using multiple tools
func (m *Manager) GetLinuxDisks() ([]types.DiskInfo, []types.RAIDInfo) {
	var allDisks []types.DiskInfo
	var allRAIDs []types.RAIDInfo

	log.Println("Starting Linux disk detection with multi-tool approach...")

	// Check hardware RAID arrays
	if m.tools.MegaCLI {
		log.Println("Detecting MegaCLI RAID arrays...")
		raidArrays := m.checkMegaCLI()
		raidDisks := m.getRaidDisks()
		allRAIDs = append(allRAIDs, raidArrays...)
		allDisks = append(allDisks, raidDisks...)
		log.Printf("Found %d MegaCLI RAID arrays and %d RAID disks", len(raidArrays), len(raidDisks))
	}

	// Check other hardware RAID controllers
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

	// Check software RAID
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

	// Get regular disks using smartctl and other tools
	log.Println("Detecting regular disks...")
	regularDisks := m.getRegularDisksMultiTool()
	allDisks = append(allDisks, regularDisks...)
	log.Printf("Found %d regular disks", len(regularDisks))

	m.enrichDiskData(allDisks)

	// 6. Apply filtering based on configuration
	log.Println("Applying disk filtering...")
	filteredDisks := m.filterDisks(allDisks)

	log.Printf("Total: %d disks detected, %d disks after filtering, %d RAID arrays detected", len(allDisks), len(filteredDisks), len(allRAIDs))
	return filteredDisks, allRAIDs
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
	var adapterID string

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.Contains(line, "Adapter") && strings.Contains(line, ":") {
			// Extract adapter ID for battery info
			re := regexp.MustCompile(`Adapter (\d+):`)
			matches := re.FindStringSubmatch(line)
			if len(matches) > 1 {
				adapterID = matches[1]
			}
		} else if strings.Contains(line, "Virtual Drive:") {
			// Extract array ID
			re := regexp.MustCompile(`Virtual Drive: (\d+)`)
			matches := re.FindStringSubmatch(line)
			if len(matches) > 1 {
				currentArray.ArrayID = matches[1]
			}
		} else if strings.Contains(line, "RAID Level") {
			// Extract RAID level - handle format like "Primary-5, Secondary-0, RAID Level Qualifier-3"
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				raidLevelStr := strings.TrimSpace(parts[1])
				// Extract the primary RAID level (e.g., "Primary-5" -> "RAID 5")
				if strings.Contains(raidLevelStr, "Primary-") {
					re := regexp.MustCompile(`Primary-(\d+)`)
					matches := re.FindStringSubmatch(raidLevelStr)
					if len(matches) > 1 {
						currentArray.RaidLevel = "RAID " + matches[1]
					} else {
						currentArray.RaidLevel = raidLevelStr
					}
				} else {
					currentArray.RaidLevel = raidLevelStr
				}
			}
		} else if strings.Contains(line, "Size") && strings.Contains(line, ":") && !strings.Contains(line, "Sector Size") && !strings.Contains(line, "Parity Size") && !strings.Contains(line, "Strip Size") {
			// Extract array size - store as string for now since MegaCLI uses human-readable format (e.g., "113.795 TB")
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				sizeStr := strings.TrimSpace(parts[1])
				// Convert human-readable size to bytes if possible, otherwise store as 0
				currentArray.Size = parseSizeToBytes(sizeStr)
			}
		} else if strings.Contains(line, "Number Of Drives") {
			// Extract number of drives
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				numDrivesStr := strings.TrimSpace(parts[1])
				if num, err := strconv.Atoi(numDrivesStr); err == nil && num > 0 {
					currentArray.NumDrives = num
				}
			}
		} else if strings.Contains(line, "State") {
			// Extract state
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				state := strings.TrimSpace(parts[1])
				currentArray.State = state
				currentArray.Status = getRaidStatusValue(state)
				currentArray.Type = "hardware"
				currentArray.Controller = "MegaCLI"

				// Get battery information for this adapter
				if adapterID != "" {
					currentArray.Battery = m.getMegaCLIBatteryInfo(adapterID)
				}

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
			if !seenDevices[disk.Device] && m.shouldIncludeDisk(disk.Device) {
				allDisks = append(allDisks, disk)
				seenDevices[disk.Device] = true
			}
		}
	}

	// Method 2: Use lsblk for device discovery
	if m.tools.Lsblk {
		lsblkDisks := m.getDisksFromLsblk()
		for _, disk := range lsblkDisks {
			if !seenDevices[disk.Device] && m.shouldIncludeDisk(disk.Device) {
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
			if !seenDevices[disk.Device] && m.shouldIncludeDisk(disk.Device) {
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
			// Skip if this device is part of a RAID array or should be ignored
			if !isPartOfSoftwareRAID(device) && m.shouldIncludeDisk(device) {
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

// parseSizeToBytes converts human-readable size strings (e.g., "113.795 TB") to bytes
func parseSizeToBytes(sizeStr string) int64 {
	if sizeStr == "" {
		return 0
	}

	matches := sizeRegex.FindStringSubmatch(sizeStr)
	if len(matches) != 3 {
		return 0
	}

	value, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return 0
	}

	unit := strings.ToUpper(matches[2])
	multipliers := map[string]float64{
		"B":     1,
		"BYTES": 1,
		"KB":    1 << 10,
		"MB":    1 << 20,
		"GB":    1 << 30,
		"TB":    1 << 40,
		"PB":    1 << 50,
	}

	multiplier, ok := multipliers[unit]
	if !ok {
		return 0
	}

	return int64(value * multiplier)
}

// getMegaCLIBatteryInfo gets battery information from MegaCLI
func (m *Manager) getMegaCLIBatteryInfo(adapterID string) *types.RAIDBatteryInfo {
	// Check if megacli is available
	if !commandExists("megacli") && !commandExists("MegaCli64") {
		return nil
	}

	cmd := "megacli"
	if commandExists("MegaCli64") {
		cmd = "MegaCli64"
	}

	// Get battery information
	output, err := exec.Command(cmd, "-AdpBbuCmd", "-a"+adapterID).Output()
	if err != nil {
		log.Printf("Error executing MegaCLI for battery info: %v", err)
		return nil
	}

	// Parse battery information
	lines := strings.Split(string(output), "\n")
	batteryInfo := &types.RAIDBatteryInfo{}

	// Parse adapter ID
	if id, err := strconv.Atoi(adapterID); err == nil {
		batteryInfo.AdapterID = id
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.Contains(line, "BatteryType:") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				batteryInfo.BatteryType = strings.TrimSpace(parts[1])
			}
		} else if strings.Contains(line, "Voltage:") && !strings.Contains(line, "Design Voltage") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				voltageStr := strings.TrimSpace(parts[1])
				// Extract voltage value (e.g., "9481 mV" -> 9481)
				re := regexp.MustCompile(`(\d+)\s*mV`)
				matches := re.FindStringSubmatch(voltageStr)
				if len(matches) > 1 {
					if voltage, err := strconv.Atoi(matches[1]); err == nil {
						batteryInfo.Voltage = voltage
					}
				}
			}
		} else if strings.Contains(line, "Current:") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				currentStr := strings.TrimSpace(parts[1])
				// Extract current value (e.g., "0 mA" -> 0)
				re := regexp.MustCompile(`(\d+)\s*mA`)
				matches := re.FindStringSubmatch(currentStr)
				if len(matches) > 1 {
					if current, err := strconv.Atoi(matches[1]); err == nil {
						batteryInfo.Current = current
					}
				}
			}
		} else if strings.Contains(line, "Temperature:") && !strings.Contains(line, "Temperature                             :") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				tempStr := strings.TrimSpace(parts[1])
				// Extract temperature value (e.g., "35 C" -> 35)
				re := regexp.MustCompile(`(\d+)\s*C`)
				matches := re.FindStringSubmatch(tempStr)
				if len(matches) > 1 {
					if temp, err := strconv.Atoi(matches[1]); err == nil {
						batteryInfo.Temperature = temp
					}
				}
			}
		} else if strings.Contains(line, "Battery State:") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				batteryInfo.State = strings.TrimSpace(parts[1])
			}
		} else if strings.Contains(line, "Charging Status") && strings.Contains(line, ":") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				batteryInfo.ChargingStatus = strings.TrimSpace(parts[1])
			}
		} else if strings.Contains(line, "Voltage                                 :") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				batteryInfo.VoltageStatus = strings.TrimSpace(parts[1])
			}
		} else if strings.Contains(line, "Temperature                             :") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				batteryInfo.TemperatureStatus = strings.TrimSpace(parts[1])
			}
		} else if strings.Contains(line, "Learn Cycle Active") && strings.Contains(line, ":") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				status := strings.TrimSpace(parts[1])
				batteryInfo.LearnCycleActive = strings.ToLower(status) == "yes"
			}
		} else if strings.Contains(line, "Learn Cycle Status") && strings.Contains(line, ":") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				batteryInfo.LearnCycleStatus = strings.TrimSpace(parts[1])
			}
		} else if strings.Contains(line, "Battery Pack Missing") && strings.Contains(line, ":") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				status := strings.TrimSpace(parts[1])
				batteryInfo.BatteryMissing = strings.ToLower(status) == "yes"
			}
		} else if strings.Contains(line, "Battery Replacement required") && strings.Contains(line, ":") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				status := strings.TrimSpace(parts[1])
				batteryInfo.ReplacementRequired = strings.ToLower(status) == "yes"
			}
		} else if strings.Contains(line, "Remaining Capacity Low") && strings.Contains(line, ":") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				status := strings.TrimSpace(parts[1])
				batteryInfo.RemainingCapacityLow = strings.ToLower(status) == "yes"
			}
		} else if strings.Contains(line, "Pack energy") && strings.Contains(line, ":") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				energyStr := strings.TrimSpace(parts[1])
				// Extract energy value (e.g., "233 J" -> 233)
				re := regexp.MustCompile(`(\d+)\s*J`)
				matches := re.FindStringSubmatch(energyStr)
				if len(matches) > 1 {
					if energy, err := strconv.Atoi(matches[1]); err == nil {
						batteryInfo.PackEnergy = energy
					}
				}
			}
		} else if strings.Contains(line, "Capacitance") && strings.Contains(line, ":") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				capacitanceStr := strings.TrimSpace(parts[1])
				if capacitance, err := strconv.Atoi(capacitanceStr); err == nil {
					batteryInfo.Capacitance = capacitance
				}
			}
		} else if strings.Contains(line, "Battery backup charge time") && strings.Contains(line, ":") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				timeStr := strings.TrimSpace(parts[1])
				// Extract hours (e.g., "0 hours" -> 0)
				re := regexp.MustCompile(`(\d+)\s*hours`)
				matches := re.FindStringSubmatch(timeStr)
				if len(matches) > 1 {
					if hours, err := strconv.Atoi(matches[1]); err == nil {
						batteryInfo.BackupChargeTime = hours
					}
				}
			}
		} else if strings.Contains(line, "Date of Manufacture:") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				batteryInfo.ManufactureDate = strings.TrimSpace(parts[1])
			}
		} else if strings.Contains(line, "Design Capacity:") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				capacityStr := strings.TrimSpace(parts[1])
				// Extract capacity value (e.g., "288 J" -> 288)
				re := regexp.MustCompile(`(\d+)\s*J`)
				matches := re.FindStringSubmatch(capacityStr)
				if len(matches) > 1 {
					if capacity, err := strconv.Atoi(matches[1]); err == nil {
						batteryInfo.DesignCapacity = capacity
					}
				}
			}
		} else if strings.Contains(line, "Design Voltage:") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				voltageStr := strings.TrimSpace(parts[1])
				// Extract voltage value (e.g., "9500 mV" -> 9500)
				re := regexp.MustCompile(`(\d+)\s*mV`)
				matches := re.FindStringSubmatch(voltageStr)
				if len(matches) > 1 {
					if voltage, err := strconv.Atoi(matches[1]); err == nil {
						batteryInfo.DesignVoltage = voltage
					}
				}
			}
		} else if strings.Contains(line, "Serial Number:") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				batteryInfo.SerialNumber = strings.TrimSpace(parts[1])
			}
		} else if strings.Contains(line, "Manufacture Name:") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				batteryInfo.ManufactureName = strings.TrimSpace(parts[1])
			}
		} else if strings.Contains(line, "Firmware Version") && strings.Contains(line, ":") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				batteryInfo.FirmwareVersion = strings.TrimSpace(parts[1])
			}
		} else if strings.Contains(line, "Device Name:") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				batteryInfo.DeviceName = strings.TrimSpace(parts[1])
			}
		} else if strings.Contains(line, "Device Chemistry:") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				batteryInfo.DeviceChemistry = strings.TrimSpace(parts[1])
			}
		} else if strings.Contains(line, "Auto Learn Period:") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				periodStr := strings.TrimSpace(parts[1])
				// Extract days (e.g., "27 Days" -> 27)
				re := regexp.MustCompile(`(\d+)\s*Days`)
				matches := re.FindStringSubmatch(periodStr)
				if len(matches) > 1 {
					if days, err := strconv.Atoi(matches[1]); err == nil {
						batteryInfo.AutoLearnPeriod = days
					}
				}
			}
		} else if strings.Contains(line, "Next Learn time:") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				// Join the rest as it contains the full datetime
				batteryInfo.NextLearnTime = strings.TrimSpace(strings.Join(parts[1:], ":"))
			}
		}
	}

	// Only return battery info if we found meaningful data
	if batteryInfo.BatteryType == "" && batteryInfo.State == "" {
		return nil
	}

	return batteryInfo
}
