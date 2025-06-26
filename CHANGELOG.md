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
