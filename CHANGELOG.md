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
