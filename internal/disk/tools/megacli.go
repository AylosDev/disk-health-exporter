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

	log.Printf("Detecting RAID arrays using MegaCLI...")

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

// GetRAIDDisks returns disk information from RAID arrays
func (m *MegaCLITool) GetRAIDDisks() []types.DiskInfo {
	var disks []types.DiskInfo

	if !m.IsAvailable() {
		return disks
	}

	log.Printf("Detecting RAID disks using MegaCLI...")

	// Get physical disk information
	output, err := exec.Command(m.command, "-PDList", "-aALL", "-NoLog").Output()
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

	log.Printf("Found %d RAID disks using MegaCLI", len(disks))
	return disks
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
