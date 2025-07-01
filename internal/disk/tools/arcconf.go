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

// ArcconfTool represents the arcconf CLI tool for Adaptec RAID controllers
type ArcconfTool struct{}

// NewArcconfTool creates a new ArcconfTool instance
func NewArcconfTool() *ArcconfTool {
	return &ArcconfTool{}
}

// IsAvailable checks if arcconf is available on the system
func (a *ArcconfTool) IsAvailable() bool {
	return utils.CommandExists("arcconf")
}

// GetVersion returns the arcconf version
func (a *ArcconfTool) GetVersion() string {
	if !a.IsAvailable() {
		return ""
	}

	version, err := utils.GetToolVersion("arcconf", "version")
	if err != nil {
		return "unknown"
	}
	return version
}

// GetName returns the tool name
func (a *ArcconfTool) GetName() string {
	return "arcconf"
}

// GetRAIDArrays returns RAID array information detected by arcconf
func (a *ArcconfTool) GetRAIDArrays() []types.RAIDInfo {
	var raidArrays []types.RAIDInfo

	if !a.IsAvailable() {
		return raidArrays
	}

	// Get list of controllers first
	controllers := a.getControllers()

	for _, controllerID := range controllers {
		arrays := a.getArraysForController(controllerID)
		raidArrays = append(raidArrays, arrays...)
	}

	return raidArrays
}

// GetRAIDDisks returns disk information from RAID arrays
func (a *ArcconfTool) GetRAIDDisks() []types.DiskInfo {
	var disks []types.DiskInfo

	if !a.IsAvailable() {
		return disks
	}

	// Get list of controllers first
	controllers := a.getControllers()

	for _, controllerID := range controllers {
		controllerDisks := a.getDisksForController(controllerID)
		// Enrich each disk with SMART data
		for i := range controllerDisks {
			a.enrichRAIDDiskWithSMART(&controllerDisks[i], controllerID)
		}
		disks = append(disks, controllerDisks...)
	}

	return disks
}

// GetBatteryInfo returns battery information for Arcconf controllers
func (a *ArcconfTool) GetBatteryInfo(controllerID string) *types.RAIDBatteryInfo {
	if !a.IsAvailable() {
		return nil
	}

	// Get battery information using Arcconf
	output, err := exec.Command("arcconf", "getconfig", controllerID, "bbu").Output()
	if err != nil {
		// Try alternative command format
		output, err = exec.Command("arcconf", "getconfig", controllerID, "pd").Output()
		if err != nil {
			log.Printf("Error executing arcconf for battery info: %v", err)
			return nil
		}
	}

	// Parse battery information
	batteryInfo := a.parseBatteryInfo(string(output), controllerID)
	return batteryInfo
}

// parseBatteryInfo parses Arcconf battery output
func (a *ArcconfTool) parseBatteryInfo(output, controllerID string) *types.RAIDBatteryInfo {
	// Convert controllerID string to int
	adapterID, err := strconv.Atoi(controllerID)
	if err != nil {
		adapterID = 0
	}

	battery := &types.RAIDBatteryInfo{
		AdapterID: adapterID,
		ToolName:  "Arcconf",
	}

	lines := strings.Split(output, "\n")
	batterySection := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Look for battery section
		if strings.Contains(line, "Battery Information") || strings.Contains(line, "Battery Unit") {
			batterySection = true
			continue
		}

		if !batterySection {
			continue
		}

		if strings.Contains(line, "Battery Type") {
			parts := strings.Split(line, ":")
			if len(parts) >= 2 {
				battery.BatteryType = strings.TrimSpace(parts[1])
			}
		} else if strings.Contains(line, "Voltage") {
			// Extract voltage value
			re := regexp.MustCompile(`(\d+\.?\d*)\s*V`)
			matches := re.FindStringSubmatch(line)
			if len(matches) > 1 {
				if voltage, err := strconv.ParseFloat(matches[1], 64); err == nil {
					battery.Voltage = int(voltage * 1000) // Convert to mV
				}
			}
		} else if strings.Contains(line, "Current") {
			// Extract current value
			re := regexp.MustCompile(`(\d+\.?\d*)\s*A`)
			matches := re.FindStringSubmatch(line)
			if len(matches) > 1 {
				if current, err := strconv.ParseFloat(matches[1], 64); err == nil {
					battery.Current = int(current * 1000) // Convert to mA
				}
			}
		} else if strings.Contains(line, "Temperature") {
			// Extract temperature value
			re := regexp.MustCompile(`(\d+\.?\d*)\s*C`)
			matches := re.FindStringSubmatch(line)
			if len(matches) > 1 {
				if temp, err := strconv.ParseFloat(matches[1], 64); err == nil {
					battery.Temperature = int(temp)
				}
			}
		} else if strings.Contains(line, "Status") || strings.Contains(line, "State") {
			parts := strings.Split(line, ":")
			if len(parts) >= 2 {
				battery.State = strings.TrimSpace(parts[1])
			}
		} else if strings.Contains(line, "Replacement required") {
			battery.ReplacementRequired = strings.Contains(strings.ToLower(line), "yes")
		} else if strings.Contains(line, "Capacity Low") {
			battery.RemainingCapacityLow = strings.Contains(strings.ToLower(line), "yes")
		} else if strings.Contains(line, "Battery pack missing") {
			battery.BatteryMissing = strings.Contains(strings.ToLower(line), "yes")
		}
	}

	return battery
}

// getControllers gets list of available Adaptec controllers
func (a *ArcconfTool) getControllers() []string {
	var controllers []string

	// Get controller list
	output, err := exec.Command("arcconf", "list").Output()
	if err != nil {
		log.Printf("Error getting arcconf controller list: %v", err)
		return controllers
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Look for controller lines like "Controller 1: Adaptec ASR-6805"
		if strings.Contains(line, "Controller") && strings.Contains(line, ":") {
			re := regexp.MustCompile(`Controller\s+(\d+):`)
			matches := re.FindStringSubmatch(line)
			if len(matches) > 1 {
				controllers = append(controllers, matches[1])
			}
		}
	}

	return controllers
}

// getArraysForController gets RAID arrays for a specific controller
func (a *ArcconfTool) getArraysForController(controllerID string) []types.RAIDInfo {
	var arrays []types.RAIDInfo

	// Get logical device info for this controller
	output, err := exec.Command("arcconf", "getconfig", controllerID, "ld").Output()
	if err != nil {
		log.Printf("Error getting arcconf logical devices for controller %s: %v", controllerID, err)
		return arrays
	}

	lines := strings.Split(string(output), "\n")
	var currentArray types.RAIDInfo
	var inLogicalDevice bool

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "Logical device number") {
			// Save previous array if exists
			if inLogicalDevice && currentArray.ArrayID != "" {
				arrays = append(arrays, currentArray)
			}

			// Start new logical device
			currentArray = types.RAIDInfo{
				Controller: "arcconf",
				Type:       "hardware",
			}
			inLogicalDevice = true

			// Extract logical device number
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				currentArray.ArrayID = controllerID + ":" + parts[3]
			}
		} else if inLogicalDevice {
			if strings.Contains(line, "RAID level") {
				// Extract RAID level
				parts := strings.Split(line, ":")
				if len(parts) > 1 {
					raidLevel := strings.TrimSpace(parts[1])
					currentArray.RaidLevel = a.normalizeRAIDLevel(raidLevel)
				}
			} else if strings.Contains(line, "Status of logical device") {
				// Extract status
				parts := strings.Split(line, ":")
				if len(parts) > 1 {
					status := strings.TrimSpace(parts[1])
					currentArray.State = status
					currentArray.Status = a.getRAIDStatusValue(status)
				}
			} else if strings.Contains(line, "Size") {
				// Extract size
				parts := strings.Split(line, ":")
				if len(parts) > 1 {
					sizeStr := strings.TrimSpace(parts[1])
					currentArray.Size = utils.ParseSizeToBytes(sizeStr)
				}
			} else if strings.Contains(line, "Segment size") || strings.HasPrefix(line, "Group") {
				// End of logical device section
				if currentArray.ArrayID != "" {
					// Try to get battery info for this controller
					batteryInfo := a.GetBatteryInfo(controllerID)
					if batteryInfo != nil {
						currentArray.Battery = batteryInfo
					}
					arrays = append(arrays, currentArray)
					inLogicalDevice = false
				}
			}
		}
	}

	// Add the last array if exists
	if inLogicalDevice && currentArray.ArrayID != "" {
		// Try to get battery info for this controller
		batteryInfo := a.GetBatteryInfo(controllerID)
		if batteryInfo != nil {
			currentArray.Battery = batteryInfo
		}
		arrays = append(arrays, currentArray)
	}

	return arrays
}

// getDisksForController gets physical disks for a specific controller
func (a *ArcconfTool) getDisksForController(controllerID string) []types.DiskInfo {
	var disks []types.DiskInfo

	// Get physical device info for this controller
	output, err := exec.Command("arcconf", "getconfig", controllerID, "pd").Output()
	if err != nil {
		log.Printf("Error getting arcconf physical devices for controller %s: %v", controllerID, err)
		return disks
	}

	lines := strings.Split(string(output), "\n")
	var currentDisk types.DiskInfo
	var inPhysicalDevice bool

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.Contains(line, "Device #") {
			// Save previous disk if exists
			if inPhysicalDevice && currentDisk.Device != "" {
				disks = append(disks, currentDisk)
			}

			// Start new physical device
			currentDisk = types.DiskInfo{
				Type: "raid",
			}
			inPhysicalDevice = true

			// Extract device number for identification
			re := regexp.MustCompile(`Device #(\d+)`)
			matches := re.FindStringSubmatch(line)
			if len(matches) > 1 {
				currentDisk.Device = "arcconf:" + controllerID + ":" + matches[1]
			}
		} else if inPhysicalDevice {
			if strings.Contains(line, "Model") {
				parts := strings.Split(line, ":")
				if len(parts) > 1 {
					currentDisk.Model = strings.TrimSpace(parts[1])
				}
			} else if strings.Contains(line, "Serial number") {
				parts := strings.Split(line, ":")
				if len(parts) > 1 {
					currentDisk.Serial = strings.TrimSpace(parts[1])
				}
			} else if strings.Contains(line, "State") {
				parts := strings.Split(line, ":")
				if len(parts) > 1 {
					state := strings.TrimSpace(parts[1])
					currentDisk.Health = state
				}
			} else if strings.Contains(line, "Size") {
				parts := strings.Split(line, ":")
				if len(parts) > 1 {
					sizeStr := strings.TrimSpace(parts[1])
					currentDisk.Capacity = utils.ParseSizeToBytes(sizeStr)
				}
			} else if strings.Contains(line, "Interface") {
				parts := strings.Split(line, ":")
				if len(parts) > 1 {
					currentDisk.Interface = strings.TrimSpace(parts[1])
				}
			} else if strings.Contains(line, "Location") {
				parts := strings.Split(line, ":")
				if len(parts) > 1 {
					currentDisk.Location = strings.TrimSpace(parts[1])
				}
			} else if line == "" {
				// End of physical device section
				if currentDisk.Device != "" {
					disks = append(disks, currentDisk)
					inPhysicalDevice = false
				}
			}
		}
	}

	// Add the last disk if exists
	if inPhysicalDevice && currentDisk.Device != "" {
		disks = append(disks, currentDisk)
	}

	// Enrich disks with SMART data
	for i := range disks {
		a.enrichRAIDDiskWithSMART(&disks[i], controllerID)
	}

	return disks
}

// enrichRAIDDiskWithSMART enriches RAID disk information with SMART data via Arcconf
func (a *ArcconfTool) enrichRAIDDiskWithSMART(disk *types.DiskInfo, controllerID string) {
	if !a.IsAvailable() {
		return
	}

	// Parse device location for Arcconf format (e.g., "Channel 0, Device 0")
	if !strings.Contains(disk.Location, "Channel") {
		return
	}

	// Extract channel and device from location
	re := regexp.MustCompile(`Channel\s+(\d+),\s+Device\s+(\d+)`)
	matches := re.FindStringSubmatch(disk.Location)
	if len(matches) < 3 {
		return
	}

	channel := matches[1]
	device := matches[2]

	// Try to get SMART data via Arcconf
	output, err := exec.Command("arcconf", "getconfig", controllerID, "pd", fmt.Sprintf("%s:%s", channel, device)).Output()
	if err != nil {
		return
	}

	// Parse output for SMART data
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.Contains(line, "Temperature") {
			// Extract temperature value
			re := regexp.MustCompile(`(\d+)\s*C`)
			matches := re.FindStringSubmatch(line)
			if len(matches) > 1 {
				if temp, err := strconv.Atoi(matches[1]); err == nil {
					disk.Temperature = float64(temp)
				}
			}
		} else if strings.Contains(line, "S.M.A.R.T.") {
			disk.SmartEnabled = !strings.Contains(strings.ToLower(line), "disabled")
		} else if strings.Contains(line, "State") || strings.Contains(line, "Status") {
			if strings.Contains(strings.ToLower(line), "online") || strings.Contains(strings.ToLower(line), "optimal") {
				disk.SmartHealthy = true
				disk.Health = "OK"
			} else if strings.Contains(strings.ToLower(line), "failed") || strings.Contains(strings.ToLower(line), "critical") {
				disk.SmartHealthy = false
				disk.Health = "FAILED"
			}
		}
	}
}

// normalizeRAIDLevel converts arcconf RAID level to standard format
func (a *ArcconfTool) normalizeRAIDLevel(raidLevel string) string {
	raidLevel = strings.ToLower(strings.TrimSpace(raidLevel))

	switch raidLevel {
	case "0":
		return "RAID 0"
	case "1":
		return "RAID 1"
	case "5":
		return "RAID 5"
	case "6":
		return "RAID 6"
	case "10", "1+0":
		return "RAID 10"
	case "50", "5+0":
		return "RAID 50"
	case "60", "6+0":
		return "RAID 60"
	default:
		return "RAID " + raidLevel
	}
}

// getRAIDStatusValue converts RAID state to numeric value
func (a *ArcconfTool) getRAIDStatusValue(state string) int {
	state = strings.ToLower(strings.TrimSpace(state))

	switch state {
	case "optimal", "ok":
		return 1
	case "degraded", "rebuilding", "initializing":
		return 2
	case "failed", "offline":
		return 3
	default:
		return 0
	}
}
