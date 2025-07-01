package tools

import (
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"disk-health-exporter/internal/utils"
	"disk-health-exporter/pkg/types"
)

// MegaCLITool represents the MegaCLI tool
type MegaCLITool struct {
	command string // "megacli" or "MegaCli64"
}

// NewMegaCLITool creates a new MegaCLITool instance
func NewMegaCLITool() *MegaCLITool {
	tool := &MegaCLITool{}

	// Determine which command to use
	if utils.CommandExists("MegaCli64") {
		tool.command = "MegaCli64"
	} else if utils.CommandExists("megacli") {
		tool.command = "megacli"
	}

	return tool
}

// IsAvailable checks if MegaCLI is available on the system
func (m *MegaCLITool) IsAvailable() bool {
	return utils.CommandExists("megacli") || utils.CommandExists("MegaCli64")
}

// GetVersion returns the MegaCLI version
func (m *MegaCLITool) GetVersion() string {
	if !m.IsAvailable() {
		return ""
	}

	version, err := utils.GetToolVersion(m.command, "-v")
	if err != nil {
		return "unknown"
	}
	return version
}

// GetName returns the tool name
func (m *MegaCLITool) GetName() string {
	return "MegaCLI"
}

// GetRAIDArrays returns RAID array information detected by MegaCLI
func (m *MegaCLITool) GetRAIDArrays() []types.RAIDInfo {
	var raidArrays []types.RAIDInfo

	if !m.IsAvailable() {
		return raidArrays
	}

	// Get RAID array information
	output, err := exec.Command(m.command, "-LDInfo", "-Lall", "-aALL", "-NoLog").Output()
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

		if strings.Contains(line, "Adapter") && strings.Contains(line, "--") {
			// Extract adapter ID for battery info (format: "Adapter 0 -- Virtual Drive Information:")
			re := regexp.MustCompile(`Adapter (\d+) --`)
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
				currentArray.Size = utils.ParseSizeToBytes(sizeStr)
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
				currentArray.Status = utils.GetRaidStatusValue(state)
				currentArray.Type = "hardware"
				currentArray.Controller = "MegaCLI"

				// Get battery information for this adapter
				if adapterID != "" {
					currentArray.Battery = m.GetBatteryInfo(adapterID)
				}

				if currentArray.ArrayID != "" {
					raidArrays = append(raidArrays, currentArray)
					currentArray = types.RAIDInfo{} // Reset for next array
				}
			}
		}
	}

	log.Printf("Found %d RAID arrays using MegaCLI", len(raidArrays))
	return raidArrays
}

// GetRAIDDisks returns disk information from RAID arrays with utilization calculations
func (m *MegaCLITool) GetRAIDDisks() []types.DiskInfo {
	var disks []types.DiskInfo

	if !m.IsAvailable() {
		return disks
	}

	// First get RAID array information to understand the configuration
	raidArrays := m.GetRAIDArrays()

	// Try to get detailed RAID to Physical Disk mapping first
	output, err := exec.Command(m.command, "-LDPDInfo", "-aALL", "-NoLog").Output()
	if err == nil {
		// Parse the mapping and calculate utilization (detailed version)
		detailedDisks := m.parseDetailedMegaCLIDisks(string(output), raidArrays)
		if len(detailedDisks) > 0 {
			log.Printf("Found %d RAID disks with utilization using MegaCLI", len(detailedDisks))
			return detailedDisks
		}
	}

	// Fallback to basic physical disk information
	output, err = exec.Command(m.command, "-PDList", "-aALL", "-NoLog").Output()
	if err != nil {
		log.Printf("Error executing MegaCLI for disk info: %v", err)
		return disks
	}

	// Parse physical disk information
	lines := strings.Split(string(output), "\n")
	var currentDisk types.DiskInfo
	var enclosure, slot string

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.Contains(line, "Enclosure Device ID:") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				enclosure = strings.TrimSpace(parts[1])
			}
		} else if strings.Contains(line, "Slot Number:") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				slot = strings.TrimSpace(parts[1])
			}
			// Set location when we have both enclosure and slot
			if enclosure != "" && slot != "" {
				currentDisk.Location = fmt.Sprintf("Enc:%s Slot:%s", enclosure, slot)
			}
		} else if strings.Contains(line, "Device Id:") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				currentDisk.Device = strings.TrimSpace(parts[1])
			}
		} else if strings.Contains(line, "Inquiry Data:") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				inquiry := strings.TrimSpace(parts[1])
				// Extract model from inquiry data - first field is typically the model
				fields := strings.Fields(inquiry)
				if len(fields) > 0 {
					currentDisk.Model = fields[0]
				}
			}
		} else if strings.Contains(line, "Firmware state:") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				state := strings.TrimSpace(parts[1])
				currentDisk.Health = state
				currentDisk.Type = "raid"

				if currentDisk.Device != "" {
					// Add basic utilization calculation
					m.calculateBasicMegaCLIUtilization(&currentDisk, raidArrays)
					disks = append(disks, currentDisk)
					currentDisk = types.DiskInfo{} // Reset for next disk
					enclosure = ""                 // Reset enclosure
					slot = ""                      // Reset slot
				}
			}
		}
	}

	log.Printf("Found %d RAID disks using MegaCLI", len(disks))
	return disks
}

// parseDetailedMegaCLIDisks parses detailed MegaCLI output
func (m *MegaCLITool) parseDetailedMegaCLIDisks(output string, raidArrays []types.RAIDInfo) []types.DiskInfo {
	var disks []types.DiskInfo
	lines := strings.Split(output, "\n")
	var currentArray *types.RAIDInfo
	var currentDisk types.DiskInfo
	var enclosure, slot string

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.Contains(line, "Virtual Drive:") {
			// Find the corresponding array from our earlier query
			re := regexp.MustCompile(`Virtual Drive: (\d+)`)
			matches := re.FindStringSubmatch(line)
			if len(matches) > 1 {
				arrayID := matches[1]
				for i, raid := range raidArrays {
					if raid.ArrayID == arrayID {
						currentArray = &raidArrays[i]
						break
					}
				}
			}
		} else if strings.Contains(line, "PD:") && strings.Contains(line, "Information") {
			// Start of a new physical disk in the current array
			currentDisk = types.DiskInfo{}
		} else if strings.Contains(line, "Enclosure Device ID:") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				enclosure = strings.TrimSpace(parts[1])
			}
		} else if strings.Contains(line, "Slot Number:") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				slot = strings.TrimSpace(parts[1])
			}
			if enclosure != "" && slot != "" {
				currentDisk.Location = fmt.Sprintf("Enc:%s Slot:%s", enclosure, slot)
			}
		} else if strings.Contains(line, "Device Id:") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				currentDisk.Device = strings.TrimSpace(parts[1])
			}
		} else if strings.Contains(line, "Coerced Size:") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				sizeStr := strings.TrimSpace(parts[1])
				currentDisk.Capacity = utils.ParseSizeToBytes(sizeStr)
			}
		} else if strings.Contains(line, "Inquiry Data:") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				inquiry := strings.TrimSpace(parts[1])
				fields := strings.Fields(inquiry)
				if len(fields) > 0 {
					currentDisk.Model = fields[0]
				}
			}
		} else if strings.Contains(line, "WWN:") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				currentDisk.Serial = strings.TrimSpace(parts[1])
			}
		} else if strings.Contains(line, "Drive's position:") {
			// Extract RAID position information
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				position := strings.TrimSpace(parts[1])
				currentDisk.RaidPosition = position

				// Extract array ID from position (e.g., "DiskGroup: 0, Span: 0, Arm: 0")
				if strings.Contains(position, "DiskGroup:") {
					re := regexp.MustCompile(`DiskGroup:\s*(\d+)`)
					matches := re.FindStringSubmatch(position)
					if len(matches) > 1 {
						currentDisk.RaidArrayID = matches[1]
						currentDisk.RaidRole = "active"
					}
				}
			}
		} else if strings.Contains(line, "Commissioned Spare") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				spareStatus := strings.TrimSpace(parts[1])
				currentDisk.IsCommissionedSpare = strings.ToLower(spareStatus) == "yes"
				if currentDisk.IsCommissionedSpare {
					currentDisk.RaidRole = "commissioned_spare"
				}
			}
		} else if strings.Contains(line, "Emergency Spare") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				spareStatus := strings.TrimSpace(parts[1])
				currentDisk.IsEmergencySpare = strings.ToLower(spareStatus) == "yes"
				if currentDisk.IsEmergencySpare {
					currentDisk.RaidRole = "emergency_spare"
				}
			}
		} else if strings.Contains(line, "Drive Temperature") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				tempStr := strings.TrimSpace(parts[1])
				re := regexp.MustCompile(`(\d+)C`)
				matches := re.FindStringSubmatch(tempStr)
				if len(matches) > 1 {
					if temp, err := strconv.ParseFloat(matches[1], 64); err == nil {
						currentDisk.Temperature = temp
					}
				}
			}
		} else if strings.Contains(line, "Firmware state:") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				state := strings.TrimSpace(parts[1])
				currentDisk.Health = state
				currentDisk.Type = "raid"

				// Determine disk role based on firmware state
				stateLower := strings.ToLower(state)
				if strings.Contains(stateLower, "online") {
					if currentDisk.RaidRole == "" {
						currentDisk.RaidRole = "active"
					}
				} else if strings.Contains(stateLower, "hotspare") || strings.Contains(stateLower, "hot spare") {
					currentDisk.RaidRole = "hot_spare"
					currentDisk.IsGlobalSpare = true
				} else if strings.Contains(stateLower, "spare") {
					if currentDisk.RaidRole == "" {
						currentDisk.RaidRole = "spare"
					}
				} else if strings.Contains(stateLower, "failed") {
					currentDisk.RaidRole = "failed"
				} else if strings.Contains(stateLower, "rebuild") {
					currentDisk.RaidRole = "rebuilding"
				} else if strings.Contains(stateLower, "unconfigured") {
					currentDisk.RaidRole = "unconfigured"
				}

				// Calculate disk utilization if we have array information
				if currentArray != nil && currentDisk.Device != "" {
					m.calculateDiskUtilization(&currentDisk, currentArray)
				}

				if currentDisk.Device != "" {
					disks = append(disks, currentDisk)
					currentDisk = types.DiskInfo{}
					enclosure = ""
					slot = ""
				}
			}
		}
	}

	return disks
}

// calculateBasicMegaCLIUtilization calculates basic utilization for disks without detailed role information
func (m *MegaCLITool) calculateBasicMegaCLIUtilization(disk *types.DiskInfo, raidArrays []types.RAIDInfo) {
	// Try to determine basic RAID role from health status
	healthLower := strings.ToLower(disk.Health)

	switch {
	case strings.Contains(healthLower, "online"):
		disk.RaidRole = "active"
		// Try to find which array this disk belongs to (basic guess)
		if len(raidArrays) > 0 {
			disk.RaidArrayID = raidArrays[0].ArrayID // Default to first array
			m.calculateDiskUtilization(disk, &raidArrays[0])
		}
	case strings.Contains(healthLower, "spare") || strings.Contains(healthLower, "hotspare"):
		disk.RaidRole = "hot_spare"
		disk.UsedBytes = 0
		disk.AvailableBytes = disk.Capacity
		disk.UsagePercentage = 0.0
		disk.Mountpoint = "SPARE"
		disk.Filesystem = "Hot-Spare"
	case strings.Contains(healthLower, "failed") || strings.Contains(healthLower, "fail"):
		disk.RaidRole = "failed"
		disk.UsedBytes = 0
		disk.AvailableBytes = 0
		disk.UsagePercentage = 0.0
		disk.Mountpoint = "FAILED"
		disk.Filesystem = "Failed-Drive"
	case strings.Contains(healthLower, "unconfigured"):
		disk.RaidRole = "unconfigured"
		disk.UsedBytes = 0
		disk.AvailableBytes = disk.Capacity
		disk.UsagePercentage = 0.0
		disk.Mountpoint = "UNCONFIGURED"
		disk.Filesystem = "Unconfigured"
	default:
		disk.RaidRole = "unknown"
		// For unknown roles, assume basic utilization
		if disk.Capacity > 0 {
			disk.UsagePercentage = 50.0 // Conservative estimate
			disk.UsedBytes = disk.Capacity / 2
			disk.AvailableBytes = disk.Capacity / 2
			disk.Mountpoint = "RAID-UNKNOWN"
			disk.Filesystem = "Unknown-Array"
		}
	}
}

// GetBatteryInfo returns battery information for a specific adapter
func (m *MegaCLITool) GetBatteryInfo(adapterID string) *types.RAIDBatteryInfo {
	if !m.IsAvailable() {
		return nil
	}

	// Get battery information
	output, err := exec.Command(m.command, "-AdpBbuCmd", "-a"+adapterID).Output()
	if err != nil {
		log.Printf("Error executing MegaCLI for battery info: %v", err)
		return nil
	}

	// Parse battery information
	lines := strings.Split(string(output), "\n")
	batteryInfo := &types.RAIDBatteryInfo{
		ToolName: "MegaCLI",
	}

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
		}
	}

	// Only return battery info if we found meaningful data
	if batteryInfo.BatteryType != "" {
		return batteryInfo
	}

	return nil
}

// calculateDiskUtilization calculates how much of a physical disk's capacity is used by the RAID array
func (m *MegaCLITool) calculateDiskUtilization(disk *types.DiskInfo, array *types.RAIDInfo) {
	if disk.Capacity <= 0 {
		return
	}

	// Handle spare drives differently - they are not actively used in arrays
	if disk.RaidRole == "hot_spare" || disk.RaidRole == "spare" ||
		disk.RaidRole == "commissioned_spare" || disk.RaidRole == "emergency_spare" ||
		disk.IsCommissionedSpare || disk.IsEmergencySpare || disk.IsGlobalSpare {
		// Spare drives are reserved but not actively used
		disk.UsedBytes = 0
		disk.AvailableBytes = disk.Capacity
		disk.UsagePercentage = 0.0
		disk.Mountpoint = "SPARE"
		disk.Filesystem = fmt.Sprintf("%s-Spare", strings.ToUpper(string(disk.RaidRole[0]))+disk.RaidRole[1:])
		return
	}

	// Handle failed or rebuilding drives
	if disk.RaidRole == "failed" {
		disk.UsedBytes = 0
		disk.AvailableBytes = 0 // Failed drives have no available space
		disk.UsagePercentage = 0.0
		disk.Mountpoint = "FAILED"
		disk.Filesystem = "Failed-Drive"
		return
	}

	if disk.RaidRole == "rebuilding" {
		// During rebuild, assume partial utilization
		disk.UsagePercentage = 50.0 // Rebuilding state
		disk.Mountpoint = "REBUILDING"
		disk.Filesystem = "Rebuilding-Drive"
	}

	// Handle unconfigured drives
	if disk.RaidRole == "unconfigured" {
		disk.UsedBytes = 0
		disk.AvailableBytes = disk.Capacity
		disk.UsagePercentage = 0.0
		disk.Mountpoint = "UNCONFIGURED"
		disk.Filesystem = "Unconfigured"
		return
	}

	// For active drives in RAID arrays, calculate based on RAID level
	if array.Size <= 0 {
		return
	}

	// Parse RAID level to understand data distribution
	raidLevel := strings.ToLower(array.RaidLevel)
	numDrives := array.NumDrives

	if numDrives <= 0 {
		return
	}

	var usableCapacityPerDisk int64
	var utilizationPercentage float64

	switch {
	case strings.Contains(raidLevel, "raid 0") || strings.Contains(raidLevel, "primary-0"):
		// RAID 0: All disk space is used for data
		usableCapacityPerDisk = array.Size / int64(numDrives)
		utilizationPercentage = 100.0

	case strings.Contains(raidLevel, "raid 1") || strings.Contains(raidLevel, "primary-1"):
		// RAID 1: 50% of disk space is used (mirrored)
		usableCapacityPerDisk = disk.Capacity / 2
		utilizationPercentage = 50.0

	case strings.Contains(raidLevel, "raid 5") || strings.Contains(raidLevel, "primary-5"):
		// RAID 5: (n-1)/n of disk space is used for data, 1/n for parity
		// Each disk contributes equally to the array
		usableCapacityPerDisk = array.Size / int64(numDrives-1) * int64(numDrives) / int64(numDrives)
		utilizationPercentage = float64(numDrives-1) / float64(numDrives) * 100.0

	case strings.Contains(raidLevel, "raid 6") || strings.Contains(raidLevel, "primary-6"):
		// RAID 6: (n-2)/n of disk space is used for data, 2/n for parity
		usableCapacityPerDisk = array.Size / int64(numDrives-2) * int64(numDrives) / int64(numDrives)
		utilizationPercentage = float64(numDrives-2) / float64(numDrives) * 100.0

	case strings.Contains(raidLevel, "raid 10") || strings.Contains(raidLevel, "primary-10"):
		// RAID 10: 50% of disk space is used (striped mirrors)
		usableCapacityPerDisk = disk.Capacity / 2
		utilizationPercentage = 50.0

	default:
		// Unknown RAID level, assume full utilization for active drives
		usableCapacityPerDisk = disk.Capacity
		utilizationPercentage = 100.0
	}

	// Set the calculated values for active drives
	if disk.RaidRole == "active" {
		disk.UsedBytes = usableCapacityPerDisk
		disk.AvailableBytes = disk.Capacity - usableCapacityPerDisk
		disk.UsagePercentage = utilizationPercentage
		disk.Mountpoint = fmt.Sprintf("RAID-%s", array.ArrayID)
		disk.Filesystem = fmt.Sprintf("%s-Array", array.RaidLevel)
	}
}

// GetSpareDisks returns information about spare drives
func (m *MegaCLITool) GetSpareDisks() []types.DiskInfo {
	allDisks := m.GetRAIDDisks()
	var spareDisks []types.DiskInfo

	for _, disk := range allDisks {
		if disk.RaidRole == "hot_spare" || disk.RaidRole == "spare" ||
			disk.RaidRole == "commissioned_spare" || disk.RaidRole == "emergency_spare" ||
			disk.IsCommissionedSpare || disk.IsEmergencySpare || disk.IsGlobalSpare {
			spareDisks = append(spareDisks, disk)
		}
	}

	log.Printf("Found %d spare disks using MegaCLI", len(spareDisks))
	return spareDisks
}

// GetUnconfiguredDisks returns information about unconfigured drives
func (m *MegaCLITool) GetUnconfiguredDisks() []types.DiskInfo {
	allDisks := m.GetRAIDDisks()
	var unconfiguredDisks []types.DiskInfo

	for _, disk := range allDisks {
		if disk.RaidRole == "unconfigured" {
			unconfiguredDisks = append(unconfiguredDisks, disk)
		}
	}

	log.Printf("Found %d unconfigured disks using MegaCLI", len(unconfiguredDisks))
	return unconfiguredDisks
}
