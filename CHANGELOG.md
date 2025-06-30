# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

### Changed

### Deprecated

### Removed

### Fixed

### Security

## [0.0.8] - 2025-06-30

### Added

- **Complete macOS disk detection** - Fully implemented `getDiskutilDisks` function for macOS systems
  - Uses `diskutil list` and `diskutil info` to detect physical disks
  - Parses disk information including model, capacity, interface, and vendor
  - Correctly identifies and filters physical vs virtual disks (excludes APFS containers)
  - Enhanced SMART data collection with macOS-compatible smartctl parameters
  - Graceful error handling for smartctl failures common on macOS systems

### Changed

- **Performance optimization: Tool availability caching** - Tool detection now occurs only at startup instead of every collection cycle
  - Linux: Cached availability for lsblk, smartctl, nvme, megacli, mdadm, arcconf, storcli, zpool, hdparm
  - macOS: Cached availability for diskutil, smartctl, nvme, zpool  
  - Windows: Cached availability for smartctl, nvme
  - Eliminates 9+ system calls per collection cycle, significantly improving performance
  - Tool availability logged once at startup with detailed detection information

### Removed

- **Tool availability metrics** - Removed `disk_monitoring_tool_available` metrics as they are not relevant for monitoring purposes
  - Removed `ToolAvailable` metric from metrics system
  - Removed `updateToolMetrics()` function from collector
  - Cleaned up metric registration and reset operations
  - Updated all documentation to remove references to tool availability metrics

### Fixed

- **macOS disk detection** - Previously empty implementation now fully functional
  - Physical disk detection using diskutil with proper filtering logic
  - SMART data integration when smartctl is available
  - Proper handling of APFS containers and synthesized volumes
  - Comprehensive disk information extraction (model, capacity, interface, vendor)

### Documentation

- **Updated metrics documentation** - Removed tool availability metrics section
- **Updated usage examples** - Removed tool availability monitoring examples and alerting rules
- **Updated installation guide** - Removed tool detection verification commands
- **Updated README** - Removed tool availability examples from sample metrics
- **Updated troubleshooting** - Replaced tool availability checks with general disk detection guidance

## [0.0.7] - 2025-06-30

### Added

- **Version flag support** - Added `-version` command-line flag to display version information
- **Grafana Dashboard** - Comprehensive JSON dashboard template for visualizing disk health metrics
  - Overview section with health status gauges and distribution charts
  - Temperature monitoring with time-series graphs and threshold alerts
  - RAID array status monitoring including rebuild/scrub progress
  - Disk error tracking with sector error metrics and SSD/NVMe wear indicators
  - Performance metrics showing power-on hours and disk capacity
  - RAID battery monitoring with voltage, temperature, and status indicators
  - System status panels for exporter health and tool availability
  - Color-coded thresholds and responsive design
  - Auto-refresh every 30 seconds with proper units and formatting

### Changed

- **Enhanced configuration system** - Version information now properly propagated from build-time variables
- **Improved help system** - Version and help flags now exit immediately as expected
- **Updated documentation** - Added Grafana dashboard to examples folder in docs

### Fixed

- **Command-line flag parsing** - Version flag now works correctly and exits after displaying information
- **Configuration structure** - Proper version handling throughout the application lifecycle

## [0.0.6] - 2025-06-27

### Added

- **RAID Controller Battery Monitoring** - Comprehensive BBU (Backup Battery Unit) monitoring for hardware RAID controllers
- **New Battery Information Structure** - `RAIDBatteryInfo` type with 20+ fields covering all aspects of battery health and status
- **14 New Battery Metrics** for Prometheus monitoring:
  - `raid_battery_status` - Overall battery health status (0-3)
  - `raid_battery_voltage_millivolts` - Current voltage measurement
  - `raid_battery_current_milliamps` - Current draw measurement
  - `raid_battery_temperature_celsius` - Battery temperature monitoring
  - `raid_battery_missing` - Battery missing detection
  - `raid_battery_replacement_required` - Replacement required indicator
  - `raid_battery_capacity_low` - Low capacity warning
  - `raid_battery_learn_cycle_active` - Learn cycle status monitoring
  - `raid_battery_pack_energy_joules` - Current energy level
  - `raid_battery_capacitance` - Battery capacitance measurement
  - `raid_battery_backup_charge_time_hours` - Available backup time
  - `raid_battery_design_capacity_joules` - Design capacity specification
  - `raid_battery_design_voltage_millivolts` - Design voltage specification
  - `raid_battery_auto_learn_period_days` - Auto learn cycle scheduling
- **MegaCLI Battery Integration** - Automatic battery information collection using `megacli -AdpBbuCmd -aAll`
- **9 Comprehensive Alert Rules** for battery monitoring including critical alerts for missing/failed batteries and warning alerts for capacity/temperature issues
- **Battery Demo Script** - Interactive demonstration script showing new battery monitoring capabilities
- **Complete Documentation** - Updated metrics reference, README, and alert examples with battery monitoring information
- **Unit Tests** - Battery parsing logic tested with real MegaCLI output samples

### Changed

- **Enhanced RAID Detection** - RAID array detection now includes automatic battery information collection when MegaCLI is available
- **Updated Metrics Documentation** - Added comprehensive battery metrics documentation with label descriptions and status value references
- **Enhanced README** - Updated feature list and sample metrics to showcase battery monitoring capabilities

### Fixed

- **Robust Battery Parsing** - Smart parsing with regex for extracting numeric values from MegaCLI text output
- **Error Handling** - Proper error handling for battery data collection with graceful degradation when battery information is unavailable

## [0.0.5] - 2025-06-26

### Added

- Command-line flag support for all configuration options
- Help system with `-help` flag showing usage information and examples
- Environment variable fallback for backwards compatibility
- Priority system where flags override environment variables
- Comprehensive configuration documentation in README.md
- Enhanced test suite for configuration functionality
- Support for both duration strings (30s, 2m) and integer seconds for interval parsing
- **Disk filtering functionality** with `-target-disks` flag for monitoring specific disks
- **Automatic filtering** of loop devices (`/dev/loop*`), RAM disks (`/dev/ram*`), and device mapper devices (`/dev/dm-*`)
- `TARGET_DISKS` environment variable support for disk filtering
- Comprehensive disk filtering documentation in `docs/disk-filtering.md`
- Performance-optimized filtering applied during disk detection rather than post-processing
- Detailed logging for disk inclusion/exclusion decisions

### Changed

- Configuration system now uses command-line flags as primary method instead of environment variables
- Startup message moved after flag parsing to prevent noise with help output
- Updated README.md with comprehensive configuration section including flag documentation
- Flag parsing happens early in startup process for better user experience
- **Enhanced disk manager** to support configuration-based filtering
- **Improved collector** with `NewWithConfig()` constructor for configuration-aware disk detection
- **Updated disk detection methods** on both Linux and macOS to apply filtering during detection

### Fixed

- Improved MegaCLI RAID level parsing to handle complex formats like "Primary-5, Secondary-0, RAID Level Qualifier-3"
- Added proper size parsing for human-readable formats (TB, GB, etc.) in RAID array detection
- Enhanced MegaCLI output parsing to capture additional array information (size, drive count)
- **Major macOS disk detection improvements**:
  - Fixed smartctl scan output parsing bug that incorrectly extracted device types (was getting `-d` instead of actual type like `nvme`)
  - Improved error handling for smartctl exit codes, especially exit code 4 which is common on macOS but still returns valid data
  - Enhanced NVMe drive support with extraction of detailed health metrics (percentage used, available spare, media errors)
  - Added proper ATA/SATA drive support for non-NVMe drives with RPM, form factor, and capacity detection
  - Implemented multi-protocol fallback detection (nvme → auto → ata → scsi) for better compatibility
  - Added structured approach with separate scanning, direct detection, and basic entry creation methods
  - Enhanced logging with more informative messages including temperature readings and health status
  - Better temperature handling using NVMe-specific sensors when available for more accurate readings

## [0.0.4] - 2025-06-26

### Added

- CHANGELOG.md file to track project changes and version history
- Proper semantic versioning documentation following Keep a Changelog format

### Fixed

- Updated the release pipeline to include the new CHANGELOG.md

## [0.0.3] - 2025-06-26

### Changed

- Removed the installation process of the `upx` package from the installation pipeline

## [0.0.2] - 2025-06-26

### Added

- GitHub releases integration for binary distribution
- Version-specific installation support with `-v` flag
- Optional service installation with `-s` flag
- Command-line argument parsing with help system
- Cross-platform architecture detection (amd64/arm64)
- Comprehensive error handling and user feedback
- Support for multiple Linux distributions and macOS variants

### Changed

- Installation script now downloads pre-compiled binaries instead of building from source
- Dependencies are checked and warned about but not automatically installed
- Service installation is now optional and requires explicit `-s` flag
- Improved documentation in README.md and docs/installation.md

### Fixed

- GitHub releases URL format to use proper API endpoints
- Binary download validation and error handling

### Removed

- Automatic dependency installation functionality
- Requirement for Go toolchain during installation

## [0.0.1] - 2025-06-25

### Added

- Initial release of Disk Health Prometheus Exporter
- SMART data monitoring for disks
- RAID array monitoring support
- Cross-platform support (Linux/macOS)
- Prometheus metrics endpoint
- Basic installation script
- Comprehensive documentation
