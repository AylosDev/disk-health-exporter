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
// megacli -LDInfo -Lall -aALL -NoLog # get logical drive information for all arrays
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
			if raidLevelStr, ok := parseKeyValue(line, "RAID Level"); ok {
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
			if sizeStr, ok := parseKeyValue(line, "Size"); ok {
				// Convert human-readable size to bytes if possible, otherwise store as 0
				currentArray.Size = utils.ParseSizeToBytes(sizeStr)
			}
		} else if strings.Contains(line, "Number Of Drives") {
			// Extract number of drives
			if numDrivesStr, ok := parseKeyValue(line, "Number Of Drives"); ok {
				if num, err := strconv.Atoi(numDrivesStr); err == nil && num > 0 {
					currentArray.NumDrives = num
				}
			}
		} else if strings.Contains(line, "State") {
			// Extract state
			if state, ok := parseKeyValue(line, "State"); ok {
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

	spareDisks := m.getUnassignedPhysicalDisks()
	numSpareDrives := 0
	numFailedDrives := 0
	numUnconfiguredDrives := 0

	for _, disk := range spareDisks {
		switch disk.RaidRole {
		case "hot_spare", "spare":
			numSpareDrives++
		case "failed":
			numFailedDrives++
		case "unconfigured":
			numUnconfiguredDrives++
		}
	}

	for i := range raidArrays {
		raidArrays[i].NumSpareDrives = numSpareDrives
		raidArrays[i].NumFailedDrives += numFailedDrives // Add to existing count from array parsing

		if raidArrays[i].NumActiveDrives == 0 && raidArrays[i].NumDrives > 0 {
			raidArrays[i].NumActiveDrives = raidArrays[i].NumDrives
		}
	}

	// If we have spare drives but no arrays, create a virtual array entry for the spares
	// This ensures spare drives are always reported in metrics
	if len(raidArrays) == 0 && numSpareDrives > 0 {
		virtualArray := types.RAIDInfo{
			ArrayID:         "spare",
			RaidLevel:       "spare-only",
			State:           "spare",
			Status:          1, // OK status
			Size:            0,
			NumDrives:       numSpareDrives,
			NumActiveDrives: 0,
			NumSpareDrives:  numSpareDrives,
			NumFailedDrives: numFailedDrives,
			Type:            "hardware",
			Controller:      "MegaCLI",
		}
		raidArrays = append(raidArrays, virtualArray)
	}

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

	// Get detailed physical disk information with RAID array mapping
	disks = m.getAllPhysicalDisksForArrays(raidArrays)

	return disks
}

// getAllPhysicalDisksForArrays gets physical disks for all arrays in one optimized pass
// megacli -LdPdInfo -aALL -NoLog # get logical drive and physical drive information for all arrays
func (m *MegaCLITool) getAllPhysicalDisksForArrays(raidArrays []types.RAIDInfo) []types.DiskInfo {
	var disks []types.DiskInfo
	processedDisks := make(map[string]bool) // Track processed disks to avoid duplicates

	// Get the LdPdInfo output once for all arrays (more efficient)
	ldPdOutput, err := exec.Command(m.command, "-LdPdInfo", "-aALL", "-NoLog").Output()
	if err != nil {
		log.Printf("MegaCLI LdPdInfo command failed: %v", err)
		return disks
	}

	// Create a map of target array IDs for quick lookup
	targetArrays := make(map[string]bool)
	for _, raidArray := range raidArrays {
		targetArrays[raidArray.ArrayID] = true
	}

	// Parse the output once for all arrays
	arrayDisks := m.parseLdPdInfoOutputForAllArrays(string(ldPdOutput), targetArrays)
	for _, disk := range arrayDisks {
		diskKey := generateDiskKey(disk)
		if !processedDisks[diskKey] {
			disks = append(disks, disk)
			processedDisks[diskKey] = true
		}
	}

	// Also get unassigned physical disks (hot spares, unconfigured, etc.)
	unassignedDisks := m.getUnassignedPhysicalDisks()
	for _, disk := range unassignedDisks {
		diskKey := generateDiskKey(disk)
		if !processedDisks[diskKey] {
			disks = append(disks, disk)
			processedDisks[diskKey] = true
		}
	}

	return disks
}

// parseLdPdInfoOutput parses the output of MegaCli -LdPdInfo -aALL -NoLog

// finalizeLdPdInfoDisk finalizes disk information from LdPdInfo parsing
func (m *MegaCLITool) finalizeLdPdInfoDisk(disk *types.DiskInfo, arrayID string, enclosure string, slot string) {
	// Set location if we have enclosure and slot
	if enclosure != "" && slot != "" {
		disk.Location = fmt.Sprintf("Enc:%s Slot:%s", enclosure, slot)
	}

	// Set RAID information
	disk.RaidArrayID = arrayID
	disk.Type = "raid"

	// Determine role based on health status
	disk.RaidRole = determineRaidRole(disk.Health)

	// Set global spare flag for hot spares
	if disk.RaidRole == "hot_spare" {
		disk.IsGlobalSpare = strings.Contains(strings.ToLower(disk.Health), "global")
	}
}

// getUnassignedPhysicalDisks gets physical disks that are not assigned to any array (hot spares, unconfigured, etc.)
// megacli -PDList -aALL -NoLog # list all physical disks from all adapters
func (m *MegaCLITool) getUnassignedPhysicalDisks() []types.DiskInfo {
	var disks []types.DiskInfo

	// Get all physical disks
	output, err := exec.Command(m.command, "-PDList", "-aALL", "-NoLog").Output()
	if err != nil {
		log.Printf("Error executing MegaCLI for unassigned disk info: %v", err)
		return disks
	}

	lines := strings.Split(string(output), "\n")
	var currentDisk types.DiskInfo
	var enclosure, slot string

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Parse disk information line by line
		m.parsePhysicalDiskLine(line, &currentDisk, &enclosure, &slot)

		// Check if this is the end of a disk section and if disk is unassigned
		if strings.Contains(line, "Firmware state:") && currentDisk.Device != "" {
			// Check if this disk is not part of an active array (hot spare, unconfigured, etc.)
			state := strings.ToLower(currentDisk.Health)
			if strings.Contains(state, "hotspare") || strings.Contains(state, "spare") ||
				strings.Contains(state, "unconfigured") || strings.Contains(state, "jbod") ||
				strings.Contains(state, "failed") {
				m.finalizeUnassignedDisk(&currentDisk)
				disks = append(disks, currentDisk)
			}
			currentDisk = types.DiskInfo{} // Reset for next disk
			enclosure = ""
			slot = ""
		}
	}

	return disks
}

// parsePhysicalDiskLine parses a single line of physical disk information
func (m *MegaCLITool) parsePhysicalDiskLine(line string, currentDisk *types.DiskInfo, enclosure *string, slot *string) {
	if value, ok := parseKeyValue(line, "Enclosure Device ID"); ok {
		*enclosure = value
	} else if value, ok := parseKeyValue(line, "Slot Number"); ok {
		*slot = value
		// Set location when we have both enclosure and slot
		if *enclosure != "" && *slot != "" {
			currentDisk.Location = fmt.Sprintf("Enc:%s Slot:%s", *enclosure, *slot)
		}
	} else if value, ok := parseKeyValue(line, "Device Id"); ok {
		currentDisk.Device = value
	} else if value, ok := parseKeyValue(line, "Coerced Size"); ok {
		sizeStr := cleanSizeString(value)
		currentDisk.Capacity = utils.ParseSizeToBytes(sizeStr)
	} else if value, ok := parseKeyValue(line, "Inquiry Data"); ok {
		// Extract model from inquiry data
		currentDisk.Model = m.extractModelFromInquiry(value)
	} else if value, ok := parseKeyValue(line, "WWN"); ok {
		currentDisk.Serial = value
	} else if value, ok := parseKeyValue(line, "Drive Temperature"); ok {
		if temp, tempOk := parseTemperature(value); tempOk {
			currentDisk.Temperature = temp
		}
	} else if strings.Contains(line, "Hotspare Information:") {
		// This indicates the disk is a hot spare
		currentDisk.RaidRole = "hot_spare"
		currentDisk.IsGlobalSpare = true
	} else if strings.Contains(line, "Type:") && currentDisk.RaidRole == "hot_spare" {
		// Parse hotspare type information
		if value, ok := parseKeyValue(line, "Type"); ok {
			if strings.Contains(strings.ToLower(value), "global") {
				currentDisk.IsGlobalSpare = true
			}
		}
	} else if value, ok := parseKeyValue(line, "Firmware state"); ok {
		currentDisk.Health = value
		currentDisk.Type = "raid"
	}
}

// finalizeRAIDDisk finalizes a RAID disk with array-specific information
func (m *MegaCLITool) finalizeRAIDDisk(disk *types.DiskInfo, raidArray types.RAIDInfo) {
	disk.RaidArrayID = raidArray.ArrayID
	disk.Type = "raid"

	// Determine role based on health status
	healthLower := strings.ToLower(disk.Health)
	switch {
	case strings.Contains(healthLower, "online"):
		disk.RaidRole = "active"
		m.calculateDiskUtilization(disk, &raidArray)
	case strings.Contains(healthLower, "rebuilding"):
		disk.RaidRole = "rebuilding"
		m.calculateDiskUtilization(disk, &raidArray)
	case strings.Contains(healthLower, "failed") || strings.Contains(healthLower, "fail"):
		disk.RaidRole = "failed"
		disk.UsedBytes = 0
		disk.AvailableBytes = 0
		disk.UsagePercentage = 0.0
		disk.Mountpoint = "FAILED"
		disk.Filesystem = "Failed-Drive"
	default:
		disk.RaidRole = "unknown"
		m.calculateDiskUtilization(disk, &raidArray)
	}
}

// finalizeUnassignedDisk finalizes an unassigned disk (hot spare, unconfigured, etc.)
func (m *MegaCLITool) finalizeUnassignedDisk(disk *types.DiskInfo) {
	disk.Type = "raid"

	// Determine role based on health status
	disk.RaidRole = determineRaidRole(disk.Health)

	// Set utilization based on role
	switch disk.RaidRole {
	case "hot_spare":
		disk.IsGlobalSpare = true
		setDiskUtilization(disk, 0, disk.Capacity, 0.0, "SPARE", "Hot-Spare")
	case "unconfigured":
		setDiskUtilization(disk, 0, disk.Capacity, 0.0, "UNCONFIGURED", "Unconfigured")
	case "failed":
		setDiskUtilization(disk, 0, 0, 0.0, "FAILED", "Failed-Drive")
	default:
		// Handle JBOD case specifically
		healthLower := strings.ToLower(disk.Health)
		if strings.Contains(healthLower, "jbod") {
			disk.RaidRole = "unconfigured"
			setDiskUtilization(disk, 0, disk.Capacity, 0.0, "JBOD", "JBOD")
		} else {
			// Unknown state
			setDiskUtilization(disk, 0, 0, 0.0, "UNKNOWN", "Unknown")
		}
	}
}

// extractModelFromInquiry extracts the disk model from MegaCLI inquiry data in a vendor-agnostic way
func (m *MegaCLITool) extractModelFromInquiry(inquiry string) string {
	// Parse the first meaningful field that looks like a model number
	fields := strings.Fields(inquiry)
	if len(fields) == 0 {
		return ""
	}

	// Strategy: Find the field that contains the actual model number
	// Usually it's either the first field or the field that contains alphanumeric model pattern
	var modelCandidate string

	// Check if first field contains a dash (vendor-model format)
	if strings.Contains(fields[0], "-") {
		// Split by dash and take the second part (model part), but only until next dash
		dashParts := strings.Split(fields[0], "-")
		if len(dashParts) >= 2 {
			// Take the second part, which is typically the model
			modelCandidate = dashParts[1]
		} else {
			modelCandidate = fields[0]
		}
	} else if len(fields) >= 2 {
		// If we have multiple fields, check if second field looks like a model
		// Model numbers usually contain alphanumeric characters and are substantial
		if len(fields[1]) >= 3 && (strings.ContainsAny(fields[1], "0123456789") || len(fields[1]) > 5) {
			modelCandidate = fields[1]
		} else {
			modelCandidate = fields[0]
		}
	} else {
		// Single field, use it as is
		modelCandidate = fields[0]
	}

	// Clean up the model candidate (remove extra characters if needed)
	return strings.TrimSpace(modelCandidate)
}

// GetBatteryInfo returns battery information for a specific adapter
// megacli -AdpBbuCmd -aX # get battery backup unit information for adapter X
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

		if value, ok := parseKeyValue(line, "BatteryType"); ok {
			batteryInfo.BatteryType = value
		} else if strings.Contains(line, "Voltage:") && !strings.Contains(line, "Design Voltage") {
			if voltageStr, ok := parseKeyValue(line, "Voltage"); ok {
				// Extract voltage value (e.g., "9481 mV" -> 9481)
				if voltage, voltageOk := extractNumericValue(voltageStr, "mV"); voltageOk {
					batteryInfo.Voltage = voltage
				}
			}
		} else if value, ok := parseKeyValue(line, "Current"); ok {
			// Extract current value (e.g., "0 mA" -> 0)
			if current, currentOk := extractNumericValue(value, "mA"); currentOk {
				batteryInfo.Current = current
			}
		} else if strings.Contains(line, "Temperature:") && !strings.Contains(line, "Temperature                             :") {
			if tempStr, ok := parseKeyValue(line, "Temperature"); ok {
				// Extract temperature value (e.g., "35 C" -> 35)
				if temp, tempOk := extractNumericValue(tempStr, "C"); tempOk {
					batteryInfo.Temperature = temp
				}
			}
		} else if value, ok := parseKeyValue(line, "Battery State"); ok {
			batteryInfo.State = value
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

	// Handle nil array (fallback case) - don't make estimations, leave values as zero
	if array == nil {
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
		return
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

	return unconfiguredDisks
}

// parseLdPdInfoOutputForAllArrays parses the output of MegaCli -LdPdInfo -aALL -NoLog for all target arrays
// (This function processes output from: megacli -LdPdInfo -aALL -NoLog # get detailed logical and physical drive info)
func (m *MegaCLITool) parseLdPdInfoOutputForAllArrays(output string, targetArrays map[string]bool) []types.DiskInfo {
	var disks []types.DiskInfo
	lines := strings.Split(output, "\n")
	processedDisks := make(map[string]bool) // Track processed disks to avoid duplicates

	var currentLogicalDrive string
	var inTargetArray bool
	var currentDisk types.DiskInfo
	var enclosure, slot string
	inPhysicalDiskSection := false

	for i, line := range lines {
		line = strings.TrimSpace(line)

		// Detect logical drive sections
		if strings.HasPrefix(line, "Virtual Drive:") || strings.Contains(line, "Virtual Drive") {
			// Reset state for new logical drive
			inTargetArray = false
			inPhysicalDiskSection = false

			// Extract logical drive number/ID
			if strings.Contains(line, "Virtual Drive:") {
				if driveInfo, ok := parseKeyValue(line, "Virtual Drive"); ok {
					// Extract just the number (e.g., "0 (Target Id: 0)" -> "0")
					if idx := strings.Index(driveInfo, " "); idx != -1 {
						currentLogicalDrive = driveInfo[:idx]
					} else {
						currentLogicalDrive = driveInfo
					}

					// Check if this is one of our target arrays
					inTargetArray = targetArrays[currentLogicalDrive]
				}
			}
			continue
		}

		// Skip lines if we're not in a target array
		if !inTargetArray {
			continue
		}

		// Detect physical disk sections within the target array
		if strings.Contains(line, "PD:") && (strings.Contains(line, "Information") || strings.Contains(line, "Info")) {
			inPhysicalDiskSection = true
			// Reset disk info for new physical disk
			currentDisk = types.DiskInfo{}
			enclosure = ""
			slot = ""
			continue
		}

		if !inPhysicalDiskSection {
			continue
		}

		// Parse disk information
		if value, ok := parseKeyValue(line, "Enclosure Device ID"); ok {
			enclosure = value
		} else if value, ok := parseKeyValue(line, "Slot Number"); ok {
			slot = value
		} else if value, ok := parseKeyValue(line, "Device Id"); ok {
			currentDisk.Device = value
		} else if value, ok := parseKeyValue(line, "Inquiry Data"); ok {
			currentDisk.Model = m.extractModelFromInquiry(value)
		} else if value, ok := parseKeyValue(line, "Firmware state"); ok {
			currentDisk.Health = value
		} else if value, ok := parseKeyValue(line, "Coerced Size"); ok {
			sizeStr := cleanSizeString(value)
			currentDisk.Capacity = utils.ParseSizeToBytes(sizeStr)
		} else if value, ok := parseKeyValue(line, "WWN"); ok {
			currentDisk.Serial = value
		} else if value, ok := parseKeyValue(line, "Drive Temperature"); ok {
			if temp, tempOk := parseTemperature(value); tempOk {
				currentDisk.Temperature = temp
			}
		}

		// Check if we've reached the end of a physical disk section
		isEndOfDisk := false

		// Look at the next non-empty line to see if it's a new PD section
		if i+1 < len(lines) {
			nextLine := strings.TrimSpace(lines[i+1])
			if nextLine != "" && strings.Contains(nextLine, "PD:") && strings.Contains(nextLine, "Information") {
				isEndOfDisk = true
			}
		} else if i == len(lines)-1 {
			// End of file
			isEndOfDisk = true
		}

		if isEndOfDisk && inPhysicalDiskSection {
			// Finalize disk information
			if enclosure != "" && slot != "" && currentDisk.Device != "" {
				// Use the existing finalization method to properly set all disk properties
				m.finalizeLdPdInfoDisk(&currentDisk, currentLogicalDrive, enclosure, slot)

				// Only add if we haven't processed this disk yet
				diskKey := generateDiskKey(currentDisk)
				if !processedDisks[diskKey] {
					disks = append(disks, currentDisk)
					processedDisks[diskKey] = true
				}
			}

			inPhysicalDiskSection = false
		}
	}

	return disks
}

// Utility functions to reduce code duplication

// parseKeyValue extracts value from "key: value" formatted lines using regex
func parseKeyValue(line, key string) (string, bool) {
	// Use regex to match "key" followed by optional spaces, colon, and capture the value
	pattern := fmt.Sprintf(`%s\s*:\s*(.+)`, regexp.QuoteMeta(key))
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(line)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1]), true
	}
	return "", false
}

// determineRaidRole determines RAID role based on health status
func determineRaidRole(health string) string {

	healthLower := strings.ToLower(health)
	switch {
	case strings.Contains(healthLower, "hotspare") || strings.Contains(healthLower, "spare"):
		return "hot_spare"
	case strings.Contains(healthLower, "online") || strings.Contains(healthLower, "optimal"):
		return "active"
	case strings.Contains(healthLower, "rebuild"):
		return "rebuilding"
	case strings.Contains(healthLower, "failed") || strings.Contains(healthLower, "offline"):
		return "failed"
	case strings.Contains(healthLower, "unconfigured"):
		return "unconfigured"
	default:
		return "unknown"
	}
}

// setDiskUtilization sets disk utilization values
func setDiskUtilization(disk *types.DiskInfo, usedBytes, availableBytes int64, usagePercentage float64, mountpoint, filesystem string) {
	disk.UsedBytes = usedBytes
	disk.AvailableBytes = availableBytes
	disk.UsagePercentage = usagePercentage
	disk.Mountpoint = mountpoint
	disk.Filesystem = filesystem
}

// parseTemperature extracts temperature value from strings like "35 C" or "35C"
func parseTemperature(tempStr string) (float64, bool) {
	re := regexp.MustCompile(`(\d+)C`)
	matches := re.FindStringSubmatch(tempStr)
	if len(matches) > 1 {
		if temp, err := strconv.ParseFloat(matches[1], 64); err == nil {
			return temp, true
		}
	}
	return 0, false
}

// extractNumericValue extracts numeric values with units (e.g., "9481 mV" -> 9481)
func extractNumericValue(text, unit string) (int, bool) {
	pattern := fmt.Sprintf(`(\d+)\s*%s`, unit)
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(text)
	if len(matches) > 1 {
		if value, err := strconv.Atoi(matches[1]); err == nil {
			return value, true
		}
	}
	return 0, false
}

// generateDiskKey creates a unique disk identifier
func generateDiskKey(disk types.DiskInfo) string {
	return fmt.Sprintf("%s-%s", disk.Location, disk.Device)
}

// cleanSizeString removes bracket information from size strings
func cleanSizeString(sizeStr string) string {
	if bracketIndex := strings.Index(sizeStr, "["); bracketIndex != -1 {
		return strings.TrimSpace(sizeStr[:bracketIndex])
	}
	return sizeStr
}
