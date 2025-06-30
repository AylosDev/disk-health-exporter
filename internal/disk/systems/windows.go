package systems

import (
	"log"
	"strings"

	"disk-health-exporter/internal/utils"
	"disk-health-exporter/pkg/types"
)

// WindowsSystem represents the Windows-specific disk detection implementation
type WindowsSystem struct {
	targetDisks    []string
	ignorePatterns []string
	toolsAvailable struct {
		smartctl bool
		nvme     bool
	}
}

// NewWindowsSystem creates a new WindowsSystem instance
func NewWindowsSystem(targetDisks []string, ignorePatterns []string) *WindowsSystem {
	w := &WindowsSystem{
		targetDisks:    targetDisks,
		ignorePatterns: ignorePatterns,
	}

	// Check tool availability once at startup
	w.toolsAvailable.smartctl = utils.CommandExists("smartctl")
	w.toolsAvailable.nvme = utils.CommandExists("nvme")

	log.Printf("Windows tool availability detected: smartctl=%v, nvme=%v",
		w.toolsAvailable.smartctl, w.toolsAvailable.nvme)

	return w
}

// GetDisks gets all disks on Windows systems using available tools
func (w *WindowsSystem) GetDisks() ([]types.DiskInfo, []types.RAIDInfo) {
	var allDisks []types.DiskInfo
	var allRAIDs []types.RAIDInfo

	log.Println("Detecting disks on Windows system...")

	// Use WMI or PowerShell for Windows disk detection
	disks := w.getWindowsDisks()
	filtered := w.filterDisks(disks)
	allDisks = append(allDisks, filtered...)
	log.Printf("Found %d disks via Windows APIs", len(filtered))

	// Enhance with SMART data if available
	if w.toolsAvailable.smartctl {
		for i, disk := range allDisks {
			smartInfo := w.getSmartInfo(disk.Device)
			if smartInfo.Device != "" {
				allDisks[i] = w.mergeDiskInfo(disk, smartInfo)
			}
		}
	}

	log.Printf("Total: %d disks, %d RAID arrays", len(allDisks), len(allRAIDs))
	return allDisks, allRAIDs
}

// getWindowsDisks gets disk information using Windows APIs
func (w *WindowsSystem) getWindowsDisks() []types.DiskInfo {
	var disks []types.DiskInfo

	// TODO: Implement Windows disk detection using WMI or PowerShell
	log.Println("Windows disk detection - implementation needed")

	return disks
}

// getSmartInfo gets SMART information for a disk
func (w *WindowsSystem) getSmartInfo(device string) types.DiskInfo {
	// TODO: Implement SMART info retrieval for Windows
	return types.DiskInfo{}
}

// mergeDiskInfo merges disk information from different sources
func (w *WindowsSystem) mergeDiskInfo(existing, smart types.DiskInfo) types.DiskInfo {
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
func (w *WindowsSystem) filterDisks(disks []types.DiskInfo) []types.DiskInfo {
	var filtered []types.DiskInfo
	for _, disk := range disks {
		if w.shouldIncludeDisk(disk.Device) {
			filtered = append(filtered, disk)
		}
	}
	return filtered
}

// shouldIncludeDisk checks if a disk should be included based on configuration
func (w *WindowsSystem) shouldIncludeDisk(device string) bool {
	// First check ignore patterns
	for _, pattern := range w.ignorePatterns {
		if strings.HasPrefix(device, pattern) {
			log.Printf("Ignoring disk %s (matches ignore pattern: %s)", device, pattern)
			return false
		}
	}

	// If target disks are specified, only include those
	if len(w.targetDisks) > 0 {
		for _, target := range w.targetDisks {
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
func (w *WindowsSystem) GetSystemType() string {
	return "windows"
}

// GetToolInfo reports which tools are available on this Windows system
func (w *WindowsSystem) GetToolInfo() types.ToolInfo {
	var toolInfo types.ToolInfo

	// Use cached tool availability
	toolInfo.SmartCtl = w.toolsAvailable.smartctl
	toolInfo.Nvme = w.toolsAvailable.nvme

	// Get versions for available tools
	if toolInfo.SmartCtl {
		if version, err := utils.GetToolVersion("smartctl", "--version"); err == nil {
			toolInfo.SmartCtlVersion = version
		}
	}

	return toolInfo
}
