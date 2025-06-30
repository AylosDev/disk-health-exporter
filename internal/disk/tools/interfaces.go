package tools

import "disk-health-exporter/pkg/types"

// ToolInterface defines the common interface for all CLI tools
type ToolInterface interface {
	// IsAvailable checks if the tool is available on the system
	IsAvailable() bool

	// GetVersion returns the tool version
	GetVersion() string

	// GetName returns the tool name
	GetName() string
}

// DiskToolInterface defines the interface for disk detection tools
type DiskToolInterface interface {
	ToolInterface

	// GetDisks returns disk information detected by this tool
	GetDisks() []types.DiskInfo
}

// RAIDToolInterface defines the interface for RAID detection tools
type RAIDToolInterface interface {
	ToolInterface

	// GetRAIDArrays returns RAID array information detected by this tool
	GetRAIDArrays() []types.RAIDInfo

	// GetRAIDDisks returns disk information from RAID arrays
	GetRAIDDisks() []types.DiskInfo
}

// CombinedToolInterface defines the interface for tools that can detect both disks and RAID
type CombinedToolInterface interface {
	DiskToolInterface
	RAIDToolInterface
}

// BatteryToolInterface defines the interface for RAID battery monitoring tools
type BatteryToolInterface interface {
	ToolInterface

	// GetBatteryInfo returns battery information for a specific adapter
	GetBatteryInfo(adapterID string) *types.RAIDBatteryInfo
}

// SoftwareRAIDToolInterface defines the interface for software RAID tools
type SoftwareRAIDToolInterface interface {
	ToolInterface

	// GetSoftwareRAIDs returns software RAID information
	GetSoftwareRAIDs() []types.SoftwareRAIDInfo
}
