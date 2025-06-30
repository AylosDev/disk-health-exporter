package disk

import (
	"runtime"
	"strings"

	"disk-health-exporter/internal/disk/systems"
	"disk-health-exporter/pkg/types"
)

// SystemInterface defines the interface for system-specific disk detection
type SystemInterface interface {
	GetDisks() ([]types.DiskInfo, []types.RAIDInfo)
	GetSystemType() string
	GetToolInfo() types.ToolInfo
}

// Manager handles disk detection and monitoring
type Manager struct {
	targetDisks    []string        // Specific disks to monitor (empty = all)
	ignorePatterns []string        // Patterns to ignore
	systemImpl     SystemInterface // System-specific implementation
}

// New creates a new disk manager with default settings
func New() *Manager {
	return NewWithConfig("", []string{})
}

// NewWithConfig creates a new disk manager with specific configuration
func NewWithConfig(targetDisks string, ignorePatterns []string) *Manager {
	m := &Manager{
		ignorePatterns: ignorePatterns,
	}

	// Parse target disks
	if targetDisks != "" {
		m.targetDisks = strings.Split(strings.ReplaceAll(targetDisks, " ", ""), ",")
		// Filter out empty strings
		var filtered []string
		for _, disk := range m.targetDisks {
			if disk != "" {
				filtered = append(filtered, disk)
			}
		}
		m.targetDisks = filtered
	}

	// Set system implementation
	m.systemImpl = createSystemImplementation(m.targetDisks, m.ignorePatterns)
	return m
}

// GetDisks returns all detected disks and RAID arrays
func (m *Manager) GetDisks() ([]types.DiskInfo, []types.RAIDInfo) {
	return m.systemImpl.GetDisks()
}

// GetSystemType returns the current system type
func (m *Manager) GetSystemType() string {
	return m.systemImpl.GetSystemType()
}

// GetToolInfo returns information about available tools
func (m *Manager) GetToolInfo() types.ToolInfo {
	return m.systemImpl.GetToolInfo()
}

// createSystemImplementation creates the appropriate system implementation
func createSystemImplementation(targetDisks []string, ignorePatterns []string) SystemInterface {
	switch runtime.GOOS {
	case "linux":
		return systems.NewLinuxSystem(targetDisks, ignorePatterns)
	case "darwin":
		return systems.NewMacOSSystem(targetDisks, ignorePatterns)
	case "windows":
		return systems.NewWindowsSystem(targetDisks, ignorePatterns)
	default:
		// Default to Linux implementation for unknown systems
		return systems.NewLinuxSystem(targetDisks, ignorePatterns)
	}
}
