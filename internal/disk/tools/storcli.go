package tools

import (
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"disk-health-exporter/internal/utils"
	"disk-health-exporter/pkg/types"
)

// Ensure StoreCLITool implements the CombinedToolInterface
var _ CombinedToolInterface = (*StoreCLITool)(nil)

// StoreCLITool represents the StoreCLI tool (Broadcom)
type StoreCLITool struct {
	command string // "storcli64" or "storcli"
}

// NewStoreCLITool creates a new StoreCLITool instance
func NewStoreCLITool() *StoreCLITool {
	tool := &StoreCLITool{}

	// Determine which command to use
	if utils.CommandExists("storcli64") {
		tool.command = "storcli64"
	} else if utils.CommandExists("storcli") {
		tool.command = "storcli"
	}

	return tool
}

// IsAvailable checks if StoreCLI is available on the system
func (s *StoreCLITool) IsAvailable() bool {
	return utils.CommandExists("storcli64") || utils.CommandExists("storcli")
}

// GetVersion returns the StoreCLI version
func (s *StoreCLITool) GetVersion() string {
	if !s.IsAvailable() {
		return ""
	}

	version, err := utils.GetToolVersion(s.command, "version")
	if err != nil {
		return "unknown"
	}
	return version
}

// GetName returns the tool name
func (s *StoreCLITool) GetName() string {
	return "StoreCLI"
}

// GetRAIDArrays returns RAID array information detected by StoreCLI
func (s *StoreCLITool) GetRAIDArrays() []types.RAIDInfo {
	var raidArrays []types.RAIDInfo

	if !s.IsAvailable() {
		return raidArrays
	}

	// Get RAID array information using JSON output
	output, err := exec.Command(s.command, "/call", "show", "J").Output()
	if err != nil {
		return s.getRAIDArraysPlainText()
	}

	// Parse JSON output
	arrays := s.parseStoreCLIJSON(output)
	if len(arrays) > 0 {
		return arrays
	}

	return s.getRAIDArraysPlainText()
}

// GetDisks returns disk information detected by StoreCLI (implements DiskToolInterface)
func (s *StoreCLITool) GetDisks() []types.DiskInfo {
	return s.GetRAIDDisks()
}

// GetRAIDDisks returns disk information from RAID arrays with utilization calculations
func (s *StoreCLITool) GetRAIDDisks() []types.DiskInfo {
	var disks []types.DiskInfo

	if !s.IsAvailable() {
		return disks
	}

	// First get RAID array information to understand the configuration
	raidArrays := s.GetRAIDArrays()

	// Try to get physical disk information with detailed parsing first
	output, err := exec.Command(s.command, "/call", "/eall", "/sall", "show", "all").Output()
	if err == nil {
		// Parse the detailed output for disk roles and spare information
		detailedDisks := s.parseStoreCLIDisksWithRoles(string(output), raidArrays)
		if len(detailedDisks) > 0 {
			return detailedDisks
		}
	}

	// Fallback to JSON parsing
	output, err = exec.Command(s.command, "/call", "/eall", "/sall", "show", "J").Output()
	if err != nil {
		log.Printf("Error executing StoreCLI for disk info: %v", err)
		return s.getRAIDDisksPlainText()
	}

	// Parse JSON output for disks
	disks = s.parseStoreCLIDisksJSON(output)
	if len(disks) > 0 {
		// Enrich each disk with SMART data and utilization
		for i := range disks {
			s.enrichRAIDDiskWithSMART(&disks[i])
			// Add basic utilization calculation for JSON-parsed disks
			s.calculateBasicUtilization(&disks[i], raidArrays)
		}
		return disks
	}

	// Final fallback to plain text parsing
	disks = s.getRAIDDisksPlainText()
	// Enrich plain text disks with SMART data too
	for i := range disks {
		s.enrichRAIDDiskWithSMART(&disks[i])
		s.calculateBasicUtilization(&disks[i], raidArrays)
	}
	return disks
}

// GetBatteryInfo returns battery information for StoreCLI controllers
func (s *StoreCLITool) GetBatteryInfo(controllerID string) *types.RAIDBatteryInfo {
	if !s.IsAvailable() {
		return nil
	}

	// Get battery information using StoreCLI
	output, err := exec.Command(s.command, fmt.Sprintf("/c%s", controllerID), "/bbu", "show", "all").Output()
	if err != nil {
		// Try alternative command format
		output, err = exec.Command(s.command, fmt.Sprintf("/c%s", controllerID), "show", "bbu").Output()
		if err != nil {
			log.Printf("Error executing StoreCLI for battery info: %v", err)
			return nil
		}
	}

	// Parse battery information
	batteryInfo := s.parseBatteryInfo(string(output), controllerID)
	return batteryInfo
}

// parseBatteryInfo parses StoreCLI battery output
func (s *StoreCLITool) parseBatteryInfo(output, controllerID string) *types.RAIDBatteryInfo {
	// Convert controllerID string to int
	adapterID, err := strconv.Atoi(controllerID)
	if err != nil {
		adapterID = 0
	}

	battery := &types.RAIDBatteryInfo{
		AdapterID: adapterID,
		ToolName:  "StoreCLI",
	}

	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Parse basic BBU_Info section
		if strings.Contains(line, "Type") && strings.Contains(line, "BBU") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				battery.BatteryType = parts[len(parts)-1]
			}
		} else if strings.Contains(line, "Voltage") && strings.Contains(line, "mV") {
			// Extract voltage: "Voltage       3932 mV"
			re := regexp.MustCompile(`Voltage\s+(\d+)\s*mV`)
			matches := re.FindStringSubmatch(line)
			if len(matches) > 1 {
				if voltage, err := strconv.Atoi(matches[1]); err == nil {
					battery.Voltage = voltage
				}
			}
		} else if strings.Contains(line, "Current") && strings.Contains(line, "mA") {
			// Extract current: "Current       0 mA"
			re := regexp.MustCompile(`Current\s+(\d+)\s*mA`)
			matches := re.FindStringSubmatch(line)
			if len(matches) > 1 {
				if current, err := strconv.Atoi(matches[1]); err == nil {
					battery.Current = current
				}
			}
		} else if strings.Contains(line, "Temperature") && strings.Contains(line, "C") {
			// Extract temperature: "Temperature   28 C"
			re := regexp.MustCompile(`Temperature\s+(\d+)\s*C`)
			matches := re.FindStringSubmatch(line)
			if len(matches) > 1 {
				if temp, err := strconv.Atoi(matches[1]); err == nil {
					battery.Temperature = temp
				}
			}
		} else if strings.Contains(line, "Battery State") {
			// Extract battery state: "Battery State Optimal"
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				battery.State = parts[2]
			}
		} else if strings.Contains(line, "Battery Pack Missing") {
			battery.BatteryMissing = strings.Contains(line, "Yes")
		} else if strings.Contains(line, "Replacement required") {
			battery.ReplacementRequired = strings.Contains(line, "Yes")
		} else if strings.Contains(line, "Remaining Capacity Low") {
			battery.RemainingCapacityLow = strings.Contains(line, "Yes")
		} else if strings.Contains(line, "Learn Cycle Active") {
			battery.LearnCycleActive = strings.Contains(line, "Yes")
		} else if strings.Contains(line, "Learn Cycle Status") {
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				battery.LearnCycleStatus = parts[3]
			}
		} else if strings.Contains(line, "Remaining Capacity") && strings.Contains(line, "mAh") {
			// Extract remaining capacity: "Remaining Capacity       611 mAh"
			re := regexp.MustCompile(`Remaining Capacity\s+(\d+)\s*mAh`)
			matches := re.FindStringSubmatch(line)
			if len(matches) > 1 {
				if capacity, err := strconv.Atoi(matches[1]); err == nil {
					battery.PackEnergy = capacity // Using PackEnergy field for mAh
				}
			}
		} else if strings.Contains(line, "Full Charge Capacity") && strings.Contains(line, "mAh") {
			// Extract full charge capacity: "Full Charge Capacity     611 mAh"
			re := regexp.MustCompile(`Full Charge Capacity\s+(\d+)\s*mAh`)
			matches := re.FindStringSubmatch(line)
			if len(matches) > 1 {
				if capacity, err := strconv.Atoi(matches[1]); err == nil {
					battery.DesignCapacity = capacity
				}
			}
		} else if strings.Contains(line, "Battery backup charge time") && strings.Contains(line, "hour") {
			// Extract backup charge time: "Battery backup charge time 0 hour(s)"
			re := regexp.MustCompile(`Battery backup charge time\s+(\d+)\s*hour`)
			matches := re.FindStringSubmatch(line)
			if len(matches) > 1 {
				if hours, err := strconv.Atoi(matches[1]); err == nil {
					battery.BackupChargeTime = hours
				}
			}
		} else if strings.Contains(line, "Auto Learn Period") {
			// Extract auto learn period: "Auto Learn Period    90d (7776000 seconds)"
			re := regexp.MustCompile(`Auto Learn Period\s+(\d+)d`)
			matches := re.FindStringSubmatch(line)
			if len(matches) > 1 {
				if days, err := strconv.Atoi(matches[1]); err == nil {
					battery.AutoLearnPeriod = days
				}
			}
		} else if strings.Contains(line, "Design Capacity") && strings.Contains(line, "mAh") {
			// Extract design capacity: "Design Capacity         0 mAh"
			re := regexp.MustCompile(`Design Capacity\s+(\d+)\s*mAh`)
			matches := re.FindStringSubmatch(line)
			if len(matches) > 1 {
				if capacity, err := strconv.Atoi(matches[1]); err == nil {
					if capacity > 0 { // Only use if non-zero
						battery.DesignCapacity = capacity
					}
				}
			}
		} else if strings.Contains(line, "Design Voltage") && strings.Contains(line, "mV") {
			// Extract design voltage: "Design Voltage          0 mV"
			re := regexp.MustCompile(`Design Voltage\s+(\d+)\s*mV`)
			matches := re.FindStringSubmatch(line)
			if len(matches) > 1 {
				if voltage, err := strconv.Atoi(matches[1]); err == nil {
					if voltage > 0 { // Only use if non-zero
						battery.DesignVoltage = voltage
					}
				}
			}
		} else if strings.Contains(line, "Serial Number") {
			parts := strings.Fields(line)
			if len(parts) >= 3 && parts[2] != "0" {
				battery.SerialNumber = parts[2]
			}
		} else if strings.Contains(line, "Manufacture Name") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				battery.ManufactureName = strings.Join(parts[2:], " ")
			}
		}
	}

	return battery
}

// parseStoreCLIJSON parses StoreCLI JSON output for RAID arrays
func (s *StoreCLITool) parseStoreCLIJSON(output []byte) []types.RAIDInfo {
	var raidArrays []types.RAIDInfo

	// StoreCLI JSON structure is complex, we'll parse it carefully
	var jsonData map[string]interface{}
	if err := json.Unmarshal(output, &jsonData); err != nil {
		return raidArrays
	}

	// Navigate through the JSON structure to find controllers and VDs
	controllers, ok := jsonData["Controllers"].([]interface{})
	if !ok {
		return raidArrays
	}

	for i, controller := range controllers {
		ctrlMap, ok := controller.(map[string]interface{})
		if !ok {
			continue
		}

		// Extract controller information
		var controllerName string
		var controllerID = fmt.Sprintf("%d", i) // Use index as controller ID
		if responseData, exists := ctrlMap["Response Data"]; exists {
			if dataMap, ok := responseData.(map[string]interface{}); ok {
				if productName, exists := dataMap["Product Name"]; exists {
					controllerName = fmt.Sprintf("StoreCLI - %v", productName)
				}
			}
		}

		// Look for Virtual Drives (RAID arrays)
		if responseData, exists := ctrlMap["Response Data"]; exists {
			if dataMap, ok := responseData.(map[string]interface{}); ok {
				if vdList, exists := dataMap["VD LIST"]; exists {
					if vdArray, ok := vdList.([]interface{}); ok {
						for _, vd := range vdArray {
							if vdMap, ok := vd.(map[string]interface{}); ok {
								raid := s.parseVirtualDrive(vdMap, controllerName)
								if raid.ArrayID != "" {
									// Try to get battery info for this controller
									batteryInfo := s.GetBatteryInfo(controllerID)
									if batteryInfo != nil {
										raid.Battery = batteryInfo
									}
									raidArrays = append(raidArrays, raid)
								}
							}
						}
					}
				}
			}
		}
	}

	return raidArrays
}

// parseVirtualDrive parses a single virtual drive from JSON
func (s *StoreCLITool) parseVirtualDrive(vdMap map[string]interface{}, controllerName string) types.RAIDInfo {
	raid := types.RAIDInfo{
		Type:       "hardware",
		Controller: controllerName,
	}

	if vdID, exists := vdMap["DG/VD"]; exists {
		raid.ArrayID = fmt.Sprintf("%v", vdID)
	}

	if raidLevel, exists := vdMap["TYPE"]; exists {
		raid.RaidLevel = fmt.Sprintf("%v", raidLevel)
	}

	if state, exists := vdMap["State"]; exists {
		stateStr := fmt.Sprintf("%v", state)
		raid.State = stateStr
		raid.Status = utils.GetRaidStatusValue(stateStr)
	}

	if size, exists := vdMap["Size"]; exists {
		if sizeStr := fmt.Sprintf("%v", size); sizeStr != "" {
			raid.Size = utils.ParseSizeToBytes(sizeStr)
		}
	}

	if numDrives, exists := vdMap["#DRIVES"]; exists {
		if drives, ok := numDrives.(float64); ok {
			raid.NumDrives = int(drives)
		}
	}

	return raid
}

// parseStoreCLIDisksJSON parses StoreCLI JSON output for physical disks
func (s *StoreCLITool) parseStoreCLIDisksJSON(output []byte) []types.DiskInfo {
	var disks []types.DiskInfo

	var jsonData map[string]interface{}
	if err := json.Unmarshal(output, &jsonData); err != nil {
		log.Printf("Error parsing StoreCLI disk JSON: %v", err)
		return disks
	}

	// Navigate through JSON to find physical drives
	controllers, ok := jsonData["Controllers"].([]interface{})
	if !ok {
		return disks
	}

	for controllerIndex, controller := range controllers {
		ctrlMap, ok := controller.(map[string]interface{})
		if !ok {
			continue
		}

		// Extract controller ID - try from response data first, fallback to index
		controllerID := fmt.Sprintf("%d", controllerIndex)
		if responseData, exists := ctrlMap["Response Data"]; exists {
			if dataMap, ok := responseData.(map[string]interface{}); ok {
				// Try to get actual controller ID from response data
				if ctrlIDField, exists := dataMap["Controller"]; exists {
					if ctrlIDStr := fmt.Sprintf("%v", ctrlIDField); ctrlIDStr != "" {
						controllerID = ctrlIDStr
					}
				}

				// Look for Drive Information
				if driveInfo, exists := dataMap["Drive Information"]; exists {
					if driveArray, ok := driveInfo.([]interface{}); ok {
						for _, drive := range driveArray {
							if driveMap, ok := drive.(map[string]interface{}); ok {
								disk := s.parsePhysicalDrive(driveMap, controllerID)
								if disk.Device != "" {
									disks = append(disks, disk)
								}
							}
						}
					}
				}
			}
		}
	}

	return disks
}

// parsePhysicalDrive parses a single physical drive from JSON
func (s *StoreCLITool) parsePhysicalDrive(driveMap map[string]interface{}, controllerID string) types.DiskInfo {
	disk := types.DiskInfo{
		Type: "raid",
	}

	if eid, exists := driveMap["EID:Slt"]; exists {
		disk.Location = fmt.Sprintf("Controller:%s EID:Slt %v", controllerID, eid)
		// Convert "252:0" to "raid-c0-enc252-slot0" for better readability
		eidStr := fmt.Sprintf("%v", eid)
		parts := strings.Split(eidStr, ":")
		if len(parts) == 2 {
			disk.Device = fmt.Sprintf("raid-c%s-enc%s-slot%s", controllerID, parts[0], parts[1])
		} else {
			disk.Device = fmt.Sprintf("raid-c%s-drive-%v", controllerID, eid)
		}
	}

	if model, exists := driveMap["Model"]; exists {
		disk.Model = fmt.Sprintf("%v", model)
	}

	if serial, exists := driveMap["SN"]; exists {
		disk.Serial = fmt.Sprintf("%v", serial)
	}

	if state, exists := driveMap["State"]; exists {
		disk.Health = fmt.Sprintf("%v", state)
	}

	if size, exists := driveMap["Size"]; exists {
		if sizeStr := fmt.Sprintf("%v", size); sizeStr != "" {
			disk.Capacity = utils.ParseSizeToBytes(sizeStr)
		}
	}

	if mediaType, exists := driveMap["Med"]; exists {
		disk.Interface = fmt.Sprintf("%v", mediaType)
	}

	return disk
}

// getRAIDArraysPlainText fallback method using plain text parsing
func (s *StoreCLITool) getRAIDArraysPlainText() []types.RAIDInfo {
	var raidArrays []types.RAIDInfo

	// Get RAID array information without JSON
	output, err := exec.Command(s.command, "/call", "show").Output()
	if err != nil {
		log.Printf("Error executing StoreCLI for plain text array info: %v", err)
		return raidArrays
	}

	// Parse plain text output (simplified)
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Look for virtual drive entries (simplified pattern)
		if strings.Contains(line, "VD") && strings.Contains(line, "RAID") {
			fields := strings.Fields(line)
			if len(fields) >= 3 {
				raid := types.RAIDInfo{
					ArrayID:    fields[0],
					RaidLevel:  fields[2],
					State:      "Optimal", // Default assumption
					Status:     1,
					Type:       "hardware",
					Controller: "StoreCLI",
				}

				// Try to extract more information if available
				for i, field := range fields {
					if strings.Contains(field, "GB") || strings.Contains(field, "TB") {
						raid.Size = utils.ParseSizeToBytes(field)
					}
					if field == "Optimal" || field == "Degraded" || field == "Failed" {
						raid.State = field
						raid.Status = utils.GetRaidStatusValue(field)
					}
					if strings.Contains(field, "RAID") && i+1 < len(fields) {
						raid.RaidLevel = field + " " + fields[i+1]
					}
				}

				raidArrays = append(raidArrays, raid)
			}
		}
	}

	return raidArrays
}

// getRAIDDisksPlainText fallback method for physical disks using plain text parsing
func (s *StoreCLITool) getRAIDDisksPlainText() []types.DiskInfo {
	var disks []types.DiskInfo

	// Compile regex once for better performance
	driveRegex := regexp.MustCompile(`^\d+:\d+\s+\d+`)

	// Get physical disk information
	output, err := exec.Command(s.command, "/call", "/eall", "/sall", "show").Output()
	if err != nil {
		log.Printf("Error executing StoreCLI for plain text disk info: %v", err)
		return disks
	}

	// Parse physical disk information (simplified)
	lines := strings.Split(string(output), "\n")
	var currentController = "0" // Default controller

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Try to detect controller information from the output
		if strings.Contains(line, "Controller =") {
			// Extract controller number: "Controller = 0"
			parts := strings.Split(line, "=")
			if len(parts) > 1 {
				currentController = strings.TrimSpace(parts[1])
			}
		}

		// Look for drive entries in the format: EID:Slt DID State DG Size Intf Med SED PI SeSz Model
		if driveRegex.MatchString(line) {
			fields := strings.Fields(line)
			if len(fields) >= 6 {
				// Convert "252:0" to "raid-c0-enc252-slot0" for consistency
				eidSlot := fields[0]
				parts := strings.Split(eidSlot, ":")
				var deviceName string
				if len(parts) == 2 {
					deviceName = fmt.Sprintf("raid-c%s-enc%s-slot%s", currentController, parts[0], parts[1])
				} else {
					deviceName = fmt.Sprintf("raid-c%s-drive-%s", currentController, eidSlot)
				}

				disk := types.DiskInfo{
					Device:   deviceName,
					Location: fmt.Sprintf("Controller:%s EID:Slt %s", currentController, eidSlot),
					Health:   fields[2],
					Type:     "raid",
				}

				// Extract size
				if len(fields) > 4 {
					disk.Capacity = utils.ParseSizeToBytes(fields[4])
				}

				// Extract interface
				if len(fields) > 5 {
					disk.Interface = fields[5]
				}

				// Extract model (usually at the end)
				if len(fields) > 9 {
					disk.Model = strings.Join(fields[9:], " ")
				}

				disks = append(disks, disk)
			}
		}
	}

	return disks
}

// enrichRAIDDiskWithSMART enriches RAID disk information with SMART data via StoreCLI
func (s *StoreCLITool) enrichRAIDDiskWithSMART(disk *types.DiskInfo) {
	if !s.IsAvailable() {

		return
	}

	// Extract controller and slot info from device name (raid-c0-enc64-slot3)
	if !strings.HasPrefix(disk.Device, "raid-c") {
		return
	}

	// Parse controller, enclosure and slot from device name
	parts := strings.Split(disk.Device, "-")
	if len(parts) < 5 {
		return
	}

	controller := strings.TrimPrefix(parts[1], "c")
	enclosure := strings.TrimPrefix(parts[2], "enc")
	slot := strings.TrimPrefix(parts[3], "slot")

	// Try to get SMART data via StoreCLI
	cmd := fmt.Sprintf("/c%s/e%s/s%s", controller, enclosure, slot)
	output, err := exec.Command(s.command, cmd, "show", "all").Output()
	if err != nil {
		// Try alternative command format
		output, err = exec.Command(s.command, cmd, "show").Output()
		if err != nil {
			return
		}
	}

	// Parse output for SMART data
	lines := strings.Split(string(output), "\n")
	smartDataFound := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.Contains(line, "Drive Temperature") {
			// Extract temperature value
			re := regexp.MustCompile(`(\d+)C`)
			matches := re.FindStringSubmatch(line)
			if len(matches) > 1 {
				if temp, err := strconv.Atoi(matches[1]); err == nil {
					disk.Temperature = float64(temp)
				}
			}
		} else if strings.Contains(line, "S.M.A.R.T alert") {
			smartDataFound = true
			disk.SmartEnabled = !strings.Contains(strings.ToLower(line), "no")
		} else if strings.Contains(line, "Drive health") || strings.Contains(line, "State") {
			smartDataFound = true
			lowerLine := strings.ToLower(line)
			if strings.Contains(lowerLine, "online") || strings.Contains(lowerLine, "optimal") || strings.Contains(lowerLine, "good") {
				disk.SmartHealthy = true
				disk.Health = "OK"
			} else if strings.Contains(lowerLine, "failed") || strings.Contains(lowerLine, "critical") || strings.Contains(lowerLine, "bad") {
				disk.SmartHealthy = false
				disk.Health = "FAILED"
			} else {
				// For unknown states, assume healthy unless explicitly failed
				disk.SmartHealthy = true
				disk.Health = "UNKNOWN"
			}
		}
	}

	if !smartDataFound {
		// If no explicit SMART data found, but disk.Health is already set from parsing, use that
		if disk.Health == "Onln" || strings.Contains(strings.ToLower(disk.Health), "online") {
			disk.SmartHealthy = true
			disk.SmartEnabled = true // Assume SMART is available through RAID controller
		}
	}
}

// parseStoreCLIDisksWithRoles parses StoreCLI output to identify disk roles and spare drives
func (s *StoreCLITool) parseStoreCLIDisksWithRoles(output string, raidArrays []types.RAIDInfo) []types.DiskInfo {
	lines := strings.Split(output, "\n")

	// First, try to parse the summary table format (like the output you provided)
	if summaryDisks := s.parseStoreCLISummaryTable(lines, raidArrays); len(summaryDisks) > 0 {
		return summaryDisks
	}

	// Fallback to detailed per-drive parsing
	detailedDisks := s.parseStoreCLIDetailedFormat(lines, raidArrays)
	return detailedDisks
}

// parseStoreCLISummaryTable parses the table format output from StoreCLI
func (s *StoreCLITool) parseStoreCLISummaryTable(lines []string, raidArrays []types.RAIDInfo) []types.DiskInfo {
	var disks []types.DiskInfo
	var inDriveTable bool
	var currentController = "0" // Default controller, will be updated when we detect controller sections

	for i, line := range lines {
		line = strings.TrimSpace(line)

		// Try to detect controller information from the output
		if strings.Contains(line, "Controller =") {
			// Extract controller number: "Controller = 0"
			parts := strings.Split(line, "=")
			if len(parts) > 1 {
				currentController = strings.TrimSpace(parts[1])
			}
		}

		// Detect the start of the drive table (header line)
		if strings.Contains(line, "EID:Slt DID State DG") && strings.Contains(line, "Size") && strings.Contains(line, "Model") {
			inDriveTable = true
			continue
		}

		// Detect the end of the drive table - but don't break, continue looking for more tables
		if inDriveTable && strings.Contains(line, "--------") && i < len(lines)-1 {
			// Check if this is the closing line by looking for separator or explanation lines
			nextLine := strings.TrimSpace(lines[i+1])
			if nextLine == "" || strings.Contains(nextLine, "EID=Enclosure") || strings.Contains(nextLine, "DHS=Dedicated") {
				inDriveTable = false
				continue
			}
		}

		// Parse drive data lines
		if inDriveTable && strings.Contains(line, ":") && !strings.Contains(line, "EID:Slt") {
			disk := s.parseStoreCLITableLine(line, raidArrays, currentController)
			if disk.Device != "" {
				disks = append(disks, disk)
			}
		}
	}

	return disks
}

// parseStoreCLITableLine parses a single line from the StoreCLI drive table
func (s *StoreCLITool) parseStoreCLITableLine(line string, raidArrays []types.RAIDInfo, controllerID string) types.DiskInfo {
	fields := strings.Fields(line)
	if len(fields) < 8 {
		return types.DiskInfo{} // Invalid line
	}

	// Example line: "64:3      3 Onln   0 893.750 GB SATA SSD N   N  512B SAMSUNG MZ7KM960HAHP-00005 U  -"
	eidSlt := fields[0]     // "64:3"
	state := fields[2]      // "Onln"
	driveGroup := fields[3] // "0"
	size := fields[4]       // "893.750"
	unit := fields[5]       // "GB"
	intf := fields[6]       // "SATA"

	// Parse EID and Slot
	eidSlotParts := strings.Split(eidSlt, ":")
	if len(eidSlotParts) != 2 {
		return types.DiskInfo{} // Invalid EID:Slot format
	}

	disk := types.DiskInfo{
		Device:    fmt.Sprintf("raid-c%s-enc%s-slot%s", controllerID, eidSlotParts[0], eidSlotParts[1]),
		Location:  fmt.Sprintf("Controller:%s EID:%s Slot:%s", controllerID, eidSlotParts[0], eidSlotParts[1]),
		Health:    state,
		Type:      "raid",
		Interface: intf,
		Capacity:  utils.ParseSizeToBytes(size + " " + unit),
	}

	// Parse model using field position logic
	// Format: EID:Slt DID State DG Size Unit Intf Med SED PI SeSz Model... Sp Type
	// Fields 0-10 are fixed, Model starts at field 11, ends 2 fields before the end
	if len(fields) >= 13 { // Minimum: 11 fixed fields + model + Sp + Type
		modelStart := 11
		modelEnd := len(fields) - 2 // Exclude Sp and Type at the end
		if modelEnd > modelStart {
			disk.Model = strings.Join(fields[modelStart:modelEnd], " ")
		}
	}

	// Determine RAID role and calculate utilization
	s.determineStoreCLIRaidRole(&disk, state, driveGroup, raidArrays)

	// Find matching array for utilization calculation
	var matchingArray *types.RAIDInfo
	for i, raid := range raidArrays {
		if raid.ArrayID == driveGroup {
			matchingArray = &raidArrays[i]
			break
		}
	}

	s.calculateStoreCLIDiskUtilization(&disk, matchingArray)

	return disk
}

// parseStoreCLIDetailedFormat parses the detailed per-drive format
func (s *StoreCLITool) parseStoreCLIDetailedFormat(lines []string, raidArrays []types.RAIDInfo) []types.DiskInfo {
	var disks []types.DiskInfo
	var currentDisk types.DiskInfo
	var currentArray *types.RAIDInfo
	var inDriveSection bool
	var inDetailedSection bool

	// Helper function to finalize and add current disk
	finalizeCurrentDisk := func() {
		if currentDisk.Device != "" {
			// Find the corresponding array for utilization calculation
			for i, raid := range raidArrays {
				if raid.ArrayID == currentDisk.RaidArrayID {
					currentArray = &raidArrays[i]
					break
				}
			}

			// Calculate utilization
			if currentArray != nil {
				s.calculateStoreCLIDiskUtilization(&currentDisk, currentArray)
			} else {
				s.calculateStoreCLIDiskUtilization(&currentDisk, nil)
			}

			// Add the completed disk
			disks = append(disks, currentDisk)
		}
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines
		if line == "" {
			continue
		}

		// Detect start of drive section
		if strings.Contains(line, "Drive /c") && strings.Contains(line, "/e") && strings.Contains(line, " :") {

			// Finalize previous disk if exists
			finalizeCurrentDisk()

			// Initialize new drive section
			inDriveSection = true
			inDetailedSection = false
			currentDisk = types.DiskInfo{
				Type:         "raid",
				Interface:    "SATA",
				SmartEnabled: true, // Assume SMART is available through RAID controller
				SmartHealthy: true, // Default to healthy unless proven otherwise
				Health:       "OK", // Default health status
			}
			currentArray = nil

			// Extract Controller, EID and Slot from header: "Drive /c0/e64/s3 :"
			re := regexp.MustCompile(`Drive /c(\d+)/e(\d+)/s(\d+)`)
			matches := re.FindStringSubmatch(line)
			if len(matches) >= 4 {
				controller := matches[1]
				enclosure := matches[2]
				slot := matches[3]
				currentDisk.Device = fmt.Sprintf("raid-c%s-enc%s-slot%s", controller, enclosure, slot)
				currentDisk.Location = fmt.Sprintf("Controller:%s EID:%s Slot:%s", controller, enclosure, slot)
			}
			continue
		}

		// Detect detailed information section
		if strings.Contains(line, "Detailed Information") {
			inDetailedSection = true
			continue
		}

		// Parse the main drive table line (the one with EID:Slt format)
		if inDriveSection && !inDetailedSection && strings.Contains(line, ":") &&
			(strings.Contains(line, "Onln") || strings.Contains(line, "GHS") ||
				strings.Contains(line, "DHS") || strings.Contains(line, "UGood") ||
				strings.Contains(line, "UBad") || strings.Contains(line, "Offln")) {

			fields := strings.Fields(line)
			if len(fields) >= 8 {
				// Parse: EID:Slt DID State DG Size Intf Med SED PI SeSz Model...
				// Example: 64:3 3 Onln 0 893.750 GB SATA SSD N N 512B SAMSUNG MZ7KM960HAHP-00005...

				state := fields[2]
				driveGroup := fields[3]
				currentDisk.Health = state

				// Determine RAID role and array assignment
				s.determineStoreCLIRaidRole(&currentDisk, state, driveGroup, raidArrays)

				// Parse capacity
				if len(fields) > 5 {
					sizeStr := fields[4] + " " + fields[5]
					currentDisk.Capacity = utils.ParseSizeToBytes(sizeStr)
				}

				// Parse interface and media type
				if len(fields) > 6 {
					currentDisk.Interface = fields[6]
				}

				// Parse model using field position logic
				// Format: EID:Slt DID State DG Size Unit Intf Med SED PI SeSz Model... Sp Type
				// Fields 0-10 are fixed, Model starts at field 11, ends 2 fields before the end
				if len(fields) >= 13 { // Minimum: 11 fixed fields + model + Sp + Type
					modelStart := 11
					modelEnd := len(fields) - 2 // Exclude Sp and Type at the end
					if modelEnd > modelStart {
						currentDisk.Model = strings.Join(fields[modelStart:modelEnd], " ")
					}
				}
			}
			continue
		}

		// Parse detailed drive information
		if inDetailedSection {
			if strings.Contains(line, "Drive Temperature =") {
				// Extract temperature: "Drive Temperature =  28C (82.40 F)"
				re := regexp.MustCompile(`Drive Temperature =\s*(\d+)C`)
				matches := re.FindStringSubmatch(line)
				if len(matches) > 1 {
					if temp, err := strconv.Atoi(matches[1]); err == nil {
						currentDisk.Temperature = float64(temp)
					}
				}
			} else if strings.Contains(line, "SN =") {
				// Extract serial number: "SN = S2HTNX0J400715"
				parts := strings.Split(line, "=")
				if len(parts) > 1 {
					currentDisk.Serial = strings.TrimSpace(parts[1])
				}
			} else if strings.Contains(line, "Model Number =") {
				// Extract model: "Model Number = SAMSUNG MZ7KM960HAHP-00005"
				parts := strings.Split(line, "=")
				if len(parts) > 1 {
					currentDisk.Model = strings.TrimSpace(parts[1])
				}
			} else if strings.Contains(line, "Drive position =") {
				// Extract RAID position: "Drive position = DriveGroup:0, Span:0, Row:0"
				parts := strings.Split(line, "=")
				if len(parts) > 1 {
					position := strings.TrimSpace(parts[1])
					currentDisk.RaidPosition = position

					// Extract DriveGroup for array mapping
					re := regexp.MustCompile(`DriveGroup:(\d+)`)
					matches := re.FindStringSubmatch(position)
					if len(matches) > 1 {
						currentDisk.RaidArrayID = matches[1]
					}
				}
			} else if strings.Contains(line, "Commissioned Spare =") {
				// Extract spare status: "Commissioned Spare = No"
				currentDisk.IsCommissionedSpare = strings.Contains(line, "= Yes")
				if currentDisk.IsCommissionedSpare {
					currentDisk.RaidRole = "commissioned_spare"
				}
			} else if strings.Contains(line, "Emergency Spare =") {
				// Extract emergency spare status: "Emergency Spare = No"
				currentDisk.IsEmergencySpare = strings.Contains(line, "= Yes")
				if currentDisk.IsEmergencySpare {
					currentDisk.RaidRole = "emergency_spare"
				}
			} else if strings.Contains(line, "S.M.A.R.T alert flagged by drive =") {
				// Extract SMART status: "S.M.A.R.T alert flagged by drive = No"
				currentDisk.SmartEnabled = true
				// If line contains "= No", it means no SMART alert, so disk is healthy
				if strings.Contains(line, "= No") {
					currentDisk.SmartHealthy = true
				} else if strings.Contains(line, "= Yes") {
					currentDisk.SmartHealthy = false
				}
			}
		}
	}

	// Don't forget to finalize the last disk
	finalizeCurrentDisk()

	return disks
}

// calculateBasicUtilization calculates basic utilization for disks without detailed role information
func (s *StoreCLITool) calculateBasicUtilization(disk *types.DiskInfo, raidArrays []types.RAIDInfo) {
	// Try to determine basic RAID role from health status
	healthLower := strings.ToLower(disk.Health)

	switch {
	case strings.Contains(healthLower, "onln") || strings.Contains(healthLower, "online"):
		disk.RaidRole = "active"
		// Try to find which array this disk belongs to (basic guess)
		if len(raidArrays) > 0 {
			disk.RaidArrayID = raidArrays[0].ArrayID // Default to first array
			s.calculateStoreCLIDiskUtilization(disk, &raidArrays[0])
		}
	case strings.Contains(healthLower, "spare") || strings.Contains(healthLower, "hot"):
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
	case strings.Contains(healthLower, "unconfigured") || strings.Contains(healthLower, "ugood"):
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

// determineStoreCLIRaidRole determines the RAID role based on StoreCLI state and drive group
func (s *StoreCLITool) determineStoreCLIRaidRole(disk *types.DiskInfo, state string, driveGroup string, raidArrays []types.RAIDInfo) {
	stateLower := strings.ToLower(state)

	// Set health status based on state - this is important for the final health assessment
	switch {
	case strings.Contains(stateLower, "onln") || strings.Contains(stateLower, "online"):
		disk.RaidRole = "active"
		disk.RaidArrayID = driveGroup
		disk.SmartHealthy = true // Online disks are healthy
		disk.SmartEnabled = true // Assume SMART available through RAID controller
		disk.Health = "OK"       // Override the raw state with a cleaner status

		// Find and set the specific array
		for _, raid := range raidArrays {
			if raid.ArrayID == driveGroup {
				break
			}
		}

	case strings.Contains(stateLower, "ghs") || strings.Contains(stateLower, "global hot spare"):
		disk.RaidRole = "hot_spare"
		disk.IsGlobalSpare = true
		disk.SmartHealthy = true // Spare disks are typically healthy
		disk.SmartEnabled = true

	case strings.Contains(stateLower, "dhs") || strings.Contains(stateLower, "dedicated hot spare"):
		disk.RaidRole = "hot_spare"
		disk.IsDedicatedSpare = true
		disk.RaidArrayID = driveGroup
		disk.SmartHealthy = true // Spare disks are typically healthy
		disk.SmartEnabled = true

	case strings.Contains(stateLower, "spare"):
		disk.RaidRole = "spare"
		disk.SmartHealthy = true // Spare disks are typically healthy
		disk.SmartEnabled = true

	case strings.Contains(stateLower, "failed") || strings.Contains(stateLower, "fail"):
		disk.RaidRole = "failed"
		disk.SmartHealthy = false
		disk.Health = "FAILED"

	case strings.Contains(stateLower, "rebuild") || strings.Contains(stateLower, "rbld"):
		disk.RaidRole = "rebuilding"
		disk.RaidArrayID = driveGroup
		disk.SmartHealthy = true // Rebuilding disks are still functional
		disk.SmartEnabled = true

	case strings.Contains(stateLower, "ugood") || strings.Contains(stateLower, "unconfigured good"):
		disk.RaidRole = "unconfigured"
		disk.SmartHealthy = true // Good unconfigured disks are healthy
		disk.SmartEnabled = true

	case strings.Contains(stateLower, "ubad") || strings.Contains(stateLower, "unconfigured bad"):
		disk.RaidRole = "unconfigured"
		disk.Health = "FAILED"
		disk.SmartHealthy = false

	case strings.Contains(stateLower, "offln") || strings.Contains(stateLower, "offline"):
		disk.RaidRole = "failed"
		disk.SmartHealthy = false
		disk.Health = "FAILED"

	default:
		disk.RaidRole = "unknown"
		// For unknown states, be conservative but check if it looks healthy
		if !strings.Contains(stateLower, "fail") && !strings.Contains(stateLower, "bad") && !strings.Contains(stateLower, "error") {
			disk.SmartHealthy = true
			disk.SmartEnabled = true

		}
	}
}

// calculateStoreCLIDiskUtilization calculates disk utilization for StoreCLI managed disks
func (s *StoreCLITool) calculateStoreCLIDiskUtilization(disk *types.DiskInfo, array *types.RAIDInfo) {
	if disk.Capacity <= 0 {
		return
	}

	// Handle spare drives differently - they are not actively used in arrays
	if disk.RaidRole == "hot_spare" || disk.RaidRole == "spare" ||
		disk.IsGlobalSpare || disk.IsDedicatedSpare {
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
	if array == nil || array.Size <= 0 {
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
	case strings.Contains(raidLevel, "raid0") || strings.Contains(raidLevel, "r0"):
		// RAID 0: All disk space is used for data
		usableCapacityPerDisk = array.Size / int64(numDrives)
		utilizationPercentage = 100.0

	case strings.Contains(raidLevel, "raid1") || strings.Contains(raidLevel, "r1"):
		// RAID 1: 50% of disk space is used (mirrored)
		usableCapacityPerDisk = disk.Capacity / 2
		utilizationPercentage = 50.0

	case strings.Contains(raidLevel, "raid5") || strings.Contains(raidLevel, "r5"):
		// RAID 5: (n-1)/n of disk space is used for data, 1/n for parity
		usableCapacityPerDisk = array.Size / int64(numDrives-1) * int64(numDrives) / int64(numDrives)
		utilizationPercentage = float64(numDrives-1) / float64(numDrives) * 100.0

	case strings.Contains(raidLevel, "raid6") || strings.Contains(raidLevel, "r6"):
		// RAID 6: (n-2)/n of disk space is used for data, 2/n for parity
		usableCapacityPerDisk = array.Size / int64(numDrives-2) * int64(numDrives) / int64(numDrives)
		utilizationPercentage = float64(numDrives-2) / float64(numDrives) * 100.0

	case strings.Contains(raidLevel, "raid10") || strings.Contains(raidLevel, "r10"):
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
func (s *StoreCLITool) GetSpareDisks() []types.DiskInfo {
	allDisks := s.GetRAIDDisks()
	var spareDisks []types.DiskInfo

	for _, disk := range allDisks {
		if disk.RaidRole == "hot_spare" || disk.RaidRole == "spare" ||
			disk.IsGlobalSpare || disk.IsDedicatedSpare {
			spareDisks = append(spareDisks, disk)
		}
	}

	return spareDisks
}

// GetUnconfiguredDisks returns information about unconfigured drives
func (s *StoreCLITool) GetUnconfiguredDisks() []types.DiskInfo {
	allDisks := s.GetRAIDDisks()
	var unconfiguredDisks []types.DiskInfo

	for _, disk := range allDisks {
		if disk.RaidRole == "unconfigured" {
			unconfiguredDisks = append(unconfiguredDisks, disk)
		}
	}

	return unconfiguredDisks
}
