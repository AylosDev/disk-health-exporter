# Installation Guide

This guide covers installation and setup of the Disk Health Exporter on various platforms.

## Prerequisites

### Linux Systems

#### Hardware RAID Management Tools

Choose the appropriate tool based on your RAID controller:

```bash
# For LSI/Broadcom/Dell RAID controllers
sudo apt-get update
sudo apt-get install megacli

# For newer Broadcom controllers
sudo apt-get install storcli

# For Adaptec RAID controllers
sudo apt-get install arcconf

# For software RAID (mdadm)
sudo apt-get install mdadm

# For ZFS RAID
sudo apt-get install zfsutils-linux
```

#### Essential Disk Monitoring Tools

```bash
# Essential tools for comprehensive monitoring
sudo apt-get update
sudo apt-get install smartmontools  # SMART monitoring
sudo apt-get install util-linux     # lsblk and other utilities
sudo apt-get install hdparm         # ATA/IDE disk parameter utility

# For NVMe-specific monitoring
sudo apt-get install nvme-cli
```

### macOS Systems

```bash
# Install smartmontools using Homebrew
brew install smartmontools

# Optional: Install additional tools if available
brew install nvme-cli  # May not be available on all macOS versions
```

## Installation Methods

### Method 1: Universal Installation Script (Recommended)

The project includes a universal installation script that downloads binaries from GitHub releases and automatically installs them:

#### Quick Installation

```bash
# Install latest version (binary only)
curl -sSL https://raw.githubusercontent.com/AylosDev/disk-health-exporter/main/scripts/install.sh | bash

# Install with systemd/launchd service
curl -sSL https://raw.githubusercontent.com/AylosDev/disk-health-exporter/main/scripts/install.sh | bash -s -- -s
```

#### Advanced Installation Options

```bash
# Download script for local execution
wget https://raw.githubusercontent.com/AylosDev/disk-health-exporter/main/scripts/install.sh
chmod +x install.sh

# View help and options
./install.sh --help

# Install specific version
./install.sh -v v1.0.0

# Install with service (systemd on Linux, launchd on macOS)
./install.sh -s

# Install specific version with service
./install.sh -v v1.0.0 -s
```

#### What the Script Does

- **OS Detection**: Automatically detects Linux/macOS and architecture (amd64/arm64)
- **GitHub Releases**: Downloads pre-compiled binaries from GitHub releases
- **Version Management**: Installs latest version or specific version if specified
- **Binary Installation**: Installs binary to `/usr/local/bin/` with proper permissions
- **Optional Service Setup**:
  - Linux: Creates systemd service with prometheus user
  - macOS: Creates LaunchAgent for current user
- **Dependency Checking**: Warns about missing tools but continues installation

**Important**: The script does NOT automatically install system dependencies like smartmontools. You must install monitoring tools manually before running the exporter for full functionality.

#### Supported Platforms

- **Linux**: Ubuntu, Debian, CentOS, RHEL, Fedora, Arch Linux
- **macOS**: Intel and Apple Silicon
- **Architectures**: amd64 (x86_64) and arm64 (aarch64)

### Method 2: Build from Source

#### Build the Binary

```bash
# Clone the repository
git clone <repository-url>
cd disk-health-exporter

# Build using Make
make build

# Or build directly with Go
go build -o disk-health-exporter ./cmd/disk-health-exporter
```

#### Manual Service Installation

##### Linux (systemd)

```bash
# Copy binary to system location
sudo cp disk-health-exporter /usr/local/bin/

# Copy service file
sudo cp deployments/disk-health-exporter.service /etc/systemd/system/

# Enable and start service
sudo systemctl enable disk-health-exporter
sudo systemctl start disk-health-exporter

# Check status
sudo systemctl status disk-health-exporter
```

##### macOS (LaunchAgent)

```bash
# Copy binary to system location
sudo cp disk-health-exporter /usr/local/bin/

# Create LaunchAgent plist (create this file as needed)
# Copy to ~/Library/LaunchAgents/ or /Library/LaunchAgents/

# Load and start service
launchctl load ~/Library/LaunchAgents/com.diskhealth.exporter.plist
launchctl start com.diskhealth.exporter
```

### Method 3: Docker Installation

#### Build Docker Image

```bash
# Build the Docker image
docker build -t disk-health-exporter .
```

#### Run with Docker

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

**Important**: The `--privileged` flag and volume mounts are required for the exporter to access disk information.

#### Docker Compose

```yaml
version: '3.8'
services:
  disk-health-exporter:
    build: .
    privileged: true
    ports:
      - "9100:9100"
    volumes:
      - /dev:/dev:ro
      - /proc:/proc:ro
      - /sys:/sys:ro
    restart: unless-stopped
```

## Configuration

### Environment Variables

- **`PORT`**: HTTP server port (default: 9100)

```bash
# Set custom port
export PORT=9101
./disk-health-exporter
```

### Service Configuration Files

#### systemd Service File

```ini
[Unit]
Description=Disk Health Prometheus Exporter
After=network.target

[Service]
Type=simple
User=nobody
ExecStart=/usr/local/bin/disk-health-exporter
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

## Verification

### Check Service Status

```bash
# Linux (systemd)
sudo systemctl status disk-health-exporter

# macOS (launchctl)
launchctl list | grep diskhealth
```

### Test Metrics Endpoint

```bash
# Check if exporter is responding
curl http://localhost:9100/metrics

# Check specific metrics
curl -s http://localhost:9100/metrics | grep disk_health_status
```

## Platform-Specific Notes

### Linux

- Requires root or appropriate permissions to access disk devices
- Works with most Linux distributions (Ubuntu, Debian, CentOS, Fedora, Arch)
- Full hardware and software RAID support
- Complete SMART data access

### macOS

- Limited RAID support (hardware RAID controllers rare on macOS)
- Some disk information may be restricted due to macOS security
- Works best with internal drives
- smartctl required for meaningful data

### Security Considerations

- The exporter runs in **read-only mode** and never modifies system settings
- Requires access to disk devices and system information
- Consider running with minimal required permissions in production
- Use Docker security features like read-only root filesystem when possible

## Troubleshooting

### Common Issues

#### No Metrics Appearing

1. Check if required tools are installed:

   ```bash
   which smartctl
   which megacli  # or MegaCli64
   ```

2. Verify permissions:

   ```bash
   # Test manual smartctl access
   sudo smartctl --scan
   ```

#### Service Won't Start

1. Check service logs:

   ```bash
   # Linux
   sudo journalctl -u disk-health-exporter -f
   
   # macOS
   tail -f /var/log/system.log | grep diskhealth
   ```

2. Test manual startup:

   ```bash
   ./disk-health-exporter
   ```

#### Docker Issues

1. Ensure privileged mode and volume mounts are correct
2. Check container logs:

   ```bash
   docker logs disk-health-exporter
   ```

### Getting Help

1. Check the logs for error messages
2. Verify tool availability using the tool detection metrics
3. Test individual tools manually (smartctl, megacli, etc.)
4. Ensure proper permissions for disk access
