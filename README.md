# Disk Health Prometheus Exporter

This is a custom Prometheus exporter that monitors disk health on Ubuntu servers. It supports both RAID arrays (using MegaCLI) and regular disks (using smartctl).

## Features

- **Multi-Tool RAID Monitoring**: Uses MegaCLI, StorCLI, Arcconf, and mdadm to monitor both hardware and software RAID arrays
- **Comprehensive Disk Monitoring**: Uses smartctl, NVMe CLI, hdparm, and lsblk for complete disk information
- **Enhanced Health Metrics**: Reports detailed disk health with comprehensive SMART data
- **Advanced Temperature Monitoring**: Collects current, minimum, and maximum temperature data
- **Wear Leveling Tracking**: Monitors SSD and NVMe wear leveling and percentage used
- **Error Tracking**: Comprehensive error metrics including reallocated sectors, pending sectors, and media errors
- **Multi-Interface Support**: Supports SATA, NVMe, SAS, and other interfaces
- **Software RAID Support**: Complete mdadm software RAID monitoring with sync progress
- **Tool Detection**: Automatically detects and reports available monitoring tools
- **Read-Only Operation**: Only reads system information, never modifies anything

## Metrics Exported

### Disk Health Metrics

- `disk_health_status`: Disk health status with labels: device, type, serial, model, location, interface
- `disk_temperature_celsius`: Current disk temperature in Celsius with labels: device, serial, model, interface
- `disk_temperature_max_celsius`: Maximum recorded disk temperature in Celsius
- `disk_temperature_min_celsius`: Minimum recorded disk temperature in Celsius
- `disk_capacity_bytes`: Disk capacity in bytes with labels: device, serial, model, interface
- `disk_power_on_hours_total`: Total power-on hours for the disk
- `disk_power_cycles_total`: Total number of power cycles
- `disk_smart_enabled`: Whether SMART is enabled (1=enabled, 0=disabled)
- `disk_smart_healthy`: SMART overall health assessment (1=healthy, 0=unhealthy)

### Disk Error Metrics

- `disk_sector_errors_total`: Total number of disk sector errors with labels: device, serial, model, error_type
- `disk_reallocated_sectors_total`: Total number of reallocated sectors
- `disk_pending_sectors_total`: Total number of pending sectors
- `disk_uncorrectable_errors_total`: Total number of uncorrectable errors
- `disk_media_errors_total`: Total number of media errors (NVMe)
- `disk_error_log_entries_total`: Total number of error log entries

### Disk I/O Metrics

- `disk_data_units_written_total`: Total data units written
- `disk_data_units_read_total`: Total data units read

### SSD/NVMe Specific Metrics

- `disk_wear_leveling_percentage`: SSD wear leveling percentage (0-100)
- `disk_percentage_used`: NVMe percentage used (0-100)
- `disk_available_spare_percentage`: NVMe available spare percentage
- `disk_critical_warning`: NVMe critical warning flags

### Hardware RAID Metrics

- `raid_array_status`: RAID array status with labels: array_id, raid_level, state, type, controller
- `raid_array_size_bytes`: RAID array size in bytes
- `raid_array_used_size_bytes`: RAID array used size in bytes
- `raid_array_drives_total`: Total number of drives in RAID array
- `raid_array_active_drives`: Number of active drives in RAID array
- `raid_array_spare_drives`: Number of spare drives in RAID array
- `raid_array_failed_drives`: Number of failed drives in RAID array
- `raid_array_rebuild_progress_percentage`: RAID array rebuild progress (0-100)
- `raid_array_scrub_progress_percentage`: RAID array scrub progress (0-100)

### Software RAID Metrics

- `software_raid_array_status`: Software RAID array status with labels: device, level, state
- `software_raid_sync_progress_percentage`: Software RAID sync progress (0-100)
- `software_raid_array_size_bytes`: Software RAID array size in bytes

### Tool Availability Metrics

- `disk_monitoring_tool_available`: Whether a disk monitoring tool is available with labels: tool, version

### Exporter Metrics

- `disk_health_exporter_up`: Whether the disk health exporter is up and running

## Health Status Values

- `0`: Unknown status
- `1`: OK/Healthy
- `2`: Warning (e.g., rebuilding)
- `3`: Critical/Failed

## Platform Support

This exporter supports multiple platforms:

### Linux (Ubuntu/Debian/CentOS/Fedora/Arch)

- **RAID Support**: Full MegaCLI integration for hardware RAID arrays
- **Disk Support**: Complete smartctl integration for all disk types
- **Installation**: Use universal `install.sh` script (auto-detects Linux distribution)

### macOS

- **Limited RAID Support**: MegaCLI not typically available
- **Disk Support**: smartctl integration with macOS-specific device handling
- **Installation**: Use universal `install.sh` script (auto-detects macOS)
- **Testing**: Use universal `test.sh` script for development testing

### Prerequisites

#### For Linux (RAID + Regular disks)

```bash
# Install hardware RAID management tools (choose based on your hardware)

# For LSI/Broadcom/Dell RAID controllers:
sudo apt-get update
sudo apt-get install megacli

# For newer Broadcom controllers:
sudo apt-get install storcli

# For Adaptec RAID controllers:
sudo apt-get install arcconf

# For software RAID (mdadm):
sudo apt-get install mdadm

# For ZFS RAID:
sudo apt-get install zfsutils-linux
```

### For comprehensive disk monitoring

```bash
# Essential tools
sudo apt-get update
sudo apt-get install smartmontools  # SMART monitoring
sudo apt-get install util-linux     # lsblk and other utilities
sudo apt-get install hdparm         # ATA/IDE disk parameter utility

# For NVMe-specific monitoring:
sudo apt-get install nvme-cli

# For macOS (using Homebrew):
brew install smartmontools
```

## Project Structure

```
disk-health-exporter/
├── cmd/
│   └── disk-health-exporter/    # Main application entry point
│       └── main.go
├── internal/                    # Private application code
│   ├── collector/              # Metrics collection logic
│   ├── config/                 # Configuration management
│   ├── disk/                   # Disk detection and monitoring
│   └── metrics/                # Prometheus metrics definitions
├── pkg/
│   └── types/                  # Shared types and structs
├── scripts/                    # Installation and utility scripts
│   ├── install.sh              # Universal installation script (detects OS)
│   ├── test.sh                 # Universal testing script (detects OS)
│   └── demo.sh                 # System capabilities demo
├── deployments/                # Deployment configurations
│   ├── Dockerfile              # Docker container definition
│   └── disk-health-exporter.service  # systemd service file
├── go.mod                      # Go module definition
├── go.sum                      # Go module checksums
├── Makefile                    # Build automation
└── README.md                   # This file
```

## Building and Running

### Build from Source

```bash
# Build the binary
make build
# or
go build -o disk-health-exporter ./cmd/disk-health-exporter

# Run directly
./disk-health-exporter
```

### Environment Variables

- `PORT`: HTTP server port (default: 9100)

## Metrics Exposed

- `disk_health_status`: Disk health status with labels (device, type, serial, model, location)
- `disk_temperature_celsius`: Disk temperature in Celsius
- `raid_array_status`: RAID array status with labels (array_id, raid_level, state)
- `disk_health_exporter_up`: Exporter availability status

## Docker Usage

Build the Docker image:

```bash
docker build -t disk-health-exporter .
```

Run with Docker:

```bash
docker run -d \
  --name disk-health-exporter \
  --privileged \
  -p 9100:9100 \
  -v /dev:/dev:ro \
  -v /proc:/proc:ro \
  -v /sys:/sys:ro \
  disk-health-exporter
```

**Note**: The `--privileged` flag and volume mounts are required for the exporter to access disk information.

## Prometheus Configuration

Add this to your `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'disk-health'
    static_configs:
      - targets: ['localhost:9100']
    scrape_interval: 30s
```

## Example Metrics Output

```prometheus
# HELP disk_health_status Disk health status (0=unknown, 1=ok, 2=warning, 3=critical)
# TYPE disk_health_status gauge
disk_health_status{device="/dev/sda",type="regular",serial="WD-12345",model="WD Blue",location="",interface="SATA"} 1
disk_health_status{device="0",type="raid",serial="",model="SEAGATE ST1000",location="Enc:32 Slot:0",interface="SAS"} 1

# HELP disk_temperature_celsius Disk temperature in Celsius
# TYPE disk_temperature_celsius gauge
disk_temperature_celsius{device="/dev/sda",serial="WD-12345",model="WD Blue",interface="SATA"} 35
disk_temperature_celsius{device="/dev/nvme0n1",serial="NVME-67890",model="Samsung SSD 980 PRO",interface="NVMe"} 42

# HELP disk_capacity_bytes Disk capacity in bytes
# TYPE disk_capacity_bytes gauge
disk_capacity_bytes{device="/dev/sda",serial="WD-12345",model="WD Blue",interface="SATA"} 1000204886016
disk_capacity_bytes{device="/dev/nvme0n1",serial="NVME-67890",model="Samsung SSD 980 PRO",interface="NVMe"} 1000204886016

# HELP disk_power_on_hours_total Total power-on hours for the disk
# TYPE disk_power_on_hours_total gauge
disk_power_on_hours_total{device="/dev/sda",serial="WD-12345",model="WD Blue"} 8760
disk_power_on_hours_total{device="/dev/nvme0n1",serial="NVME-67890",model="Samsung SSD 980 PRO"} 4380

# HELP disk_reallocated_sectors_total Total number of reallocated sectors
# TYPE disk_reallocated_sectors_total gauge
disk_reallocated_sectors_total{device="/dev/sda",serial="WD-12345",model="WD Blue"} 0

# HELP disk_percentage_used NVMe percentage used (0-100)
# TYPE disk_percentage_used gauge
disk_percentage_used{device="/dev/nvme0n1",serial="NVME-67890",model="Samsung SSD 980 PRO"} 15

# HELP disk_available_spare_percentage NVMe available spare percentage
# TYPE disk_available_spare_percentage gauge
disk_available_spare_percentage{device="/dev/nvme0n1",serial="NVME-67890",model="Samsung SSD 980 PRO"} 100

# HELP raid_array_status RAID array status (0=unknown, 1=ok, 2=degraded, 3=failed)
# TYPE raid_array_status gauge
raid_array_status{array_id="0",raid_level="Primary-1, Secondary-0, RAID Level Qualifier-0",state="Optimal",type="hardware",controller="MegaCLI"} 1

# HELP software_raid_array_status Software RAID array status (0=unknown, 1=clean, 2=degraded, 3=failed)
# TYPE software_raid_array_status gauge
software_raid_array_status{device="/dev/md0",level="raid1",state="clean"} 1

# HELP disk_monitoring_tool_available Whether a disk monitoring tool is available (1=available, 0=not available)
# TYPE disk_monitoring_tool_available gauge
disk_monitoring_tool_available{tool="smartctl",version="smartmontools 7.2"} 1
disk_monitoring_tool_available{tool="megacli",version="MegaCLI SAS RAID Management Tool Ver 8.07.14"} 1
disk_monitoring_tool_available{tool="mdadm",version="unknown"} 1
```

## Security Considerations

- This exporter runs in read-only mode and does not modify any system settings
- It requires access to disk devices and system information
- Consider running with appropriate user permissions in production
- Use Docker security features like read-only root filesystem when possible

## macOS Usage

### Quick Test

```bash
# Test the exporter on macOS
./test-macos.sh
```

### Manual Run

```bash
# Build and run manually
go build -o disk-health-exporter main.go
./disk-health-exporter
```

### Universal Scripts (Recommended)

The project includes universal scripts that auto-detect your operating system and handle installation and testing automatically:

```bash
# Test the exporter (builds and runs tests)
./scripts/test.sh

# Install as a system service (detects OS automatically)
./scripts/install.sh

# Show system capabilities and demo
./scripts/demo.sh
```

### Install as Service

#### Using Universal Script (Recommended)

```bash
# Automatically detects OS and installs appropriately  
./scripts/install.sh
```

#### Manual Installation

##### Linux Distribution Support

```bash
# Install as systemd service
make install
```

##### macOS Installation

```bash
# Install as LaunchAgent
make install
```

### macOS Notes

- smartctl requires Homebrew installation: `brew install smartmontools`
- RAID monitoring is limited on macOS (no MegaCLI support)
- Some disk information may be limited due to macOS security restrictions
- Disk detection works best with internal drives
