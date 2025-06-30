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
	Device              string
	Serial              string
	Model               string
	Vendor              string
	Health              string
	Temperature         float64
	Type                string  // "raid", "regular", "macos-smart", etc.
	Location            string  // physical location or slot
	PowerOnHours        int64   // Total power-on hours
	PowerCycles         int64   // Number of power cycles
	ReallocatedSectors  int64   // Reallocated sectors count
	PendingSectors      int64   // Current pending sectors
	UncorrectableErrors int64   // Uncorrectable error count
	TotalLBAsWritten    int64   // Total LBAs written
	TotalLBAsRead       int64   // Total LBAs read
	DriveTemperatureMax float64 // Maximum recorded temperature
	DriveTemperatureMin float64 // Minimum recorded temperature
	Interface           string  // SATA, NVMe, SAS, etc.
	Capacity            int64   // Disk capacity in bytes
	FormFactor          string  // 2.5", 3.5", M.2, etc.
	RPM                 int     // Rotational speed (0 for SSD/NVMe)
	SmartEnabled        bool    // Whether SMART is enabled
	SmartHealthy        bool    // SMART overall health assessment
	WearLeveling        int     // SSD wear leveling percentage (0-100)
	PercentageUsed      int     // NVMe percentage used
	AvailableSpare      int     // NVMe available spare percentage
	CriticalWarning     int     // NVMe critical warning
	MediaErrors         int64   // NVMe media errors
	ErrorLogEntries     int64   // Number of error log entries
}

// RAIDInfo represents RAID array information
type RAIDInfo struct {
	ArrayID         string
	RaidLevel       string
	State           string
	Status          int
	Size            int64            // Array size in bytes
	UsedSize        int64            // Used space in bytes
	NumDrives       int              // Number of drives in array
	NumActiveDrives int              // Number of active drives
	NumSpareDrives  int              // Number of spare drives
	NumFailedDrives int              // Number of failed drives
	RebuildProgress int              // Rebuild progress percentage (0-100)
	ScrubProgress   int              // Scrub progress percentage (0-100)
	Type            string           // "hardware", "software", "zfs", etc.
	Controller      string           // Controller model/name
	Battery         *RAIDBatteryInfo // Battery information (if available)
}

// SmartCtlOutput represents smartctl JSON output structure
type SmartCtlOutput struct {
	Device struct {
		Name     string `json:"name"`
		InfoName string `json:"info_name"`
		Type     string `json:"type"`
		Protocol string `json:"protocol"`
	} `json:"device"`
	SerialNumber string `json:"serial_number"`
	ModelName    string `json:"model_name"`
	ModelFamily  string `json:"model_family"`
	UserCapacity struct {
		Blocks int64 `json:"blocks"`
		Bytes  int64 `json:"bytes"`
	} `json:"user_capacity"`
	LogicalBlockSize  int `json:"logical_block_size"`
	PhysicalBlockSize int `json:"physical_block_size"`
	RotationRate      int `json:"rotation_rate"`
	FormFactor        struct {
		AtaValue int    `json:"ata_value"`
		Name     string `json:"name"`
	} `json:"form_factor"`
	SmartStatus struct {
		Passed bool `json:"passed"`
	} `json:"smart_status"`
	SmartSupport struct {
		Available bool `json:"available"`
		Enabled   bool `json:"enabled"`
	} `json:"smart_support"`
	Temperature struct {
		Current  int `json:"current"`
		Power    int `json:"power_cycle_min_max"`
		Lifetime int `json:"lifetime_min_max"`
	} `json:"temperature"`
	PowerOnTime struct {
		Hours int `json:"hours"`
	} `json:"power_on_time"`
	PowerCycleCount    int `json:"power_cycle_count"`
	AtaSmartAttributes struct {
		Table []struct {
			ID         int    `json:"id"`
			Name       string `json:"name"`
			Value      int    `json:"value"`
			Worst      int    `json:"worst"`
			Thresh     int    `json:"thresh"`
			WhenFailed string `json:"when_failed"`
			Flags      struct {
				Value         int    `json:"value"`
				String        string `json:"string"`
				Prefailure    bool   `json:"prefailure"`
				UpdatedOnline bool   `json:"updated_online"`
				Performance   bool   `json:"performance"`
				ErrorRate     bool   `json:"error_rate"`
				EventCount    bool   `json:"event_count"`
				AutoKeep      bool   `json:"auto_keep"`
			} `json:"flags"`
			Raw struct {
				Value  int64  `json:"value"`
				String string `json:"string"`
			} `json:"raw"`
		} `json:"table"`
	} `json:"ata_smart_attributes"`
	AtaSmartErrorLog struct {
		Summary struct {
			Revision int `json:"revision"`
			Count    int `json:"count"`
		} `json:"summary"`
	} `json:"ata_smart_error_log"`
	NvmeSmartHealthInformationLog struct {
		CriticalWarning               int   `json:"critical_warning"`
		Temperature                   int   `json:"temperature"`
		AvailableSpare                int   `json:"available_spare"`
		AvailableSpareThreshold       int   `json:"available_spare_threshold"`
		PercentageUsed                int   `json:"percentage_used"`
		DataUnitsRead                 int64 `json:"data_units_read"`
		DataUnitsWritten              int64 `json:"data_units_written"`
		HostReadCommands              int64 `json:"host_read_commands"`
		HostWriteCommands             int64 `json:"host_write_commands"`
		ControllerBusyTime            int64 `json:"controller_busy_time"`
		PowerCycles                   int64 `json:"power_cycles"`
		PowerOnHours                  int64 `json:"power_on_hours"`
		UnsafeShutdowns               int64 `json:"unsafe_shutdowns"`
		MediaErrors                   int64 `json:"media_errors"`
		NumErrLogEntries              int64 `json:"num_err_log_entries"`
		WarningTempTime               int   `json:"warning_temp_time"`
		CriticalCompTime              int   `json:"critical_comp_time"`
		TemperatureSensor1            int   `json:"temperature_sensor_1"`
		TemperatureSensor2            int   `json:"temperature_sensor_2"`
		ThermalManagementT1TransCount int   `json:"thermal_mgmt_t1_trans_count"`
		ThermalManagementT2TransCount int   `json:"thermal_mgmt_t2_trans_count"`
		ThermalManagementT1TotalTime  int   `json:"thermal_mgmt_t1_total_time"`
		ThermalManagementT2TotalTime  int   `json:"thermal_mgmt_t2_total_time"`
	} `json:"nvme_smart_health_information_log"`
}

// ToolInfo represents information about available system tools
type ToolInfo struct {
	SmartCtl        bool // smartctl available
	MegaCLI         bool // MegaCLI available
	Mdadm           bool // mdadm available
	Arcconf         bool // arcconf (Adaptec) available
	Storcli         bool // storcli (Broadcom) available
	Zpool           bool // zpool (ZFS) available
	Diskutil        bool // diskutil (macOS) available
	Nvme            bool // nvme-cli available
	Hdparm          bool // hdparm available
	Lsblk           bool // lsblk available
	SmartCtlVersion string
	MegaCLIVersion  string
	StorCLIVersion  string
	HdparmVersion   string
	ArcconfVersion  string
	ZpoolVersion    string
}

// SoftwareRAIDInfo represents software RAID information
type SoftwareRAIDInfo struct {
	Device        string   // /dev/md0, /dev/md1, etc.
	Level         string   // raid0, raid1, raid5, raid6, raid10
	State         string   // clean, active, degraded, etc.
	ArraySize     int64    // Array size in KB
	UsedDevSize   int64    // Used device size in KB
	RaidDevices   int      // Number of RAID devices
	TotalDevices  int      // Total devices (including spares)
	Persistence   string   // Superblock persistence
	UpdateTime    string   // Last update time
	ActiveDevices []string // List of active devices
	SpareDevices  []string // List of spare devices
	FailedDevices []string // List of failed devices
	SyncAction    string   // Current sync action (resync, recover, etc.)
	SyncProgress  float64  // Sync progress percentage
	Bitmap        string   // Bitmap information
	UUID          string   // Array UUID
}

// RAIDBatteryInfo represents RAID controller battery information
type RAIDBatteryInfo struct {
	AdapterID            int    // Adapter ID
	BatteryType          string // Battery type (e.g., CVPM02)
	Voltage              int    // Voltage in mV
	Current              int    // Current in mA
	Temperature          int    // Temperature in Celsius
	State                string // Battery state (Optimal, Warning, Critical, etc.)
	ChargingStatus       string // Charging status
	VoltageStatus        string // Voltage status (OK, Warning, etc.)
	TemperatureStatus    string // Temperature status
	LearnCycleActive     bool   // Learn cycle active
	LearnCycleStatus     string // Learn cycle status
	BatteryMissing       bool   // Battery pack missing
	ReplacementRequired  bool   // Battery replacement required
	RemainingCapacityLow bool   // Remaining capacity low
	PackEnergy           int    // Pack energy in Joules
	Capacitance          int    // Capacitance
	BackupChargeTime     int    // Battery backup charge time in hours
	ManufactureDate      string // Date of manufacture
	DesignCapacity       int    // Design capacity in Joules
	DesignVoltage        int    // Design voltage in mV
	SerialNumber         string // Serial number
	ManufactureName      string // Manufacturer name
	FirmwareVersion      string // Firmware version
	DeviceName           string // Device name
	DeviceChemistry      string // Device chemistry
	AutoLearnPeriod      int    // Auto learn period in days
	NextLearnTime        string // Next learn time
}
