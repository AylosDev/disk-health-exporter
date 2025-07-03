package tools

import (
	"log"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"disk-health-exporter/internal/utils"
	"disk-health-exporter/pkg/types"
)

// MdadmTool represents the mdadm CLI tool for software RAID
type MdadmTool struct{}

// NewMdadmTool creates a new MdadmTool instance
func NewMdadmTool() *MdadmTool {
	return &MdadmTool{}
}

// IsAvailable checks if mdadm is available on the system
func (m *MdadmTool) IsAvailable() bool {
	return utils.CommandExists("mdadm")
}

// GetVersion returns the mdadm version
func (m *MdadmTool) GetVersion() string {
	if !m.IsAvailable() {
		return ""
	}

	version, err := utils.GetToolVersion("mdadm", "--version")
	if err != nil {
		return "unknown"
	}
	return version
}

// GetName returns the tool name
func (m *MdadmTool) GetName() string {
	return "mdadm"
}

// GetSoftwareRAIDs returns software RAID information detected by mdadm
func (m *MdadmTool) GetSoftwareRAIDs() []types.SoftwareRAIDInfo {
	var softwareRAIDs []types.SoftwareRAIDInfo

	if !m.IsAvailable() {
		return softwareRAIDs
	}

	log.Printf("Detecting software RAID arrays using mdadm...")

	// Get list of RAID devices from /proc/mdstat
	mdstat, err := os.ReadFile("/proc/mdstat")
	if err != nil {
		log.Printf("Error reading /proc/mdstat: %v", err)
		return softwareRAIDs
	}

	lines := strings.Split(string(mdstat), "\n")
	var currentRAID types.SoftwareRAIDInfo
	var inRAIDBlock bool

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Check for RAID device line (e.g., "md0 : active raid1 sdb1[1] sda1[0]")
		if strings.HasPrefix(line, "md") && strings.Contains(line, " : ") {
			// Save previous RAID if exists
			if inRAIDBlock && currentRAID.Device != "" {
				m.enrichSoftwareRAIDInfo(&currentRAID)
				softwareRAIDs = append(softwareRAIDs, currentRAID)
			}

			// Parse new RAID device
			currentRAID = types.SoftwareRAIDInfo{}
			inRAIDBlock = true

			parts := strings.Fields(line)
			if len(parts) >= 4 {
				currentRAID.Device = "/dev/" + parts[0]
				currentRAID.State = parts[2] // active, inactive, etc.
				currentRAID.Level = parts[3] // raid0, raid1, etc.

				// Extract device list
				for i := 4; i < len(parts); i++ {
					device := parts[i]
					// Remove bracketed information [0], [1], etc.
					deviceName := regexp.MustCompile(`\[[^\]]*\]`).ReplaceAllString(device, "")
					deviceName = "/dev/" + deviceName

					if strings.Contains(device, "(F)") {
						currentRAID.FailedDevices = append(currentRAID.FailedDevices, deviceName)
					} else if strings.Contains(device, "(S)") {
						currentRAID.SpareDevices = append(currentRAID.SpareDevices, deviceName)
					} else {
						currentRAID.ActiveDevices = append(currentRAID.ActiveDevices, deviceName)
					}
				}
			}
		} else if inRAIDBlock && strings.Contains(line, " blocks ") {
			// Parse size line (e.g., "1000000 blocks super 1.2 [2/2] [UU]")
			parts := strings.Fields(line)
			if len(parts) >= 1 {
				if size, err := strconv.ParseInt(parts[0], 10, 64); err == nil {
					currentRAID.ArraySize = size * 1024 // Convert from KB to bytes
				}
			}

			// Parse RAID configuration [2/2] [UU]
			if len(parts) >= 4 {
				raidConfigRe := regexp.MustCompile(`\[(\d+)/(\d+)\]`)
				matches := raidConfigRe.FindStringSubmatch(line)
				if len(matches) == 3 {
					if total, err := strconv.Atoi(matches[1]); err == nil {
						currentRAID.TotalDevices = total
					}
					if active, err := strconv.Atoi(matches[2]); err == nil {
						currentRAID.RaidDevices = active
					}
				}
			}
		} else if inRAIDBlock && (strings.Contains(line, "resync") || strings.Contains(line, "recover") || strings.Contains(line, "check")) {
			// Parse sync progress line
			currentRAID.SyncAction = m.extractSyncAction(line)
			currentRAID.SyncProgress = m.extractSyncProgress(line)
		} else if line == "" {
			// End of RAID block
			if inRAIDBlock && currentRAID.Device != "" {
				m.enrichSoftwareRAIDInfo(&currentRAID)
				softwareRAIDs = append(softwareRAIDs, currentRAID)
				inRAIDBlock = false
			}
		}
	}

	// Add the last RAID if exists
	if inRAIDBlock && currentRAID.Device != "" {
		m.enrichSoftwareRAIDInfo(&currentRAID)
		softwareRAIDs = append(softwareRAIDs, currentRAID)
	}

	log.Printf("Found %d software RAID arrays using mdadm", len(softwareRAIDs))
	return softwareRAIDs
}

// enrichSoftwareRAIDInfo adds detailed information using mdadm --detail
// mdadm --detail DEVICE # get detailed information about RAID device
func (m *MdadmTool) enrichSoftwareRAIDInfo(raid *types.SoftwareRAIDInfo) {
	output, err := exec.Command("mdadm", "--detail", raid.Device).Output()
	if err != nil {
		log.Printf("Error getting mdadm details for %s: %v", raid.Device, err)
		return
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.Contains(line, "UUID :") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				raid.UUID = strings.TrimSpace(parts[1])
			}
		} else if strings.Contains(line, "Update Time :") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				raid.UpdateTime = strings.TrimSpace(strings.Join(parts[1:], ":"))
			}
		} else if strings.Contains(line, "Persistence :") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				raid.Persistence = strings.TrimSpace(parts[1])
			}
		} else if strings.Contains(line, "Bitmap :") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				raid.Bitmap = strings.TrimSpace(parts[1])
			}
		}
	}
}

// extractSyncAction extracts the sync action from mdstat line
func (m *MdadmTool) extractSyncAction(line string) string {
	if strings.Contains(line, "resync") {
		return "resync"
	} else if strings.Contains(line, "recover") {
		return "recover"
	} else if strings.Contains(line, "check") {
		return "check"
	}
	return ""
}

// extractSyncProgress extracts sync progress percentage from mdstat line
func (m *MdadmTool) extractSyncProgress(line string) float64 {
	// Look for pattern like "resync = 45.2%"
	re := regexp.MustCompile(`(\d+\.?\d*)%`)
	matches := re.FindStringSubmatch(line)
	if len(matches) > 1 {
		if progress, err := strconv.ParseFloat(matches[1], 64); err == nil {
			return progress
		}
	}
	return 0.0
}
