package types

// HealthResponse represents the JSON health response
type HealthResponse struct {
	Status      string       `json:"status"`
	Service     string       `json:"service"`
	Version     string       `json:"version"`
	Timestamp   string       `json:"timestamp"`
	SystemInfo  SystemInfo   `json:"system_info"`
	DiskSummary DiskSummary  `json:"disk_summary"`
	Disks       []DiskHealth `json:"disks"`
	RAIDArrays  []RAIDHealth `json:"raid_arrays,omitempty"`
}

// SystemInfo represents system information in JSON
type SystemInfo struct {
	Platform     string `json:"platform"`
	OS           string `json:"os"`
	SmartSupport bool   `json:"smart_support"`
	RAIDSupport  bool   `json:"raid_support"`
	SmartctlPath string `json:"smartctl_path,omitempty"`
	MegaCLIPath  string `json:"megacli_path,omitempty"`
}

// DiskSummary provides a summary of disk health
type DiskSummary struct {
	TotalDisks    int `json:"total_disks"`
	HealthyDisks  int `json:"healthy_disks"`
	WarningDisks  int `json:"warning_disks"`
	CriticalDisks int `json:"critical_disks"`
	UnknownDisks  int `json:"unknown_disks"`
}

// DiskHealth represents individual disk health in JSON
type DiskHealth struct {
	Device       string  `json:"device"`
	Serial       string  `json:"serial"`
	Model        string  `json:"model"`
	Type         string  `json:"type"`
	Location     string  `json:"location"`
	Health       string  `json:"health"`
	HealthCode   int     `json:"health_code"`
	Temperature  float64 `json:"temperature,omitempty"`
	SectorErrors int     `json:"sector_errors"`
}

// RAIDHealth represents RAID array health in JSON
type RAIDHealth struct {
	ArrayID    string `json:"array_id"`
	RaidLevel  string `json:"raid_level"`
	State      string `json:"state"`
	Status     string `json:"status"`
	StatusCode int    `json:"status_code"`
}
