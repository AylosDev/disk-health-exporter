package tools

import (
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"regexp"
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

// GetRAIDDisks returns disk information from RAID arrays
func (s *StoreCLITool) GetRAIDDisks() []types.DiskInfo {
	var disks []types.DiskInfo

	if !s.IsAvailable() {
		return disks
	}

	// Get physical disk information
	output, err := exec.Command(s.command, "/call", "/eall", "/sall", "show", "J").Output()
	if err != nil {
		log.Printf("Error executing StoreCLI for disk info: %v", err)
		return s.getRAIDDisksPlainText()
	}

	// Parse JSON output for disks
	disks = s.parseStoreCLIDisksJSON(output)
	if len(disks) > 0 {
		return disks
	}

	return s.getRAIDDisksPlainText()
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

	for _, controller := range controllers {
		ctrlMap, ok := controller.(map[string]interface{})
		if !ok {
			continue
		}

		// Extract controller information
		var controllerName string
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

	for _, controller := range controllers {
		ctrlMap, ok := controller.(map[string]interface{})
		if !ok {
			continue
		}

		if responseData, exists := ctrlMap["Response Data"]; exists {
			if dataMap, ok := responseData.(map[string]interface{}); ok {
				// Look for Drive Information
				if driveInfo, exists := dataMap["Drive Information"]; exists {
					if driveArray, ok := driveInfo.([]interface{}); ok {
						for _, drive := range driveArray {
							if driveMap, ok := drive.(map[string]interface{}); ok {
								disk := s.parsePhysicalDrive(driveMap)
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
func (s *StoreCLITool) parsePhysicalDrive(driveMap map[string]interface{}) types.DiskInfo {
	disk := types.DiskInfo{
		Type: "raid",
	}

	if eid, exists := driveMap["EID:Slt"]; exists {
		disk.Location = fmt.Sprintf("EID:Slt %v", eid)
		// Convert "252:0" to "raid-enc252-slot0" for better readability
		eidStr := fmt.Sprintf("%v", eid)
		parts := strings.Split(eidStr, ":")
		if len(parts) == 2 {
			disk.Device = fmt.Sprintf("raid-enc%s-slot%s", parts[0], parts[1])
		} else {
			disk.Device = fmt.Sprintf("raid-drive-%v", eid)
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
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Look for drive entries in the format: EID:Slt DID State DG Size Intf Med SED PI SeSz Model
		if driveRegex.MatchString(line) {
			fields := strings.Fields(line)
			if len(fields) >= 6 {
				// Convert "252:0" to "raid-enc252-slot0" for consistency
				eidSlot := fields[0]
				parts := strings.Split(eidSlot, ":")
				var deviceName string
				if len(parts) == 2 {
					deviceName = fmt.Sprintf("raid-enc%s-slot%s", parts[0], parts[1])
				} else {
					deviceName = fmt.Sprintf("raid-drive-%s", eidSlot)
				}

				disk := types.DiskInfo{
					Device:   deviceName,
					Location: fmt.Sprintf("EID:Slt %s", eidSlot),
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
