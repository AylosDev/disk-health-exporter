package systems

import (
	"fmt"
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

	return l
}

// GetDisks gets all disks on Linux systems using multiple tools
func (l *LinuxSystem) GetDisks() ([]types.DiskInfo, []types.RAIDInfo) {
	var allDisks []types.DiskInfo
	var allRAIDs []types.RAIDInfo

	// Use available tools to detect disks
	if l.toolsAvailable.lsblk {
		lsblkTool := tools.NewLsblkTool()
		disks := lsblkTool.GetDisks()
		filtered := l.filterDisks(disks)
		allDisks = append(allDisks, filtered...)
	}

	if l.toolsAvailable.smartctl {
		smartTool := tools.NewSmartCtlTool()
		disks := smartTool.GetDisks()
		filtered := l.filterDisks(disks)
		allDisks = l.mergeDisks(allDisks, filtered)
	}

	if l.toolsAvailable.nvme {
		nvmeTool := tools.NewNvmeTool()
		disks := nvmeTool.GetDisks()
		filtered := l.filterDisks(disks)
		allDisks = l.mergeDisks(allDisks, filtered)
	}

	if l.toolsAvailable.hdparm {
		hdparmTool := tools.NewHdparmTool()
		disks := hdparmTool.GetDisks()
		filtered := l.filterDisks(disks)
		allDisks = l.mergeDisks(allDisks, filtered)
	}

	// Handle RAID arrays
	if l.toolsAvailable.megacli {
		megaTool := tools.NewMegaCLITool()
		if megaTool.IsAvailable() {
			raids := megaTool.GetRAIDArrays()
			allRAIDs = append(allRAIDs, raids...)
			// Get individual disks with utilization calculations
			raidDisks := megaTool.GetRAIDDisks()
			filtered := l.filterDisks(raidDisks)
			allDisks = l.mergeDisks(allDisks, filtered)
		}
	}

	if l.toolsAvailable.storcli {
		storeTool := tools.NewStoreCLITool()
		if storeTool.IsAvailable() {
			raids := storeTool.GetRAIDArrays()
			allRAIDs = append(allRAIDs, raids...)
			raidDisks := storeTool.GetRAIDDisks()
			filtered := l.filterDisks(raidDisks)
			allDisks = l.mergeDisks(allDisks, filtered)
		}
	}

	if l.toolsAvailable.arcconf {
		arcconfTool := tools.NewArcconfTool()
		if arcconfTool.IsAvailable() {
			raids := arcconfTool.GetRAIDArrays()
			allRAIDs = append(allRAIDs, raids...)
			raidDisks := arcconfTool.GetRAIDDisks()
			filtered := l.filterDisks(raidDisks)
			allDisks = l.mergeDisks(allDisks, filtered)
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
	}

	if l.toolsAvailable.zpool {
		zpoolTool := tools.NewZpoolTool()
		if zpoolTool.IsAvailable() {
			zfsPools := zpoolTool.GetZFSPools()
			// Convert ZFS pools to RAIDInfo format (they're already in that format)
			allRAIDs = append(allRAIDs, zfsPools...)
			zfsDisks := zpoolTool.GetDisks()
			filtered := l.filterDisks(zfsDisks)
			allDisks = l.mergeDisks(allDisks, filtered)
		}
	}

	// Deduplicate disks to prevent reporting the same physical disk multiple times
	allDisks = l.deduplicateDisks(allDisks)

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
			return false
		}
	}

	// If target disks are specified, only include those
	if len(l.targetDisks) > 0 {
		for _, target := range l.targetDisks {
			if device == target {
				return true
			}
		}
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
			if newDisk.Type != "" {
				merged.Type = newDisk.Type
			}
			if newDisk.Location != "" {
				merged.Location = newDisk.Location
			}
			if newDisk.Interface != "" {
				merged.Interface = newDisk.Interface
			}

			// Merge boolean fields - prioritize true values and explicit health information
			// If either source has SmartEnabled=true, keep it true
			if newDisk.SmartEnabled || existingDisk.SmartEnabled {
				merged.SmartEnabled = true
			}

			// For SmartHealthy, if we have explicit health info from a reliable source (smartctl/storcli), use it
			// Priority: explicit SMART data > RAID controller health data > default false
			if newDisk.SmartEnabled && (newDisk.Type == "regular" || strings.Contains(newDisk.Type, "smart")) {
				// If new disk has SMART enabled from a direct SMART source, trust its health status
				merged.SmartHealthy = newDisk.SmartHealthy
			} else if existingDisk.SmartEnabled && (existingDisk.Type == "regular" || strings.Contains(existingDisk.Type, "smart")) {
				// Keep existing SMART health if it was from a SMART-enabled source
				merged.SmartHealthy = existingDisk.SmartHealthy
			} else if strings.Contains(newDisk.Type, "raid") && newDisk.Health == "OK" {
				// For RAID disks, if health is explicitly OK, trust that
				merged.SmartHealthy = true
			} else if strings.Contains(existingDisk.Type, "raid") && existingDisk.Health == "OK" {
				// For existing RAID disks, if health is explicitly OK, trust that
				merged.SmartHealthy = true
			} else {
				// Use the more recent health assessment, but prefer true over false
				if newDisk.SmartHealthy || existingDisk.SmartHealthy {
					merged.SmartHealthy = true
				} else {
					merged.SmartHealthy = newDisk.SmartHealthy
				}
			}

			// Merge other boolean fields with logical OR (any true wins)
			if newDisk.IsCommissionedSpare || existingDisk.IsCommissionedSpare {
				merged.IsCommissionedSpare = true
			}
			if newDisk.IsEmergencySpare || existingDisk.IsEmergencySpare {
				merged.IsEmergencySpare = true
			}
			if newDisk.IsGlobalSpare || existingDisk.IsGlobalSpare {
				merged.IsGlobalSpare = true
			}
			if newDisk.IsDedicatedSpare || existingDisk.IsDedicatedSpare {
				merged.IsDedicatedSpare = true
			}

			// Merge numeric fields - prefer non-zero values, with newer data taking precedence
			if newDisk.PowerOnHours > 0 {
				merged.PowerOnHours = newDisk.PowerOnHours
			}
			if newDisk.PowerCycles > 0 {
				merged.PowerCycles = newDisk.PowerCycles
			}
			if newDisk.ReallocatedSectors > 0 {
				merged.ReallocatedSectors = newDisk.ReallocatedSectors
			}
			if newDisk.PendingSectors > 0 {
				merged.PendingSectors = newDisk.PendingSectors
			}
			if newDisk.UncorrectableErrors > 0 {
				merged.UncorrectableErrors = newDisk.UncorrectableErrors
			}
			if newDisk.TotalLBAsWritten > 0 {
				merged.TotalLBAsWritten = newDisk.TotalLBAsWritten
			}
			if newDisk.TotalLBAsRead > 0 {
				merged.TotalLBAsRead = newDisk.TotalLBAsRead
			}
			if newDisk.WearLeveling > 0 {
				merged.WearLeveling = newDisk.WearLeveling
			}
			if newDisk.PercentageUsed > 0 {
				merged.PercentageUsed = newDisk.PercentageUsed
			}
			if newDisk.AvailableSpare > 0 {
				merged.AvailableSpare = newDisk.AvailableSpare
			}
			if newDisk.CriticalWarning > 0 {
				merged.CriticalWarning = newDisk.CriticalWarning
			}
			if newDisk.MediaErrors > 0 {
				merged.MediaErrors = newDisk.MediaErrors
			}
			if newDisk.ErrorLogEntries > 0 {
				merged.ErrorLogEntries = newDisk.ErrorLogEntries
			}

			// Merge RAID-specific fields
			if newDisk.RaidRole != "" {
				merged.RaidRole = newDisk.RaidRole
			}
			if newDisk.RaidArrayID != "" {
				merged.RaidArrayID = newDisk.RaidArrayID
			}
			if newDisk.RaidPosition != "" {
				merged.RaidPosition = newDisk.RaidPosition
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

// deduplicateDisks removes duplicate disks that may be reported by multiple tools
// This is conservative - we only deduplicate when we're confident it's the same physical disk
// RAID virtual devices (/dev/sdX) and physical devices (raid-encX-slotY) are kept separate
func (l *LinuxSystem) deduplicateDisks(disks []types.DiskInfo) []types.DiskInfo {
	if len(disks) <= 1 {
		return disks
	}

	// Group disks by serial number and model, but only for devices from the same "class"
	diskGroups := make(map[string][]types.DiskInfo)
	var standaloneDisks []types.DiskInfo

	for _, disk := range disks {
		// Only deduplicate disks with valid serial numbers and from the same device class
		// Don't deduplicate across RAID virtual devices vs physical devices
		if disk.Serial == "" || len(disk.Serial) < 8 {
			// Keep disks without proper serial numbers separate
			standaloneDisks = append(standaloneDisks, disk)
			continue
		}

		// Create device class identifier
		deviceClass := ""
		if strings.HasPrefix(disk.Device, "/dev/") {
			if strings.Contains(disk.Model, "PERC") || strings.Contains(disk.Model, "RAID") {
				deviceClass = "raid_virtual"
			} else {
				deviceClass = "block_device"
			}
		} else if strings.HasPrefix(disk.Device, "raid-enc") {
			deviceClass = "raid_physical"
		} else {
			deviceClass = "other"
		}

		// Create a unique key based on serial, model, and device class
		key := fmt.Sprintf("%s|%s|%s", disk.Serial, disk.Model, deviceClass)
		diskGroups[key] = append(diskGroups[key], disk)
	}

	var result []types.DiskInfo

	// Process each group of potentially duplicate disks
	for _, group := range diskGroups {
		if len(group) == 1 {
			// No duplicates, add as-is
			result = append(result, group[0])
		} else {
			// Multiple disks with same serial/model/class - merge them
			best := l.selectBestDiskFromGroup(group)
			result = append(result, best)

		}
	}

	// Add standalone disks (those without proper serials)
	result = append(result, standaloneDisks...)

	return result
}

// selectBestDiskFromGroup selects the best disk representation from a group of duplicates
// Priority: regular block devices > RAID virtual devices, more complete data > less complete
func (l *LinuxSystem) selectBestDiskFromGroup(group []types.DiskInfo) types.DiskInfo {
	if len(group) == 1 {
		return group[0]
	}

	best := group[0]
	bestScore := l.scoreDisk(best)

	// Try to merge all the information from the group into the best disk
	for i := 1; i < len(group); i++ {
		candidate := group[i]
		candidateScore := l.scoreDisk(candidate)

		if candidateScore > bestScore {
			// Merge the previous best's data into the new best candidate
			best = l.mergeTwoDisks(candidate, best)
			bestScore = candidateScore
		} else {
			// Merge candidate's data into current best
			best = l.mergeTwoDisks(best, candidate)
		}
	}

	return best
}

// scoreDisk assigns a score to a disk based on how "good" its representation is
func (l *LinuxSystem) scoreDisk(disk types.DiskInfo) int {
	score := 0

	// Prefer regular block devices over RAID virtual devices
	if strings.HasPrefix(disk.Device, "/dev/") {
		score += 100
	} else if strings.HasPrefix(disk.Device, "raid-") {
		score += 50
	}

	// Prefer disks with more complete information
	if disk.Model != "" {
		score += 10
	}
	if disk.Serial != "" {
		score += 10
	}
	if disk.Temperature > 0 {
		score += 5
	}
	if disk.Capacity > 0 {
		score += 5
	}
	if disk.PowerOnHours > 0 {
		score += 5
	}
	if disk.SmartEnabled {
		score += 10
	}
	if disk.Interface != "" {
		score += 5
	}
	if disk.Health != "" {
		score += 5
	}

	return score
}

// mergeTwoDisks merges information from source disk into target disk
func (l *LinuxSystem) mergeTwoDisks(target, source types.DiskInfo) types.DiskInfo {
	merged := target

	// Merge non-empty string fields
	if merged.Model == "" && source.Model != "" {
		merged.Model = source.Model
	}
	if merged.Serial == "" && source.Serial != "" {
		merged.Serial = source.Serial
	}
	if merged.Vendor == "" && source.Vendor != "" {
		merged.Vendor = source.Vendor
	}
	if merged.Health == "" && source.Health != "" {
		merged.Health = source.Health
	}
	if merged.Type == "" && source.Type != "" {
		merged.Type = source.Type
	}
	if merged.Location == "" && source.Location != "" {
		merged.Location = source.Location
	}
	if merged.Interface == "" && source.Interface != "" {
		merged.Interface = source.Interface
	}
	if merged.FormFactor == "" && source.FormFactor != "" {
		merged.FormFactor = source.FormFactor
	}
	if merged.Mountpoint == "" && source.Mountpoint != "" {
		merged.Mountpoint = source.Mountpoint
	}
	if merged.Filesystem == "" && source.Filesystem != "" {
		merged.Filesystem = source.Filesystem
	}
	if merged.RaidRole == "" && source.RaidRole != "" {
		merged.RaidRole = source.RaidRole
	}
	if merged.RaidArrayID == "" && source.RaidArrayID != "" {
		merged.RaidArrayID = source.RaidArrayID
	}
	if merged.RaidPosition == "" && source.RaidPosition != "" {
		merged.RaidPosition = source.RaidPosition
	}

	// Merge numeric fields (prefer non-zero values)
	if merged.Temperature == 0 && source.Temperature > 0 {
		merged.Temperature = source.Temperature
	}
	if merged.Capacity == 0 && source.Capacity > 0 {
		merged.Capacity = source.Capacity
	}
	if merged.UsedBytes == 0 && source.UsedBytes > 0 {
		merged.UsedBytes = source.UsedBytes
	}
	if merged.AvailableBytes == 0 && source.AvailableBytes > 0 {
		merged.AvailableBytes = source.AvailableBytes
	}
	if merged.UsagePercentage == 0 && source.UsagePercentage > 0 {
		merged.UsagePercentage = source.UsagePercentage
	}
	if merged.DriveTemperatureMax == 0 && source.DriveTemperatureMax > 0 {
		merged.DriveTemperatureMax = source.DriveTemperatureMax
	}
	if merged.DriveTemperatureMin == 0 && source.DriveTemperatureMin > 0 {
		merged.DriveTemperatureMin = source.DriveTemperatureMin
	}
	if merged.RPM == 0 && source.RPM > 0 {
		merged.RPM = source.RPM
	}
	if merged.PowerOnHours == 0 && source.PowerOnHours > 0 {
		merged.PowerOnHours = source.PowerOnHours
	}
	if merged.PowerCycles == 0 && source.PowerCycles > 0 {
		merged.PowerCycles = source.PowerCycles
	}
	if merged.ReallocatedSectors == 0 && source.ReallocatedSectors > 0 {
		merged.ReallocatedSectors = source.ReallocatedSectors
	}
	if merged.PendingSectors == 0 && source.PendingSectors > 0 {
		merged.PendingSectors = source.PendingSectors
	}
	if merged.UncorrectableErrors == 0 && source.UncorrectableErrors > 0 {
		merged.UncorrectableErrors = source.UncorrectableErrors
	}
	if merged.TotalLBAsWritten == 0 && source.TotalLBAsWritten > 0 {
		merged.TotalLBAsWritten = source.TotalLBAsWritten
	}
	if merged.TotalLBAsRead == 0 && source.TotalLBAsRead > 0 {
		merged.TotalLBAsRead = source.TotalLBAsRead
	}
	if merged.WearLeveling == 0 && source.WearLeveling > 0 {
		merged.WearLeveling = source.WearLeveling
	}
	if merged.PercentageUsed == 0 && source.PercentageUsed > 0 {
		merged.PercentageUsed = source.PercentageUsed
	}
	if merged.AvailableSpare == 0 && source.AvailableSpare > 0 {
		merged.AvailableSpare = source.AvailableSpare
	}
	if merged.CriticalWarning == 0 && source.CriticalWarning > 0 {
		merged.CriticalWarning = source.CriticalWarning
	}
	if merged.MediaErrors == 0 && source.MediaErrors > 0 {
		merged.MediaErrors = source.MediaErrors
	}
	if merged.ErrorLogEntries == 0 && source.ErrorLogEntries > 0 {
		merged.ErrorLogEntries = source.ErrorLogEntries
	}

	// Merge boolean fields (logical OR - any true wins)
	if !merged.SmartEnabled && source.SmartEnabled {
		merged.SmartEnabled = source.SmartEnabled
	}
	if !merged.SmartHealthy && source.SmartHealthy {
		merged.SmartHealthy = source.SmartHealthy
	}
	if !merged.IsCommissionedSpare && source.IsCommissionedSpare {
		merged.IsCommissionedSpare = source.IsCommissionedSpare
	}
	if !merged.IsEmergencySpare && source.IsEmergencySpare {
		merged.IsEmergencySpare = source.IsEmergencySpare
	}
	if !merged.IsGlobalSpare && source.IsGlobalSpare {
		merged.IsGlobalSpare = source.IsGlobalSpare
	}
	if !merged.IsDedicatedSpare && source.IsDedicatedSpare {
		merged.IsDedicatedSpare = source.IsDedicatedSpare
	}

	return merged
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
	if toolInfo.Storcli {
		cmd := "storcli"
		if utils.CommandExists("storcli64") {
			cmd = "storcli64"
		}
		if version, err := utils.GetToolVersion(cmd, "version"); err == nil {
			toolInfo.StorCLIVersion = version
		}
	}
	if toolInfo.Hdparm {
		if version, err := utils.GetToolVersion("hdparm", "-V"); err == nil {
			toolInfo.HdparmVersion = version
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
