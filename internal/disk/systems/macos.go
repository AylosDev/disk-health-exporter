package systems

import (
	"log"
	"strings"

	"disk-health-exporter/internal/utils"
	"disk-health-exporter/pkg/types"
)

// MacOSSystem represents the macOS-specific disk detection implementation
type MacOSSystem struct {
	targetDisks    []string
	ignorePatterns []string
}

// NewMacOSSystem creates a new MacOSSystem instance
func NewMacOSSystem(targetDisks []string, ignorePatterns []string) *MacOSSystem {
	return &MacOSSystem{
		targetDisks:    targetDisks,
		ignorePatterns: ignorePatterns,
	}
}

// GetDisks gets all disks on macOS systems using available tools
func (m *MacOSSystem) GetDisks() ([]types.DiskInfo, []types.RAIDInfo) {
	var allDisks []types.DiskInfo
	var allRAIDs []types.RAIDInfo

	log.Println("Detecting disks on macOS system...")

	// Use diskutil for macOS disk detection
	if utils.CommandExists("diskutil") {
		disks := m.getDiskutilDisks()
		filtered := m.filterDisks(disks)
		allDisks = append(allDisks, filtered...)
		log.Printf("Found %d disks via diskutil", len(filtered))
	}

	// Enhance with SMART data if available
	if utils.CommandExists("smartctl") {
		for i, disk := range allDisks {
			smartInfo := m.getSmartInfo(disk.Device)
			if smartInfo.Device != "" {
				allDisks[i] = m.mergeDiskInfo(disk, smartInfo)
			}
		}
	}

	log.Printf("Total: %d disks, %d RAID arrays", len(allDisks), len(allRAIDs))
	return allDisks, allRAIDs
}

// getDiskutilDisks gets disk information using diskutil
func (m *MacOSSystem) getDiskutilDisks() []types.DiskInfo {
	var disks []types.DiskInfo

	// TODO: Implement diskutil parsing
	log.Println("diskutil disk detection - implementation needed")

	return disks
}

// getSmartInfo gets SMART information for a disk
func (m *MacOSSystem) getSmartInfo(device string) types.DiskInfo {
	// TODO: Implement SMART info retrieval for macOS
	return types.DiskInfo{}
}

// mergeDiskInfo merges disk information from different sources
func (m *MacOSSystem) mergeDiskInfo(existing, smart types.DiskInfo) types.DiskInfo {
	merged := existing
	if smart.Model != "" {
		merged.Model = smart.Model
	}
	if smart.Serial != "" {
		merged.Serial = smart.Serial
	}
	if smart.Health != "" {
		merged.Health = smart.Health
	}
	if smart.Temperature > 0 {
		merged.Temperature = smart.Temperature
	}
	if smart.Capacity > 0 {
		merged.Capacity = smart.Capacity
	}
	return merged
}

// filterDisks filters disks based on target and ignore patterns
func (m *MacOSSystem) filterDisks(disks []types.DiskInfo) []types.DiskInfo {
	var filtered []types.DiskInfo
	for _, disk := range disks {
		if m.shouldIncludeDisk(disk.Device) {
			filtered = append(filtered, disk)
		}
	}
	return filtered
}

// shouldIncludeDisk checks if a disk should be included based on configuration
func (m *MacOSSystem) shouldIncludeDisk(device string) bool {
	// First check ignore patterns
	for _, pattern := range m.ignorePatterns {
		if strings.HasPrefix(device, pattern) {
			log.Printf("Ignoring disk %s (matches ignore pattern: %s)", device, pattern)
			return false
		}
	}

	// If target disks are specified, only include those
	if len(m.targetDisks) > 0 {
		for _, target := range m.targetDisks {
			if device == target {
				log.Printf("Including target disk: %s", device)
				return true
			}
		}
		log.Printf("Skipping disk %s (not in target list)", device)
		return false
	}

	// No specific targets, include if not ignored
	return true
}

// GetSystemType returns the system type
func (m *MacOSSystem) GetSystemType() string {
	return "macOS"
}

// GetToolInfo reports which tools are available on this macOS system
func (m *MacOSSystem) GetToolInfo() types.ToolInfo {
	var toolInfo types.ToolInfo

	// Check tool availability - macOS specific tools
	toolInfo.Diskutil = utils.CommandExists("diskutil")
	toolInfo.SmartCtl = utils.CommandExists("smartctl")
	toolInfo.Nvme = utils.CommandExists("nvme")
	toolInfo.Zpool = utils.CommandExists("zpool")

	// Get versions for available tools
	if toolInfo.SmartCtl {
		if version, err := utils.GetToolVersion("smartctl", "--version"); err == nil {
			toolInfo.SmartCtlVersion = version
		}
	}

	return toolInfo
}
