package tools

import (
	"log"
	"os/exec"
	"regexp"
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
		disks = append(disks, controllerDisks...)
	}

	return disks
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
					arrays = append(arrays, currentArray)
					inLogicalDevice = false
				}
			}
		}
	}

	// Add the last array if exists
	if inLogicalDevice && currentArray.ArrayID != "" {
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

	return disks
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
