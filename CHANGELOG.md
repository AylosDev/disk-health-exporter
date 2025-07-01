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

## [0.0.11] - 2025-07-01

### Added

- **Inventory and discovery metrics** - Added comprehensive system overview metrics for better disk enumeration
  - `disk_info` - Complete disk inventory with all metadata (device, type, serial, model, vendor, interface, location, RPM, capacity)
  - `disk_present` - Simple disk presence indicator for monitoring disk additions/removals
  - `system_total_disks` - Total count of disks detected in the system
  - `system_total_raid_arrays` - Total count of RAID arrays detected in the system
  - `system_monitoring_tools_available` - Availability and versions of monitoring tools

- **macOS RAID support enhancement** - Added appropriate RAID tool support for macOS systems
  - **ZFS support** - OpenZFS integration for macOS systems that have ZFS installed
  - **Hardware RAID exclusion** - Correctly excluded enterprise RAID controllers (MegaCLI, StoreCLI, Arcconf) that don't run on Darwin/macOS
  - **Realistic tool selection** - Limited to tools that actually exist and make sense on macOS: diskutil, smartctl, nvme, zpool
  - **Enhanced disk merging** - Added `mergeDisks()` method for consistent disk information consolidation

### Fixed

- **Critical RAID disk detection issue** - Fixed major gap where individual physical disks in hardware RAID arrays were not being detected
  - **MegaCLI integration** - Now calls both `GetRAIDArrays()` and `GetRAIDDisks()` to enumerate individual disks within RAID arrays
  - **Complete RAID disk visibility** - Tools like `lsblk` only see RAID logical volumes, but hardware RAID tools can access the actual physical disks
  - **Enhanced disk merging** - Improved `mergeDisks()` method to preserve RAID-specific metadata (Type, Location, Interface)
  - **Consistent pattern** - All RAID tools (MegaCLI, StoreCLI, Arcconf, ZFS) now use `mergeDisks()` instead of `append()` to prevent duplicates
  - **Cross-platform fix** - Applied same disk detection improvements to Linux and macOS systems

- **Disk information merging** - Enhanced disk metadata preservation during information consolidation
  - **RAID metadata preservation** - Fixed merging to retain RAID-specific fields like disk location (e.g., "Enc:1 Slot:2")
  - **Interface information** - Preserved disk interface details from specialized tools
  - **Type classification** - Maintained disk type information (regular, raid, nvme, etc.) during merging

### Changed

- **RAID tool interface unification** - Major refactoring to simplify and consolidate RAID tool interfaces
  - **Unified GetRAIDDisks() method** - Merged `GetRAIDDisksWithUtilization()` functionality into the main `GetRAIDDisks()` method for both StoreCLI and MegaCLI tools
  - **Removed redundant methods** - Eliminated duplicate `GetRAIDDisksWithUtilization()` methods from all RAID tools to reduce code complexity
  - **Enhanced utilization calculation** - Integrated per-disk space utilization and RAID role detection directly into the main disk enumeration methods
  - **Consistent interface compliance** - All RAID tools now use the same unified `RAIDToolInterface` with consistent method signatures
  - **Improved parsing logic** - Enhanced StoreCLI and MegaCLI parsing to handle both summary table and detailed per-drive output formats

- **RAID role detection improvements** - Enhanced detection and classification of disk roles within RAID arrays
  - **Comprehensive spare detection** - Improved detection of hot spares, commissioned spares, emergency spares, and global spares
  - **Unconfigured disk handling** - Better identification and reporting of unconfigured drives available for RAID configuration
  - **Failed disk detection** - Enhanced parsing to accurately identify and report failed drives within RAID arrays
  - **Real-world output compatibility** - Updated parsing logic to match actual StoreCLI and MegaCLI command output formats

- **Code architecture improvements** - Refactored RAID tool implementations for better maintainability and consistency
  - **Fallback utilization helpers** - Added `calculateBasicUtilization()` and `calculateBasicMegaCLIUtilization()` helpers for graceful degradation
  - **Unified method calls** - Updated `GetSpareDisks()` and `GetUnconfiguredDisks()` to use the consolidated `GetRAIDDisks()` method
  - **Cross-platform consistency** - Applied interface changes consistently across Linux, macOS, and Windows implementations
  - **Test suite compatibility** - Maintained all existing test coverage while updating interface usage

- **Architecture clarification** - Added comprehensive comments explaining RAID disk detection logic
  - **Hardware RAID explanation** - Documented why hardware RAID controllers hide disks from OS tools
  - **Tool necessity** - Explained why specialized RAID tools are required to access individual physical disks
  - **Detection strategy** - Clarified the multi-layered approach: OS tools + RAID tools + software storage tools

### Platform Support

- **Linux** - Complete RAID disk detection with all hardware and software RAID tools
- **macOS** - Appropriate tool selection with ZFS support, excluding non-applicable enterprise RAID controllers
- **Windows** - Maintained existing limited implementation (unchanged as requested)

## [0.0.9] - 2025-06-30

### Added

- **Complete StoreCLI support** - Implemented comprehensive StoreCLI (Broadcom) RAID tool integration
  - Supports both JSON and plain text parsing modes for maximum compatibility
  - Detects RAID arrays with full virtual drive information (size, RAID level, status)
  - Detects individual physical disks within RAID arrays with detailed metadata
  - Auto-detection of `storcli64` and `storcli` commands
  - Comprehensive error handling and fallback mechanisms
  - Complete test coverage and interface compliance verification

- **Arcconf support** - Added support for Adaptec/Microsemi RAID controllers
  - Detects RAID arrays via `arcconf getconfig <controller> ld`
  - Detects physical disks via `arcconf getconfig <controller> pd`
  - Supports multiple controllers with automatic discovery
  - Normalizes RAID levels (0, 1, 5, 6, 10, 50, 60) to standard format
  - Maps controller states to health values (optimal=1, degraded=2, failed=3)
  - Full integration with Linux disk detection system
  - **NEW**: Added SMART data enrichment for Arcconf RAID disks
  - **NEW**: Added battery support for Adaptec RAID controllers

- **Zpool support** - Added comprehensive ZFS pool and disk management
  - Detects ZFS pools via `zpool list` with health and capacity information
  - Extracts individual disks from pool configurations using `zpool status`
  - Supports various ZFS RAID levels (mirror, raidz1, raidz2, raidz3)
  - Maps ZFS health states (online=1, degraded=2, faulted=3) to numeric values
  - Enriches disk information using `lsblk` when available for additional metadata
  - Treats ZFS pools as storage arrays similar to hardware RAID

- **Hdparm support** - Added ATA/IDE disk parameter utility integration
  - Detects ATA/SATA disks and extracts detailed hardware information
  - Parses transport information (SATA version, interface type)
  - Extracts form factor, capacity, RPM, and SMART support status
  - Focuses on ATA/IDE/SATA devices (excludes NVMe which hdparm doesn't support)
  - Provides fallback disk interface detection for older systems

- **Enhanced tool version reporting** - Extended ToolInfo structure with version fields
  - Added `HdparmVersion`, `ArcconfVersion`, and `ZpoolVersion` fields
  - Complete version detection for all supported tools
  - Version information included in system tool reporting

- **Tool-specific battery parsing** - Each RAID tool now handles its own battery format
  - StoreCLI parses structured property-value battery output format
  - MegaCLI maintains existing battery parsing for legacy format
  - Arcconf added battery support with Adaptec-specific parsing
  - Added `ToolName` field to `RAIDBatteryInfo` for proper tool identification

- **Architectural improvements** - Refactored battery logic for better separation of concerns
  - Moved battery business logic from collector to utility functions
  - Created dedicated `utils/battery.go` for battery metric processing
  - Collector now acts as thin orchestration layer without business logic
  - Improved testability and maintainability of battery functionality

### Changed

- **Improved StoreCLI device naming** - Enhanced RAID disk device identifiers for better clarity
  - Changed from confusing `storcli-252:0` format to descriptive `raid-enc252-slot0`
  - Device names now clearly indicate enclosure and slot information
  - Consistent naming across JSON and plain text parsing modes
  - Better integration with monitoring dashboards and metrics

- **Complete tool integration** - All detected tools now fully integrated into Linux disk detection
  - Added arcconf, zpool, and hdparm to the main `GetDisks()` method
  - Tools are called when available and results merged with existing disk information
  - Proper filtering and deduplication of disk information across tools
  - Enhanced logging with tool-specific disk and array counts

### Fixed

- **MegaCLI parsing improvements** - Fixed critical parsing issues in MegaCLI RAID tool integration
  - **Corrected adapter ID regex** - Fixed regex pattern from `Adapter (\d+):` to `Adapter (\d+) --` to match real MegaCLI output format
  - **Fixed disk location parsing** - Updated parser to handle separate "Enclosure Device ID:" and "Slot Number:" lines instead of expecting them on the same line
  - **Corrected model extraction** - Fixed inquiry data parsing to extract first field (model name) instead of remaining fields, properly identifying disk models like `SAMSUNG-MZ7LH960HAJR`
  - **Fixed data processing order** - Ensured "Inquiry Data:" is processed before "Firmware state:" to prevent empty model names in disk information
  - **Enhanced battery metrics exposure** - Battery information now correctly parsed and exposed in Prometheus metrics for MegaCLI-based RAID systems

- **Comprehensive MegaCLI test suite** - Added robust unit testing for MegaCLI parsing logic
  - **Mock-based testing** - Created comprehensive mock MegaCLI tool with realistic command outputs for RAID arrays, battery info, and disk information
  - **Anonymized test data** - Mock data uses randomized but realistic hardware identifiers (WWNs, serial numbers, device IDs) for security
  - **Parser validation** - Mock parser logic exactly matches real parser implementation for reliable test coverage
  - **Error scenario testing** - Added tests for command failures and edge cases in MegaCLI parsing
  - **Real output alignment** - Test data format precisely matches actual MegaCLI command output structure and ordering

- **StoreCLI RAID status mapping** - Fixed RAID array status showing as 0 (unknown) instead of 1 (ok)
  - Added support for "OPTL" (Optimal) state mapping to status value 1
  - Added support for "DGRD" (Degraded) state mapping to status value 2
  - Enhanced RAID state parsing to handle Dell PERC controller states correctly
  - RAID arrays now properly show status 1 (ok) when in optimal state

- **StoreCLI battery support** - Added comprehensive battery monitoring for StoreCLI controllers
  - Battery information collection via `/c<id>/bbu show all` command
  - Battery temperature, voltage, current, and status monitoring
  - Support for battery replacement and capacity warnings
  - Battery metrics now available for Dell PERC and other StoreCLI-compatible controllers

- **StoreCLI SMART data collection** - Enhanced RAID disk monitoring with SMART data
  - Added SMART data collection for RAID disks via controller interface
  - Temperature monitoring for individual RAID disks (e.g., `raid-enc64-slot3`)
  - SMART health status detection for RAID-attached drives
  - Proper disk health status reporting instead of showing 0 (unknown)

- **Tool interface compliance** - Ensured all tools properly implement required interfaces
  - StoreCLI implements `CombinedToolInterface` (both disk and RAID detection)
  - Arcconf implements `RAIDToolInterface` (RAID-focused tool)
  - Zpool implements `DiskToolInterface` (disk detection with pool information)
  - Hdparm implements `DiskToolInterface` (disk parameter detection)
  - All tools include comprehensive test coverage and interface verification

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
