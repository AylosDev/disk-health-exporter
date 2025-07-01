package systems

import (
	"log"
	"os/exec"
	"strconv"
	"strings"

	"disk-health-exporter/internal/disk/tools"
	"disk-health-exporter/internal/utils"
	"disk-health-exporter/pkg/types"
)

// MacOSSystem represents the macOS-specific disk detection implementation
type MacOSSystem struct {
	targetDisks    []string
	ignorePatterns []string
	toolsAvailable struct {
		diskutil bool
		smartctl bool
		nvme     bool
		zpool    bool // ZFS is available on macOS via OpenZFS
	}
}

// NewMacOSSystem creates a new MacOSSystem instance
func NewMacOSSystem(targetDisks []string, ignorePatterns []string) *MacOSSystem {
	m := &MacOSSystem{
		targetDisks:    targetDisks,
		ignorePatterns: ignorePatterns,
	}

	// Check tool availability once at startup
	m.toolsAvailable.diskutil = utils.CommandExists("diskutil")
	m.toolsAvailable.smartctl = utils.CommandExists("smartctl")
	m.toolsAvailable.nvme = utils.CommandExists("nvme")
	m.toolsAvailable.zpool = utils.CommandExists("zpool")

	log.Printf("macOS tool availability detected: diskutil=%v, smartctl=%v, nvme=%v, zpool=%v",
		m.toolsAvailable.diskutil, m.toolsAvailable.smartctl, m.toolsAvailable.nvme, m.toolsAvailable.zpool)

	return m
}

// GetDisks gets all disks on macOS systems using available tools
func (m *MacOSSystem) GetDisks() ([]types.DiskInfo, []types.RAIDInfo) {
	var allDisks []types.DiskInfo
	var allRAIDs []types.RAIDInfo

	log.Println("Detecting disks on macOS system...")

	// Use diskutil for macOS disk detection
	if m.toolsAvailable.diskutil {
		disks := m.getDiskutilDisks()
		filtered := m.filterDisks(disks)
		allDisks = append(allDisks, filtered...)
		log.Printf("Found %d disks via diskutil", len(filtered))
	}

	// Enhance with SMART data if available
	if m.toolsAvailable.smartctl {
		for i, disk := range allDisks {
			smartInfo := m.getSmartInfo(disk.Device)
			if smartInfo.Device != "" {
				allDisks[i] = m.mergeDiskInfo(disk, smartInfo)
			}
		}
	}

	// Handle ZFS pools (ZFS is available on macOS via OpenZFS)
	if m.toolsAvailable.zpool {
		zpoolTool := tools.NewZpoolTool()
		if zpoolTool.IsAvailable() {
			zfsPools := zpoolTool.GetZFSPools()
			// Convert ZFS pools to RAIDInfo format (they're already in that format)
			allRAIDs = append(allRAIDs, zfsPools...)
			zfsDisks := zpoolTool.GetDisks()
			filtered := m.filterDisks(zfsDisks)
			allDisks = m.mergeDisks(allDisks, filtered)
			log.Printf("Found %d ZFS pools and %d ZFS disks via zpool", len(zfsPools), len(filtered))
		}
	}

	log.Printf("Total: %d disks, %d RAID arrays", len(allDisks), len(allRAIDs))
	return allDisks, allRAIDs
}

// getDiskutilDisks gets disk information using diskutil
func (m *MacOSSystem) getDiskutilDisks() []types.DiskInfo {
	var disks []types.DiskInfo

	// Get list of all disks using diskutil (regular format)
	cmd := exec.Command("diskutil", "list")
	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error running diskutil list: %v", err)
		return disks
	}

	// Parse the output to get disk identifiers
	diskIdentifiers := m.parseDiskutilList(string(output))
	log.Printf("Found %d disk identifiers from diskutil", len(diskIdentifiers))

	// Get detailed information for each physical disk
	for _, diskID := range diskIdentifiers {
		// Skip non-physical disks (like disk images, APFS volumes, etc.)
		if !m.isPhysicalDisk(diskID) {
			continue
		}

		diskInfo := m.getDiskutilInfo(diskID)
		if diskInfo.Device != "" {
			disks = append(disks, diskInfo)
		}
	}

	return disks
}

// parseDiskutilList parses diskutil list output to extract disk identifiers
func (m *MacOSSystem) parseDiskutilList(output string) []string {
	var identifiers []string

	// Parse diskutil list output which has format like:
	// /dev/disk0 (internal, physical):
	// /dev/disk3 (synthesized):
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "/dev/disk") && strings.Contains(line, ":") {
			// Extract disk identifier from lines like "/dev/disk0 (internal, physical):"
			parts := strings.Fields(line)
			if len(parts) > 0 {
				devicePath := parts[0]
				// Extract just the disk identifier (e.g., "disk0" from "/dev/disk0")
				diskID := strings.TrimPrefix(devicePath, "/dev/")
				identifiers = append(identifiers, diskID)
			}
		}
	}

	return identifiers
}

// isPhysicalDisk checks if the disk identifier represents a physical disk
func (m *MacOSSystem) isPhysicalDisk(diskID string) bool {
	// Additional check: run diskutil info to see if it's a physical disk
	cmd := exec.Command("diskutil", "info", diskID)
	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error checking if %s is physical disk: %v", diskID, err)
		return false
	}

	outputStr := string(output)

	// Look for indicators that this is a physical disk
	// Must have "Device Location: Internal" or "Device Location: External" and not be virtual
	hasDeviceLocation := strings.Contains(outputStr, "Device Location:") &&
		(strings.Contains(outputStr, "Internal") || strings.Contains(outputStr, "External"))

	// Check if it's a real physical media
	hasPhysicalMedia := strings.Contains(outputStr, "Media Type:") &&
		!strings.Contains(outputStr, "Disk Image")

	// Exclude virtual disks and APFS containers
	isVirtual := strings.Contains(outputStr, "Virtual:                   Yes") ||
		strings.Contains(outputStr, "APFS Container") ||
		strings.Contains(outputStr, "synthesized") ||
		strings.Contains(outputStr, "Disk Image")

	result := hasDeviceLocation && hasPhysicalMedia && !isVirtual
	return result
}

// getDiskutilInfo gets detailed information for a specific disk
func (m *MacOSSystem) getDiskutilInfo(diskID string) types.DiskInfo {
	disk := types.DiskInfo{
		Device:    "/dev/" + diskID,
		Type:      "macos-disk",
		Interface: "Unknown",
	}

	// Run diskutil info to get detailed information
	cmd := exec.Command("diskutil", "info", diskID)
	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error getting diskutil info for %s: %v", diskID, err)
		return types.DiskInfo{}
	}

	outputStr := string(output)
	lines := strings.Split(outputStr, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Parse various fields from diskutil info output
		if strings.Contains(line, "Device / Media Name:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				disk.Model = strings.TrimSpace(parts[1])
			}
		} else if strings.Contains(line, "Disk Size:") {
			// Extract capacity from "Disk Size: 500.1 GB (500107862016 Bytes)"
			parts := strings.SplitN(line, "(", 2)
			if len(parts) == 2 {
				bytesStr := strings.Split(parts[1], " ")[0]
				if capacity, err := strconv.ParseInt(bytesStr, 10, 64); err == nil {
					disk.Capacity = capacity
				}
			}
		} else if strings.Contains(line, "Protocol:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				disk.Interface = strings.TrimSpace(parts[1])
			}
		} else if strings.Contains(line, "Solid State:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 && strings.TrimSpace(parts[1]) == "Yes" {
				disk.RPM = 0 // SSD
			}
		} else if strings.Contains(line, "Physical Drive:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				vendor := strings.TrimSpace(parts[1])
				// Extract vendor from strings like "APPLE SSD SM0512F Media"
				vendorParts := strings.Fields(vendor)
				if len(vendorParts) > 0 {
					disk.Vendor = vendorParts[0]
				}
			}
		}
	}

	// Set default health status
	disk.Health = "OK"
	disk.SmartEnabled = true
	disk.SmartHealthy = true

	// Get filesystem usage information
	m.addFilesystemUsage(&disk, diskID)

	return disk
}

// getSmartInfo gets SMART information for a disk using smartctl
func (m *MacOSSystem) getSmartInfo(device string) types.DiskInfo {
	disk := types.DiskInfo{Device: device}

	// Run smartctl to get SMART data with -d auto for better macOS compatibility
	cmd := exec.Command("smartctl", "-a", "-d", "auto", device)
	output, err := cmd.Output()
	if err != nil {
		// Try without -d auto if that fails
		cmd = exec.Command("smartctl", "-a", device)
		output, err = cmd.Output()
		if err != nil {
			log.Printf("smartctl failed for %s: %v", device, err)
			return disk
		}
	}

	outputStr := string(output)
	lines := strings.Split(outputStr, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Parse SMART data
		if strings.Contains(line, "Model Family:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				disk.Vendor = strings.TrimSpace(parts[1])
			}
		} else if strings.Contains(line, "Device Model:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				disk.Model = strings.TrimSpace(parts[1])
			}
		} else if strings.Contains(line, "Serial Number:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				disk.Serial = strings.TrimSpace(parts[1])
			}
		} else if strings.Contains(line, "User Capacity:") {
			// Extract capacity from "User Capacity: 500,107,862,016 bytes [500 GB]"
			parts := strings.Fields(line)
			for i, part := range parts {
				if part == "bytes" && i > 0 {
					capacityStr := strings.ReplaceAll(parts[i-1], ",", "")
					if capacity, err := strconv.ParseInt(capacityStr, 10, 64); err == nil {
						disk.Capacity = capacity
					}
					break
				}
			}
		} else if strings.Contains(line, "Rotation Rate:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				rpmStr := strings.TrimSpace(parts[1])
				if strings.Contains(rpmStr, "Solid State") {
					disk.RPM = 0
				} else {
					// Extract RPM number
					rpmFields := strings.Fields(rpmStr)
					if len(rpmFields) > 0 {
						if rpm, err := strconv.Atoi(rpmFields[0]); err == nil {
							disk.RPM = rpm
						}
					}
				}
			}
		} else if strings.Contains(line, "SMART overall-health") {
			if strings.Contains(line, "PASSED") {
				disk.Health = "OK"
				disk.SmartHealthy = true
			} else {
				disk.Health = "FAILED"
				disk.SmartHealthy = false
			}
			disk.SmartEnabled = true
		} else if strings.Contains(line, "Current Temperature:") ||
			strings.Contains(line, "Temperature_Celsius") {
			// Parse temperature from various formats
			fields := strings.Fields(line)
			for _, field := range fields {
				if temp, err := strconv.ParseFloat(field, 64); err == nil && temp > 0 && temp < 100 {
					disk.Temperature = temp
					break
				}
			}
		} else if strings.Contains(line, "Power_On_Hours") {
			// Parse power on hours from SMART attributes
			fields := strings.Fields(line)
			if len(fields) >= 10 {
				if hours, err := strconv.ParseInt(fields[9], 10, 64); err == nil {
					disk.PowerOnHours = hours
				}
			}
		}
	}

	// Set defaults if not found
	if disk.Health == "" {
		disk.Health = "Unknown"
	}

	return disk
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

// mergeDisks merges disk information from different sources
func (m *MacOSSystem) mergeDisks(existing []types.DiskInfo, newDisks []types.DiskInfo) []types.DiskInfo {
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

	// Use cached tool availability
	toolInfo.Diskutil = m.toolsAvailable.diskutil
	toolInfo.SmartCtl = m.toolsAvailable.smartctl
	toolInfo.Nvme = m.toolsAvailable.nvme
	toolInfo.Zpool = m.toolsAvailable.zpool

	// Get versions for available tools
	if toolInfo.SmartCtl {
		if version, err := utils.GetToolVersion("smartctl", "--version"); err == nil {
			toolInfo.SmartCtlVersion = version
		}
	}
	if toolInfo.Zpool {
		if version, err := utils.GetToolVersion("zpool", "version"); err == nil {
			toolInfo.ZpoolVersion = version
		}
	}

	return toolInfo
}

// addFilesystemUsage adds filesystem usage information to a disk using diskutil
func (m *MacOSSystem) addFilesystemUsage(disk *types.DiskInfo, diskID string) {
	// First, check if this disk has any mounted volumes
	cmd := exec.Command("diskutil", "list", diskID)
	output, err := cmd.Output()
	if err != nil {
		return
	}

	// Parse the partition list to find mounted volumes
	lines := strings.Split(string(output), "\n")
	var mountedPartition string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Look for partition lines (they start with numbers and contain volume info)
		if strings.Contains(line, diskID+"s") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				partitionID := fields[1]
				// Check if this partition is mounted
				if m.isPartitionMounted(partitionID) {
					mountedPartition = partitionID
					break
				}
			}
		}
	}

	// If we found a mounted partition, get its info
	if mountedPartition != "" {
		cmd = exec.Command("diskutil", "info", mountedPartition)
		output, err = cmd.Output()
		if err != nil {
			return
		}

		outputStr := string(output)
		lines = strings.Split(outputStr, "\n")

		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "Mount Point:") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					mountpoint := strings.TrimSpace(parts[1])
					if mountpoint != "Not applicable (no filesystem)" {
						disk.Mountpoint = mountpoint
					}
				}
			}
			if strings.HasPrefix(line, "File System Personality:") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					disk.Filesystem = strings.TrimSpace(parts[1])
				}
			}
		}

		// If we have a mountpoint, get usage stats using df
		if disk.Mountpoint != "" {
			cmd = exec.Command("df", "-k", disk.Mountpoint)
			output, err = cmd.Output()
			if err != nil {
				return
			}

			lines = strings.Split(strings.TrimSpace(string(output)), "\n")
			if len(lines) >= 2 {
				fields := strings.Fields(lines[1])
				if len(fields) >= 4 {
					// df -k output: Filesystem 1K-blocks Used Available Capacity Mounted on
					if used, err := strconv.ParseInt(fields[2], 10, 64); err == nil {
						if avail, err := strconv.ParseInt(fields[3], 10, 64); err == nil {
							disk.UsedBytes = used * 1024       // Convert from KB to bytes
							disk.AvailableBytes = avail * 1024 // Convert from KB to bytes
							total := disk.UsedBytes + disk.AvailableBytes
							if total > 0 {
								disk.UsagePercentage = float64(disk.UsedBytes) / float64(total) * 100
							}
						}
					}
				}
			}
		}
	}
}

// isPartitionMounted checks if a partition is currently mounted
func (m *MacOSSystem) isPartitionMounted(partitionID string) bool {
	cmd := exec.Command("diskutil", "info", partitionID)
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	outputStr := string(output)
	return strings.Contains(outputStr, "Mount Point:") &&
		!strings.Contains(outputStr, "Not applicable (no filesystem)")
}
