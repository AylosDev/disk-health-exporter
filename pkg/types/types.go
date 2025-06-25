package types

// HealthStatus represents disk health status values
type HealthStatus int

const (
	HealthStatusUnknown  HealthStatus = 0
	HealthStatusOK       HealthStatus = 1
	HealthStatusWarning  HealthStatus = 2
	HealthStatusCritical HealthStatus = 3
)

// DiskInfo represents information about a disk
type DiskInfo struct {
	Device      string
	Serial      string
	Model       string
	Health      string
	Temperature float64
	Type        string // "raid", "regular", "macos-smart", etc.
	Location    string // physical location or slot
}

// RAIDInfo represents RAID array information
type RAIDInfo struct {
	ArrayID   string
	RaidLevel string
	State     string
	Status    int
}

// SmartCtlOutput represents smartctl JSON output structure
type SmartCtlOutput struct {
	Device struct {
		Name string `json:"name"`
	} `json:"device"`
	SerialNumber string `json:"serial_number"`
	ModelName    string `json:"model_name"`
	SmartStatus  struct {
		Passed bool `json:"passed"`
	} `json:"smart_status"`
	Temperature struct {
		Current int `json:"current"`
	} `json:"temperature"`
	AtaSmartAttributes struct {
		Table []struct {
			ID     int    `json:"id"`
			Name   string `json:"name"`
			Value  int    `json:"value"`
			Worst  int    `json:"worst"`
			Thresh int    `json:"thresh"`
			Raw    struct {
				Value  int    `json:"value"`
				String string `json:"string"`
			} `json:"raw"`
		} `json:"table"`
	} `json:"ata_smart_attributes"`
	PowerOnTime struct {
		Hours int `json:"hours"`
	} `json:"power_on_time"`
}
