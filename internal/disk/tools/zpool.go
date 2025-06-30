package tools

import (
	"log"
	"os/exec"
	"strings"

	"disk-health-exporter/internal/utils"
	"disk-health-exporter/pkg/types"
)

// ZpoolTool represents the zpool CLI tool for ZFS management
type ZpoolTool struct{}

// NewZpoolTool creates a new ZpoolTool instance
func NewZpoolTool() *ZpoolTool {
	return &ZpoolTool{}
}

// IsAvailable checks if zpool is available on the system
func (z *ZpoolTool) IsAvailable() bool {
	return utils.CommandExists("zpool")
}

// GetVersion returns the zpool version
func (z *ZpoolTool) GetVersion() string {
	if !z.IsAvailable() {
		return ""
	}

	version, err := utils.GetToolVersion("zpool", "version")
	if err != nil {
		return "unknown"
	}
	return version
}

// GetName returns the tool name
func (z *ZpoolTool) GetName() string {
	return "zpool"
}

// GetDisks returns disk information detected by zpool
func (z *ZpoolTool) GetDisks() []types.DiskInfo {
	var disks []types.DiskInfo

	if !z.IsAvailable() {
		return disks
	}

	// Get all pools first
	pools := z.getPools()

	for _, pool := range pools {
		poolDisks := z.getDisksForPool(pool)
		disks = append(disks, poolDisks...)
	}

	return disks
}

// GetZFSPools returns ZFS pool information (similar to RAID arrays)
func (z *ZpoolTool) GetZFSPools() []types.RAIDInfo {
	var pools []types.RAIDInfo

	if !z.IsAvailable() {
		return pools
	}

	// Get pool list with status
	output, err := exec.Command("zpool", "list", "-H", "-o", "name,size,alloc,free,health").Output()
	if err != nil {
		log.Printf("Error getting zpool list: %v", err)
		return pools
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) >= 5 {
			pool := types.RAIDInfo{
				ArrayID:    fields[0],
				RaidLevel:  "ZFS Pool",
				State:      fields[4],
				Status:     z.getZFSStatusValue(fields[4]),
				Size:       utils.ParseSizeToBytes(fields[1]),
				Type:       "zfs",
				Controller: "zpool",
			}

			// Get additional pool information
			z.enrichPoolInfo(&pool)
			pools = append(pools, pool)
		}
	}

	return pools
}

// getPools gets list of ZFS pools
func (z *ZpoolTool) getPools() []string {
	var pools []string

	output, err := exec.Command("zpool", "list", "-H", "-o", "name").Output()
	if err != nil {
		log.Printf("Error getting zpool names: %v", err)
		return pools
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			pools = append(pools, strings.TrimSpace(line))
		}
	}

	return pools
}

// getDisksForPool gets physical disks for a specific ZFS pool
func (z *ZpoolTool) getDisksForPool(poolName string) []types.DiskInfo {
	var disks []types.DiskInfo

	// Get pool status to see physical devices
	output, err := exec.Command("zpool", "status", "-v", poolName).Output()
	if err != nil {
		log.Printf("Error getting zpool status for %s: %v", poolName, err)
		return disks
	}

	lines := strings.Split(string(output), "\n")
	var inConfig bool

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		if strings.Contains(line, "config:") {
			inConfig = true
			continue
		}

		if inConfig && trimmedLine == "" {
			break
		}

		if inConfig && z.isPhysicalDevice(trimmedLine) {
			disk := z.parseZFSDevice(trimmedLine, poolName)
			if disk.Device != "" {
				disks = append(disks, disk)
			}
		}
	}

	return disks
}

// isPhysicalDevice checks if a line represents a physical device
func (z *ZpoolTool) isPhysicalDevice(line string) bool {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return false
	}

	deviceName := fields[0]

	// Skip pool names, vdev types, and special entries
	skipPatterns := []string{
		"mirror", "raidz", "raidz1", "raidz2", "raidz3",
		"spare", "cache", "log", "special", "dedup",
		"errors:", "NAME", "STATE", "READ", "WRITE", "CKSUM",
	}

	for _, pattern := range skipPatterns {
		if strings.Contains(strings.ToLower(deviceName), strings.ToLower(pattern)) {
			return false
		}
	}

	// Look for device patterns
	return strings.HasPrefix(deviceName, "/dev/") ||
		strings.Contains(deviceName, "sd") ||
		strings.Contains(deviceName, "nvme") ||
		strings.Contains(deviceName, "ada") // FreeBSD
}

// parseZFSDevice parses a ZFS device line from zpool status
func (z *ZpoolTool) parseZFSDevice(line, poolName string) types.DiskInfo {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return types.DiskInfo{}
	}

	deviceName := fields[0]
	state := fields[1]

	disk := types.DiskInfo{
		Device:   deviceName,
		Type:     "zfs",
		Health:   z.convertZFSDeviceState(state),
		Location: "Pool: " + poolName,
	}

	// Try to get additional info about the device
	z.enrichZFSDeviceInfo(&disk)

	return disk
}

// enrichPoolInfo gets additional information about a ZFS pool
func (z *ZpoolTool) enrichPoolInfo(pool *types.RAIDInfo) {
	// Get detailed pool information
	output, err := exec.Command("zpool", "status", pool.ArrayID).Output()
	if err != nil {
		return
	}

	lines := strings.Split(string(output), "\n")
	var deviceCount int

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		if z.isPhysicalDevice(trimmedLine) {
			deviceCount++
		}

		// Look for RAID level information
		if strings.Contains(line, "mirror") {
			pool.RaidLevel = "ZFS Mirror"
		} else if strings.Contains(line, "raidz3") {
			pool.RaidLevel = "ZFS RAIDZ3"
		} else if strings.Contains(line, "raidz2") {
			pool.RaidLevel = "ZFS RAIDZ2"
		} else if strings.Contains(line, "raidz1") || strings.Contains(line, "raidz") {
			pool.RaidLevel = "ZFS RAIDZ1"
		}
	}

	pool.NumDrives = deviceCount
	pool.NumActiveDrives = deviceCount // ZFS doesn't have inactive drives concept
}

// enrichZFSDeviceInfo adds additional information to ZFS device
func (z *ZpoolTool) enrichZFSDeviceInfo(disk *types.DiskInfo) {
	if !strings.HasPrefix(disk.Device, "/dev/") {
		return
	}

	// Try to get device information using other tools if available
	if utils.CommandExists("lsblk") {
		output, err := exec.Command("lsblk", "-d", "-o", "SIZE,MODEL", disk.Device).Output()
		if err == nil {
			lines := strings.Split(string(output), "\n")
			if len(lines) >= 2 {
				fields := strings.Fields(lines[1])
				if len(fields) >= 1 {
					disk.Capacity = utils.ParseSizeToBytes(fields[0])
				}
				if len(fields) >= 2 {
					disk.Model = strings.Join(fields[1:], " ")
				}
			}
		}
	}
}

// convertZFSDeviceState converts ZFS device state to health status
func (z *ZpoolTool) convertZFSDeviceState(state string) string {
	state = strings.ToLower(strings.TrimSpace(state))

	switch state {
	case "online":
		return "OK"
	case "degraded", "faulted":
		return "DEGRADED"
	case "offline", "removed", "unavail":
		return "FAILED"
	case "resilvering":
		return "REBUILDING"
	default:
		return "UNKNOWN"
	}
}

// getZFSStatusValue converts ZFS pool health to numeric value
func (z *ZpoolTool) getZFSStatusValue(health string) int {
	health = strings.ToLower(strings.TrimSpace(health))

	switch health {
	case "online":
		return 1
	case "degraded":
		return 2
	case "faulted", "offline", "unavail":
		return 3
	default:
		return 0
	}
}
