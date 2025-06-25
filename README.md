# Disk Health Prometheus Exporter

This is a custom Prometheus exporter that monitors disk health on Ubuntu servers. It supports both RAID arrays (using MegaCLI) and regular disks (using smartctl).

## Features

- **RAID Array Monitoring**: Uses MegaCLI to monitor RAID arrays and physical disks
- **Regular Disk Monitoring**: Uses smartctl for non-RAID disks
- **Health Status**: Reports disk health status with appropriate severity levels
- **Temperature Monitoring**: Collects disk temperature data when available
- **Read-Only Operation**: Only reads system information, never modifies anything

## Metrics Exported

### Disk Health Metrics

- `disk_health_status`: Disk health status with labels: device, type, serial, model, location
- `disk_temperature_celsius`: Disk temperature in Celsius with labels: device, serial, model
- `disk_sector_errors_total`: Total number of disk sector errors with labels: device, serial, model, error_type
- `disk_power_on_hours_total`: Total power-on hours for the disk with labels: device, serial, model

### RAID Metrics

- `raid_array_status`: RAID array status with labels: array_id, raid_level, state

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
# Install MegaCLI
sudo apt-get update
sudo apt-get install megacli
```

### For regular disk monitoring

```bash
# Install smartmontools
sudo apt-get update
sudo apt-get install smartmontools
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

```
# HELP disk_health_status Disk health status (0=unknown, 1=ok, 2=warning, 3=critical)
# TYPE disk_health_status gauge
disk_health_status{device="/dev/sda",type="regular",serial="WD-12345",model="WD Blue",location=""} 1
disk_health_status{device="0",type="raid",serial="",model="SEAGATE ST1000",location="Enc:32 Slot:0"} 1

# HELP disk_temperature_celsius Disk temperature in Celsius
# TYPE disk_temperature_celsius gauge
disk_temperature_celsius{device="/dev/sda",serial="WD-12345",model="WD Blue"} 35

# HELP raid_array_status RAID array status (0=unknown, 1=ok, 2=degraded, 3=failed)
# TYPE raid_array_status gauge
raid_array_status{array_id="0",raid_level="Primary-1, Secondary-0, RAID Level Qualifier-0",state="Optimal"} 1
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
