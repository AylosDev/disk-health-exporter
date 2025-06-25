package health

import (
	"strings"
	"time"

	"disk-health-exporter/internal/collector"
	"disk-health-exporter/internal/system"
	"disk-health-exporter/pkg/types"
)

const (
	serviceVersion = "1.0.0"
	serviceName    = "disk-health-exporter"
)

// Service provides health data collection functionality
type Service struct {
	collector *collector.Collector
	sysInfo   *system.SystemInfo
}

// New creates a new health service
func New(collector *collector.Collector, sysInfo *system.SystemInfo) *Service {
	return &Service{
		collector: collector,
		sysInfo:   sysInfo,
	}
}

// GetHealthData collects current health information for JSON response
func (s *Service) GetHealthData() *types.HealthResponse {
	// Get current disk and RAID data
	disks := s.collector.GetCurrentDisks()
	raidArrays := s.collector.GetCurrentRAIDArrays()

	// Convert disk information
	diskHealthData := make([]types.DiskHealth, len(disks))
	totalDisks := len(disks)
	healthyDisks := 0
	warningDisks := 0
	criticalDisks := 0
	unknownDisks := 0

	for i, disk := range disks {
		healthCode := s.getHealthStatusCode(disk.Health)

		diskHealthData[i] = types.DiskHealth{
			Device:       disk.Device,
			Serial:       disk.Serial,
			Model:        disk.Model,
			Type:         disk.Type,
			Location:     disk.Location,
			Health:       disk.Health,
			HealthCode:   healthCode,
			Temperature:  disk.Temperature,
			SectorErrors: 0, // This would need to be populated from SMART data
		}

		// Count health statuses
		switch healthCode {
		case 1:
			healthyDisks++
		case 2:
			warningDisks++
		case 3:
			criticalDisks++
		default:
			unknownDisks++
		}
	}

	// Convert RAID information
	raidHealthData := make([]types.RAIDHealth, len(raidArrays))
	for i, raid := range raidArrays {
		raidHealthData[i] = types.RAIDHealth{
			ArrayID:    raid.ArrayID,
			RaidLevel:  raid.RaidLevel,
			State:      raid.State,
			Status:     raid.State, // Use State as Status for display
			StatusCode: raid.Status,
		}
	}

	// Build system info
	systemInfo := types.SystemInfo{
		Platform:     string(s.sysInfo.Platform),
		OS:           s.sysInfo.OS,
		SmartSupport: s.sysInfo.CanMonitorSMART(),
		RAIDSupport:  s.sysInfo.CanMonitorRAID(),
	}

	if s.sysInfo.HasSmartctl {
		systemInfo.SmartctlPath = s.sysInfo.SmartctlPath
	}

	if s.sysInfo.HasMegaCLI {
		systemInfo.MegaCLIPath = s.sysInfo.MegaCLIPath
	}

	// Build summary
	diskSummary := types.DiskSummary{
		TotalDisks:    totalDisks,
		HealthyDisks:  healthyDisks,
		WarningDisks:  warningDisks,
		CriticalDisks: criticalDisks,
		UnknownDisks:  unknownDisks,
	}

	// Build response
	response := &types.HealthResponse{
		Status:      "ok",
		Service:     serviceName,
		Version:     serviceVersion,
		Timestamp:   time.Now().Format(time.RFC3339),
		SystemInfo:  systemInfo,
		DiskSummary: diskSummary,
		Disks:       diskHealthData,
		RAIDArrays:  raidHealthData,
	}

	return response
}

// getHealthStatusCode converts health string to numeric code
func (s *Service) getHealthStatusCode(health string) int {
	switch strings.ToUpper(strings.TrimSpace(health)) {
	case "OK":
		return 1
	case "WARNING":
		return 2
	case "FAILED", "CRITICAL":
		return 3
	default:
		return 0
	}
}
