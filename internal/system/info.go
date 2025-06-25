package system

import (
	"log"
	"os/exec"
	"runtime"
)

// SystemInfo holds detected system information
type SystemInfo struct {
	OS           string
	HasSmartctl  bool
	HasMegaCLI   bool
	HasMegaCli64 bool
	SmartctlPath string
	MegaCLIPath  string
	SupportsRAID bool
	Platform     Platform
}

// Platform represents the detected platform type
type Platform string

const (
	PlatformLinux   Platform = "linux"
	PlatformMacOS   Platform = "macos"
	PlatformUnknown Platform = "unknown"
)

// Detector handles system detection
type Detector struct {
	info *SystemInfo
}

// New creates a new system detector
func New() *Detector {
	return &Detector{}
}

// Detect performs one-time system detection
func (d *Detector) Detect() *SystemInfo {
	if d.info != nil {
		return d.info // Return cached info if already detected
	}

	log.Println("Performing one-time system detection...")

	info := &SystemInfo{
		OS: runtime.GOOS,
	}

	// Determine platform
	switch info.OS {
	case "linux":
		info.Platform = PlatformLinux
	case "darwin":
		info.Platform = PlatformMacOS
	default:
		info.Platform = PlatformUnknown
	}

	log.Printf("Detected OS: %s", info.OS)

	// Detect smartctl
	info.detectSmartctl()

	// Detect MegaCLI (mainly for Linux)
	if info.Platform == PlatformLinux {
		info.detectMegaCLI()
	} else {
		log.Printf("Skipping MegaCLI detection on %s", info.OS)
	}

	// Determine RAID support
	info.SupportsRAID = info.HasMegaCLI || info.HasMegaCli64

	// Log detected capabilities
	d.logDetectedCapabilities(info)

	// Cache the info
	d.info = info
	return info
}

// GetInfo returns the cached system info (must call Detect first)
func (d *Detector) GetInfo() *SystemInfo {
	if d.info == nil {
		log.Println("Warning: GetInfo called before Detect()")
		return d.Detect()
	}
	return d.info
}

// detectSmartctl detects smartctl availability
func (info *SystemInfo) detectSmartctl() {
	path, err := exec.LookPath("smartctl")
	if err == nil {
		info.HasSmartctl = true
		info.SmartctlPath = path
		log.Printf("✓ smartctl found at: %s", path)
	} else {
		info.HasSmartctl = false
		log.Printf("✗ smartctl not found")
	}
}

// detectMegaCLI detects MegaCLI availability
func (info *SystemInfo) detectMegaCLI() {
	// Check for megacli
	if path, err := exec.LookPath("megacli"); err == nil {
		info.HasMegaCLI = true
		info.MegaCLIPath = path
		log.Printf("✓ MegaCLI found at: %s", path)
		return
	}

	// Check for MegaCli64
	if path, err := exec.LookPath("MegaCli64"); err == nil {
		info.HasMegaCli64 = true
		info.MegaCLIPath = path
		log.Printf("✓ MegaCli64 found at: %s", path)
		return
	}

	log.Printf("✗ MegaCLI not found (RAID monitoring disabled)")
}

// logDetectedCapabilities logs the detected system capabilities
func (d *Detector) logDetectedCapabilities(info *SystemInfo) {
	log.Println("=== System Detection Summary ===")
	log.Printf("Platform: %s", info.Platform)
	log.Printf("OS: %s", info.OS)

	if info.HasSmartctl {
		log.Printf("SMART Support: ✓ (via smartctl)")
	} else {
		log.Printf("SMART Support: ✗")
	}

	if info.SupportsRAID {
		if info.HasMegaCLI {
			log.Printf("RAID Support: ✓ (via megacli)")
		} else if info.HasMegaCli64 {
			log.Printf("RAID Support: ✓ (via MegaCli64)")
		}
	} else {
		log.Printf("RAID Support: ✗")
	}

	log.Println("===============================")
}

// IsLinux returns true if running on Linux
func (info *SystemInfo) IsLinux() bool {
	return info.Platform == PlatformLinux
}

// IsMacOS returns true if running on macOS
func (info *SystemInfo) IsMacOS() bool {
	return info.Platform == PlatformMacOS
}

// CanMonitorSMART returns true if SMART monitoring is available
func (info *SystemInfo) CanMonitorSMART() bool {
	return info.HasSmartctl
}

// CanMonitorRAID returns true if RAID monitoring is available
func (info *SystemInfo) CanMonitorRAID() bool {
	return info.SupportsRAID
}
