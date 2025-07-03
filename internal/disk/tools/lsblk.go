package tools

import (
	"log"
	"os/exec"
	"strconv"
	"strings"

	"disk-health-exporter/internal/utils"
	"disk-health-exporter/pkg/types"
)

// LsblkTool represents the lsblk CLI tool
type LsblkTool struct{}

// NewLsblkTool creates a new LsblkTool instance
func NewLsblkTool() *LsblkTool {
	return &LsblkTool{}
}

// IsAvailable checks if lsblk is available on the system
func (l *LsblkTool) IsAvailable() bool {
	return utils.CommandExists("lsblk")
}

// GetVersion returns the lsblk version
func (l *LsblkTool) GetVersion() string {
	if !l.IsAvailable() {
		return ""
	}

	version, err := utils.GetToolVersion("lsblk", "--version")
	if err != nil {
		return "unknown"
	}
	return version
}

// GetName returns the tool name
func (l *LsblkTool) GetName() string {
	return "lsblk"
}

// GetDisks returns disk information detected by lsblk
// lsblk -d -o NAME,SIZE,MODEL,SERIAL,TRAN -n # list block devices with specified columns in plain format
func (l *LsblkTool) GetDisks() []types.DiskInfo {
	var disks []types.DiskInfo

	if !l.IsAvailable() {
		return disks
	}

	log.Printf("Detecting disks using lsblk...")

	output, err := exec.Command("lsblk", "-d", "-o", "NAME,SIZE,MODEL,SERIAL,TRAN", "-n").Output()
	if err != nil {
		log.Printf("Error running lsblk: %v", err)
		return disks
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) >= 2 {
			device := "/dev/" + fields[0]

			disk := types.DiskInfo{
				Device: device,
				Type:   "regular",
			}

			// Parse additional fields if available
			if len(fields) >= 3 && fields[2] != "-" {
				disk.Model = fields[2]
			}
			if len(fields) >= 4 && fields[3] != "-" {
				disk.Serial = fields[3]
			}
			if len(fields) >= 5 && fields[4] != "-" {
				disk.Interface = fields[4]
			}

			// Get filesystem usage information
			l.addFilesystemUsage(&disk)

			disks = append(disks, disk)
		}
	}

	log.Printf("Found %d disks using lsblk", len(disks))
	return disks
}

// addFilesystemUsage adds filesystem usage information to a disk
// lsblk -no MOUNTPOINTS,FSTYPE DEVICE # get mountpoints and filesystem types for device
// lsblk -no NAME,MOUNTPOINTS,FSTYPE DEVICE # get detailed partition info (fallback)
// df -B1 --output=used,avail MOUNTPOINT # get filesystem usage in bytes
func (l *LsblkTool) addFilesystemUsage(disk *types.DiskInfo) {
	// Get mountpoint and filesystem type using lsblk
	cmd := exec.Command("lsblk", "-no", "MOUNTPOINTS,FSTYPE", disk.Device)
	output, err := cmd.Output()
	if err != nil {
		return
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 0 {
		return
	}

	var mountpoint, filesystem string
	fields := strings.Fields(lines[0])
	if len(fields) >= 1 && fields[0] != "" && fields[0] != "[SWAP]" {
		mountpoint = fields[0]
		if len(fields) >= 2 {
			filesystem = fields[1]
		}
	}

	// If no mountpoint on main device, check partitions
	if mountpoint == "" || mountpoint == "[SWAP]" {
		cmd = exec.Command("lsblk", "-no", "NAME,MOUNTPOINTS,FSTYPE", disk.Device)
		output, err = cmd.Output()
		if err != nil {
			return
		}

		lines = strings.Split(strings.TrimSpace(string(output)), "\n")
		for _, line := range lines {
			if strings.TrimSpace(line) == "" {
				continue
			}
			fields = strings.Fields(line)
			if len(fields) >= 2 && fields[1] != "" && fields[1] != "[SWAP]" {
				mountpoint = fields[1]
				if len(fields) >= 3 {
					filesystem = fields[2]
				}
				break
			}
		}
	}

	disk.Mountpoint = mountpoint
	disk.Filesystem = filesystem

	// If device is mounted, get usage stats using df
	if mountpoint != "" && mountpoint != "[SWAP]" {
		cmd = exec.Command("df", "-B1", "--output=used,avail", mountpoint)
		output, err = cmd.Output()
		if err != nil {
			return
		}

		lines = strings.Split(strings.TrimSpace(string(output)), "\n")
		if len(lines) >= 2 {
			fields = strings.Fields(lines[1])
			if len(fields) >= 2 {
				if used, err := strconv.ParseInt(fields[0], 10, 64); err == nil {
					if avail, err := strconv.ParseInt(fields[1], 10, 64); err == nil {
						disk.UsedBytes = used
						disk.AvailableBytes = avail
						total := used + avail
						if total > 0 {
							disk.UsagePercentage = float64(used) / float64(total) * 100
						}
					}
				}
			}
		}
	}
}
