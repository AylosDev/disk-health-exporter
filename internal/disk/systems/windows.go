package systems

import (
	"log"
	"strings"

	"disk-health-exporter/internal/disk/tools"
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
		megacli  bool
		storcli  bool
		arcconf  bool
		zpool    bool
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
	w.toolsAvailable.megacli = utils.CommandExists("megacli") || utils.CommandExists("MegaCli64")
	w.toolsAvailable.storcli = utils.CommandExists("storcli") || utils.CommandExists("storcli64")
	w.toolsAvailable.arcconf = utils.CommandExists("arcconf")
	w.toolsAvailable.zpool = utils.CommandExists("zpool")

	log.Printf("Windows tool availability detected: smartctl=%v, nvme=%v, megacli=%v, storcli=%v, arcconf=%v, zpool=%v",
		w.toolsAvailable.smartctl, w.toolsAvailable.nvme, w.toolsAvailable.megacli,
		w.toolsAvailable.storcli, w.toolsAvailable.arcconf, w.toolsAvailable.zpool)

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

	// Handle RAID arrays and their individual disks
	if w.toolsAvailable.megacli {
		megaTool := tools.NewMegaCLITool()
		if megaTool.IsAvailable() {
			raids := megaTool.GetRAIDArrays()
			allRAIDs = append(allRAIDs, raids...)
			// Also get individual disks from RAID arrays
			raidDisks := megaTool.GetRAIDDisks()
			filtered := w.filterDisks(raidDisks)
			allDisks = w.mergeDisks(allDisks, filtered)
			log.Printf("Found %d hardware RAID arrays and %d RAID disks via MegaCLI", len(raids), len(filtered))
		}
	}

	if w.toolsAvailable.storcli {
		storeTool := tools.NewStoreCLITool()
		if storeTool.IsAvailable() {
			raids := storeTool.GetRAIDArrays()
			allRAIDs = append(allRAIDs, raids...)
			raidDisks := storeTool.GetRAIDDisks()
			filtered := w.filterDisks(raidDisks)
			allDisks = w.mergeDisks(allDisks, filtered)
			log.Printf("Found %d hardware RAID arrays and %d RAID disks via StoreCLI", len(raids), len(filtered))
		}
	}

	if w.toolsAvailable.arcconf {
		arcconfTool := tools.NewArcconfTool()
		if arcconfTool.IsAvailable() {
			raids := arcconfTool.GetRAIDArrays()
			allRAIDs = append(allRAIDs, raids...)
			raidDisks := arcconfTool.GetRAIDDisks()
			filtered := w.filterDisks(raidDisks)
			allDisks = w.mergeDisks(allDisks, filtered)
			log.Printf("Found %d hardware RAID arrays and %d RAID disks via arcconf", len(raids), len(filtered))
		}
	}

	if w.toolsAvailable.zpool {
		zpoolTool := tools.NewZpoolTool()
		if zpoolTool.IsAvailable() {
			zfsPools := zpoolTool.GetZFSPools()
			// Convert ZFS pools to RAIDInfo format (they're already in that format)
			allRAIDs = append(allRAIDs, zfsPools...)
			zfsDisks := zpoolTool.GetDisks()
			filtered := w.filterDisks(zfsDisks)
			allDisks = w.mergeDisks(allDisks, filtered)
			log.Printf("Found %d ZFS pools and %d ZFS disks via zpool", len(zfsPools), len(filtered))
		}
	}

	// TODO: Implement filesystem usage collection for Windows
	// This would require Windows-specific commands like 'wmic' or PowerShell

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

// mergeDisks merges disk information from different sources
func (w *WindowsSystem) mergeDisks(existing []types.DiskInfo, newDisks []types.DiskInfo) []types.DiskInfo {
	diskMap := make(map[string]types.DiskInfo)

	// Add existing disks to map
	for _, disk := range existing {
		diskMap[disk.Device] = disk
	}

	// Merge or add new disks
	for _, newDisk := range newDisks {
		if existingDisk, exists := diskMap[newDisk.Device]; exists {
			// Merge information (new information takes precedence for non-empty fields)
			merged := existingDisk
			if newDisk.Model != "" {
				merged.Model = newDisk.Model
			}
			if newDisk.Serial != "" {
				merged.Serial = newDisk.Serial
			}
			if newDisk.Health != "" {
				merged.Health = newDisk.Health
			}
			if newDisk.Temperature > 0 {
				merged.Temperature = newDisk.Temperature
			}
			if newDisk.Capacity > 0 {
				merged.Capacity = newDisk.Capacity
			}
			if newDisk.Type != "" {
				merged.Type = newDisk.Type
			}
			if newDisk.Location != "" {
				merged.Location = newDisk.Location
			}
			if newDisk.Interface != "" {
				merged.Interface = newDisk.Interface
			}
			diskMap[newDisk.Device] = merged
		} else {
			diskMap[newDisk.Device] = newDisk
		}
	}

	// Convert back to slice
	var result []types.DiskInfo
	for _, disk := range diskMap {
		result = append(result, disk)
	}

	return result
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
	toolInfo.MegaCLI = w.toolsAvailable.megacli
	toolInfo.Storcli = w.toolsAvailable.storcli
	toolInfo.Arcconf = w.toolsAvailable.arcconf
	toolInfo.Zpool = w.toolsAvailable.zpool

	// Get versions for available tools
	if toolInfo.SmartCtl {
		if version, err := utils.GetToolVersion("smartctl", "--version"); err == nil {
			toolInfo.SmartCtlVersion = version
		}
	}
	if toolInfo.MegaCLI {
		cmd := "megacli"
		if utils.CommandExists("MegaCli64") {
			cmd = "MegaCli64"
		}
		if version, err := utils.GetToolVersion(cmd, "-v"); err == nil {
			toolInfo.MegaCLIVersion = version
		}
	}
	if toolInfo.Storcli {
		cmd := "storcli"
		if utils.CommandExists("storcli64") {
			cmd = "storcli64"
		}
		if version, err := utils.GetToolVersion(cmd, "version"); err == nil {
			toolInfo.StorCLIVersion = version
		}
	}
	if toolInfo.Arcconf {
		if version, err := utils.GetToolVersion("arcconf", "version"); err == nil {
			toolInfo.ArcconfVersion = version
		}
	}
	if toolInfo.Zpool {
		if version, err := utils.GetToolVersion("zpool", "version"); err == nil {
			toolInfo.ZpoolVersion = version
		}
	}

	return toolInfo
}
