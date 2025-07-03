package tools

import (
	"log"
	"os/exec"
	"strings"

	"disk-health-exporter/internal/utils"
	"disk-health-exporter/pkg/types"
)

// NvmeTool represents the nvme CLI tool
type NvmeTool struct{}

// NewNvmeTool creates a new NvmeTool instance
func NewNvmeTool() *NvmeTool {
	return &NvmeTool{}
}

// IsAvailable checks if nvme CLI is available on the system
func (n *NvmeTool) IsAvailable() bool {
	return utils.CommandExists("nvme")
}

// GetVersion returns the nvme CLI version
func (n *NvmeTool) GetVersion() string {
	if !n.IsAvailable() {
		return ""
	}

	version, err := utils.GetToolVersion("nvme", "version")
	if err != nil {
		return "unknown"
	}
	return version
}

// GetName returns the tool name
func (n *NvmeTool) GetName() string {
	return "nvme"
}

// GetDisks returns NVMe disk information detected by nvme CLI
// nvme list # list all NVMe devices
func (n *NvmeTool) GetDisks() []types.DiskInfo {
	var disks []types.DiskInfo

	if !n.IsAvailable() {
		return disks
	}

	log.Printf("Detecting NVMe disks using nvme CLI...")

	// List NVMe devices
	output, err := exec.Command("nvme", "list").Output()
	if err != nil {
		log.Printf("Error running nvme list: %v", err)
		return disks
	}

	lines := strings.Split(string(output), "\n")
	for i, line := range lines {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue // Skip header and empty lines
		}

		fields := strings.Fields(line)
		if len(fields) >= 1 && strings.Contains(fields[0], "/dev/nvme") {
			device := fields[0]

			// Add basic NVMe info
			diskInfo := types.DiskInfo{
				Device:    device,
				Type:      "nvme",
				Interface: "NVMe",
				Health:    "Unknown",
			}
			if len(fields) >= 3 {
				diskInfo.Model = fields[2]
			}
			disks = append(disks, diskInfo)
		}
	}

	log.Printf("Found %d NVMe disks using nvme CLI", len(disks))
	return disks
}
