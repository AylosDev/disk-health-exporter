package systems

import (
	"log"
	"strings"

	"disk-health-exporter/internal/disk/tools"
	"disk-health-exporter/internal/utils"
	"disk-health-exporter/pkg/types"
)

type LinuxSystem struct {
	targetDisks    []string
	ignorePatterns []string
	toolsAvailable struct {
		lsblk    bool
		smartctl bool
		nvme     bool
		megacli  bool
		mdadm    bool
		arcconf  bool
		storcli  bool
		zpool    bool
		hdparm   bool
	}
}

// NewLinuxSystem creates a new LinuxSystem instance
func NewLinuxSystem(targetDisks []string, ignorePatterns []string) *LinuxSystem {
	l := &LinuxSystem{
		targetDisks:    targetDisks,
		ignorePatterns: ignorePatterns,
	}

	// Check tool availability once at startup
	l.toolsAvailable.lsblk = utils.CommandExists("lsblk")
	l.toolsAvailable.smartctl = utils.CommandExists("smartctl")
	l.toolsAvailable.nvme = utils.CommandExists("nvme")
	l.toolsAvailable.megacli = utils.CommandExists("megacli") || utils.CommandExists("MegaCli64")
	l.toolsAvailable.mdadm = utils.CommandExists("mdadm")
	l.toolsAvailable.arcconf = utils.CommandExists("arcconf")
	l.toolsAvailable.storcli = utils.CommandExists("storcli") || utils.CommandExists("storcli64")
	l.toolsAvailable.zpool = utils.CommandExists("zpool")
	l.toolsAvailable.hdparm = utils.CommandExists("hdparm")

	log.Printf("Linux tool availability detected: lsblk=%v, smartctl=%v, nvme=%v, megacli=%v, mdadm=%v, arcconf=%v, storcli=%v, zpool=%v, hdparm=%v",
		l.toolsAvailable.lsblk, l.toolsAvailable.smartctl, l.toolsAvailable.nvme,
		l.toolsAvailable.megacli, l.toolsAvailable.mdadm, l.toolsAvailable.arcconf,
		l.toolsAvailable.storcli, l.toolsAvailable.zpool, l.toolsAvailable.hdparm)

	return l
}

// GetDisks gets all disks on Linux systems using multiple tools
func (l *LinuxSystem) GetDisks() ([]types.DiskInfo, []types.RAIDInfo) {
	var allDisks []types.DiskInfo
	var allRAIDs []types.RAIDInfo

	log.Println("Detecting disks on Linux system...")

	// Use available tools to detect disks
	if l.toolsAvailable.lsblk {
		lsblkTool := tools.NewLsblkTool()
		disks := lsblkTool.GetDisks()
		filtered := l.filterDisks(disks)
		allDisks = append(allDisks, filtered...)
		log.Printf("Found %d disks via lsblk", len(filtered))
	}

	if l.toolsAvailable.smartctl {
		smartTool := tools.NewSmartCtlTool()
		disks := smartTool.GetDisks()
		filtered := l.filterDisks(disks)
		allDisks = l.mergeDisks(allDisks, filtered)
		log.Printf("Enhanced disk info via smartctl")
	}

	if l.toolsAvailable.nvme {
		nvmeTool := tools.NewNvmeTool()
		disks := nvmeTool.GetDisks()
		filtered := l.filterDisks(disks)
		allDisks = l.mergeDisks(allDisks, filtered)
		log.Printf("Found %d NVMe disks", len(filtered))
	}

	// Handle RAID arrays
	if l.toolsAvailable.megacli {
		megaTool := tools.NewMegaCLITool()
		if megaTool.IsAvailable() {
			raids := megaTool.GetRAIDArrays()
			allRAIDs = append(allRAIDs, raids...)
			log.Printf("Found %d hardware RAID arrays via MegaCLI", len(raids))
		}
	}

	if l.toolsAvailable.mdadm {
		mdadmTool := tools.NewMdadmTool()
		softwareRAIDs := mdadmTool.GetSoftwareRAIDs()
		// Convert to RAIDInfo format
		for _, sr := range softwareRAIDs {
			raid := types.RAIDInfo{
				Controller:      "mdadm",
				ArrayID:         sr.Device,
				RaidLevel:       sr.Level,
				Status:          utils.GetSoftwareRAIDStatusValue(sr.State),
				Size:            sr.ArraySize,
				NumDrives:       sr.TotalDevices,
				NumActiveDrives: sr.RaidDevices,
				Type:            "software",
				State:           sr.State,
			}
			allRAIDs = append(allRAIDs, raid)
		}
		log.Printf("Found %d software RAID arrays via mdadm", len(softwareRAIDs))
	}

	log.Printf("Total: %d disks, %d RAID arrays", len(allDisks), len(allRAIDs))
	return allDisks, allRAIDs
}

// filterDisks filters disks based on target and ignore patterns
func (l *LinuxSystem) filterDisks(disks []types.DiskInfo) []types.DiskInfo {
	var filtered []types.DiskInfo
	for _, disk := range disks {
		if l.shouldIncludeDisk(disk.Device) {
			filtered = append(filtered, disk)
		}
	}
	return filtered
}

// shouldIncludeDisk checks if a disk should be included based on configuration
func (l *LinuxSystem) shouldIncludeDisk(device string) bool {
	// First check ignore patterns
	for _, pattern := range l.ignorePatterns {
		if strings.HasPrefix(device, pattern) {
			log.Printf("Ignoring disk %s (matches ignore pattern: %s)", device, pattern)
			return false
		}
	}

	// If target disks are specified, only include those
	if len(l.targetDisks) > 0 {
		for _, target := range l.targetDisks {
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

// mergeDisks merges disk information from different sources
func (l *LinuxSystem) mergeDisks(existing []types.DiskInfo, newDisks []types.DiskInfo) []types.DiskInfo {
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

// GetSystemType returns the system type
func (l *LinuxSystem) GetSystemType() string {
	return "linux"
}

// GetToolInfo reports which tools are available on this Linux system
func (l *LinuxSystem) GetToolInfo() types.ToolInfo {
	var toolInfo types.ToolInfo

	// Use cached tool availability
	toolInfo.SmartCtl = l.toolsAvailable.smartctl
	toolInfo.MegaCLI = l.toolsAvailable.megacli
	toolInfo.Mdadm = l.toolsAvailable.mdadm
	toolInfo.Arcconf = l.toolsAvailable.arcconf
	toolInfo.Storcli = l.toolsAvailable.storcli
	toolInfo.Zpool = l.toolsAvailable.zpool
	toolInfo.Nvme = l.toolsAvailable.nvme
	toolInfo.Hdparm = l.toolsAvailable.hdparm
	toolInfo.Lsblk = l.toolsAvailable.lsblk

	// Get tool versions
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

	return toolInfo
}
