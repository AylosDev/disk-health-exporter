# Disk Health Prometheus Exporter

A comprehensive Prometheus exporter for monitoring disk health, SMART data, and RAID arrays across multiple platforms.

## ⚠️ IMPORTANT DISCLAIMER

**THIS SOFTWARE IS FOR EDUCATIONAL AND RESEARCH PURPOSES ONLY**

- **NOT INTENDED FOR PRODUCTION USE**
- **USE AT YOUR OWN RISK**
- **AUTHORS ARE NOT RESPONSIBLE FOR ANY DAMAGES**

This software performs read-only operations on system resources but can access sensitive hardware information. While designed to be safe, the authors and contributors:

- **DO NOT guarantee** the software's stability or reliability
- **ARE NOT responsible** for any data loss, system damage, or downtime
- **DISCLAIM all liability** for any consequences of using this software
- **RECOMMEND thorough testing** in development environments only

By using this software, you acknowledge these risks and accept full responsibility for any consequences.

## Overview

The Disk Health Exporter monitors:

- **Disk Health**: SMART data, temperature, errors, wear leveling
- **RAID Arrays**: Hardware (MegaCLI, StorCLI, Arcconf) and software (mdadm) RAID
- **Multiple Interfaces**: SATA, NVMe, SAS disk support
- **Cross-Platform**: Linux and macOS support
- **Tool Detection**: Automatic detection and reporting of available monitoring tools

## Quick Start

### Prerequisites

```bash
# Linux - Install monitoring tools
sudo apt-get install smartmontools megacli mdadm

# macOS - Install via Homebrew
brew install smartmontools
```

### Installation

```bash
# Build from source
git clone <repository-url>
cd disk-health-exporter
make build

# Run the exporter
./tmp/bin/disk-health-exporter
```

### Basic Usage

```bash
# Access metrics
curl http://localhost:9100/metrics

# Check tool availability
curl -s http://localhost:9100/metrics | grep tool_available
```

## Key Features

- **30+ Comprehensive Metrics**: Health status, temperature, errors, wear leveling, I/O stats
- **Multi-Tool Support**: smartctl, MegaCLI, StorCLI, Arcconf, mdadm, NVMe CLI
- **Hardware & Software RAID**: Complete RAID monitoring with rebuild progress
- **SSD/NVMe Specific**: Endurance monitoring, wear leveling, critical warnings
- **Tool Detection**: Automatic detection and graceful degradation
- **Read-Only**: Safe monitoring without system modifications

## Documentation

Comprehensive documentation is available in the `docs/` directory:

- **[Installation Guide](docs/installation.md)**: Detailed setup instructions for all platforms
- **[Usage Guide](docs/usage.md)**: Prometheus integration, alerting, and Grafana dashboards
- **[Metrics Reference](docs/metrics.md)**: Complete list of all 30+ metrics with descriptions
- **[Development Guide](docs/development.md)**: Architecture, contributing, and extending the exporter
- **[Enhancements Overview](ENHANCEMENTS.md)**: Detailed overview of recent improvements

## Sample Metrics

```prometheus
# Disk health status
disk_health_status{device="/dev/sda",type="regular",interface="SATA"} 1

# Temperature monitoring
disk_temperature_celsius{device="/dev/sda",interface="SATA"} 35

# NVMe endurance
disk_percentage_used{device="/dev/nvme0n1"} 15

# RAID array status
raid_array_status{array_id="0",type="hardware",controller="MegaCLI"} 1

# Tool availability
disk_monitoring_tool_available{tool="smartctl",version="smartmontools 7.2"} 1
```

## Supported Platforms

### Linux

- **Full support** for all features
- **RAID**: MegaCLI, StorCLI, Arcconf, mdadm
- **Disks**: smartctl, NVMe CLI, hdparm, lsblk

### macOS

- **Limited RAID support** (hardware RAID rare on Mac)
- **Disk support** via smartctl and diskutil
- **Best effort** monitoring with available tools

## Project Structure

```
├── cmd/                    # Main application
├── internal/               # Private application code
│   ├── collector/          # Metrics collection
│   ├── disk/              # Disk detection & monitoring
│   └── metrics/           # Prometheus metrics
├── pkg/types/             # Shared types
├── docs/                  # Documentation
├── scripts/               # Installation scripts
└── deployments/           # Docker & service files
```

## License

This project is licensed under the MIT License with additional disclaimers - see the [LICENSE](LICENSE) file for details.

## Security Notice

- Runs in **read-only mode** - never modifies system settings
- Requires access to disk devices and system information
- Use appropriate permissions and security measures
- Consider container security features in production-like environments

## Support and Contributing

- **Issues**: Report bugs and issues in the GitHub issue tracker
- **Contributing**: See [Development Guide](docs/development.md) for contribution guidelines
- **Documentation**: Comprehensive guides available in `docs/` directory

---

**Remember**: This is educational software. Test thoroughly and use responsibly.
