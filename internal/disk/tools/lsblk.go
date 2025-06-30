package tools

import (
	"log"
	"os/exec"
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

			disks = append(disks, disk)
		}
	}

	log.Printf("Found %d disks using lsblk", len(disks))
	return disks
}
