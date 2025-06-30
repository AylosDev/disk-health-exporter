package tools

import (
	"log"
	"os/exec"
	"strconv"
	"strings"

	"disk-health-exporter/internal/utils"
	"disk-health-exporter/pkg/types"
)

// HdparmTool represents the hdparm CLI tool
type HdparmTool struct{}

// NewHdparmTool creates a new HdparmTool instance
func NewHdparmTool() *HdparmTool {
	return &HdparmTool{}
}

// IsAvailable checks if hdparm is available on the system
func (h *HdparmTool) IsAvailable() bool {
	return utils.CommandExists("hdparm")
}

// GetVersion returns the hdparm version
func (h *HdparmTool) GetVersion() string {
	if !h.IsAvailable() {
		return ""
	}

	version, err := utils.GetToolVersion("hdparm", "-V")
	if err != nil {
		return "unknown"
	}
	return version
}

// GetName returns the tool name
func (h *HdparmTool) GetName() string {
	return "hdparm"
}

// GetDisks returns disk information detected by hdparm
func (h *HdparmTool) GetDisks() []types.DiskInfo {
	var disks []types.DiskInfo

	if !h.IsAvailable() {
		return disks
	}

	log.Printf("Detecting disks using hdparm...")

	// Get list of block devices to check
	blockDevices := h.getBlockDevices()

	for _, device := range blockDevices {
		diskInfo := h.getDiskInfo(device)
		if diskInfo.Device != "" {
			diskInfo.Type = "regular"
			disks = append(disks, diskInfo)
		}
	}

	log.Printf("Found %d disks using hdparm", len(disks))
	return disks
}

// getBlockDevices gets a list of block devices to check
func (h *HdparmTool) getBlockDevices() []string {
	var devices []string

	// Use lsblk to get block devices if available
	output, err := exec.Command("lsblk", "-d", "-n", "-o", "NAME").Output()
	if err != nil {
		// Fallback to common device patterns
		commonDevices := []string{"sda", "sdb", "sdc", "sdd", "hda", "hdb", "hdc", "hdd"}
		for _, dev := range commonDevices {
			devices = append(devices, "/dev/"+dev)
		}
		return devices
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			device := "/dev/" + strings.TrimSpace(line)
			// Only check ATA/IDE/SATA devices (hdparm doesn't work with NVMe)
			if strings.Contains(device, "sd") || strings.Contains(device, "hd") {
				devices = append(devices, device)
			}
		}
	}

	return devices
}

// getDiskInfo gets disk information using hdparm -I
func (h *HdparmTool) getDiskInfo(device string) types.DiskInfo {
	disk := types.DiskInfo{
		Device: device,
	}

	// Use hdparm -I to get detailed ATA information
	output, err := exec.Command("hdparm", "-I", device).Output()
	if err != nil {
		// Device might not support ATA commands or not accessible
		log.Printf("hdparm -I failed for %s: %v", device, err)
		return types.DiskInfo{}
	}

	h.parseHdparmOutput(&disk, string(output))
	return disk
}

// parseHdparmOutput parses hdparm -I output to extract disk information
func (h *HdparmTool) parseHdparmOutput(disk *types.DiskInfo, output string) {
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Parse model number
		if strings.Contains(line, "Model Number:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				disk.Model = strings.TrimSpace(parts[1])
			}
		}

		// Parse serial number
		if strings.Contains(line, "Serial Number:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				disk.Serial = strings.TrimSpace(parts[1])
			}
		}

		// Parse firmware version (information only, not stored in DiskInfo)
		if strings.Contains(line, "Firmware Revision:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				// Firmware version could be logged or used for vendor detection
				firmware := strings.TrimSpace(parts[1])
				log.Printf("Device %s firmware: %s", disk.Device, firmware)
			}
		}

		// Parse transport/interface information
		if strings.Contains(line, "Transport:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				transport := strings.TrimSpace(parts[1])
				disk.Interface = h.parseTransport(transport)
			}
		}

		// Parse SATA version
		if strings.Contains(line, "SATA Version is:") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				sataInfo := strings.TrimSpace(parts[1])
				// Extract SATA generation (e.g., "SATA 3.0" from "SATA 3.0, 6.0 Gb/s")
				if strings.Contains(sataInfo, "SATA") {
					disk.Interface = h.extractSATAVersion(sataInfo)
				}
			}
		}

		// Parse form factor/configuration
		if strings.Contains(line, "Nominal form factor:") || strings.Contains(line, "Form Factor:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				formFactor := strings.TrimSpace(parts[1])
				disk.FormFactor = h.parseFormFactor(formFactor)
			}
		}

		// Parse capacity
		if strings.Contains(line, "device size with M = 1024*1024:") {
			// Extract capacity from line like "device size with M = 1024*1024: 476940 MBytes"
			parts := strings.Fields(line)
			for i, part := range parts {
				if part == "MBytes" && i > 0 {
					if mbytes, err := strconv.ParseInt(parts[i-1], 10, 64); err == nil {
						disk.Capacity = mbytes * 1024 * 1024 // Convert MB to bytes
					}
					break
				}
			}
		}

		// Parse RPM (if rotational)
		if strings.Contains(line, "Nominal rotational rate:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				rpmStr := strings.TrimSpace(parts[1])
				if strings.Contains(rpmStr, "Solid State Device") {
					disk.RPM = 0 // SSD
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
		}

		// Check for SMART support
		if strings.Contains(line, "SMART feature set") {
			if strings.Contains(line, "Enabled") {
				disk.SmartEnabled = true
			}
		}
	}

	// Set default interface if not found
	if disk.Interface == "" {
		disk.Interface = "ATA"
	}

	// Set default health status
	disk.Health = "OK"
	disk.SmartHealthy = true
}

// parseTransport converts transport information to standard interface names
func (h *HdparmTool) parseTransport(transport string) string {
	transport = strings.ToLower(transport)

	if strings.Contains(transport, "sata") {
		return "SATA"
	} else if strings.Contains(transport, "pata") || strings.Contains(transport, "ide") {
		return "PATA"
	} else if strings.Contains(transport, "sas") {
		return "SAS"
	}

	return "ATA"
}

// extractSATAVersion extracts SATA version from string like "SATA 3.0, 6.0 Gb/s"
func (h *HdparmTool) extractSATAVersion(sataInfo string) string {
	// Look for pattern like "SATA 3.0" or "SATA 2.0"
	parts := strings.Fields(sataInfo)
	for i, part := range parts {
		if strings.ToUpper(part) == "SATA" && i+1 < len(parts) {
			version := strings.TrimSuffix(parts[i+1], ",")
			return "SATA " + version
		}
	}
	return "SATA"
}

// parseFormFactor converts form factor information to standard format
func (h *HdparmTool) parseFormFactor(formFactor string) string {
	formFactor = strings.ToLower(formFactor)

	if strings.Contains(formFactor, "3.5") {
		return "3.5\""
	} else if strings.Contains(formFactor, "2.5") {
		return "2.5\""
	} else if strings.Contains(formFactor, "1.8") {
		return "1.8\""
	}

	return formFactor
}
