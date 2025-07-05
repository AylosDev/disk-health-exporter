#!/bin/bash
# VERSION 1.0
# Author: EdgardoAcosta

# Universal Installation script for Disk Health Exporter

set -e

# Default configuration
GITHUB_REPO="AylosDev/disk-health-exporter"
VERSION="latest"
INSTALL_SERVICE=""
BIN_DIR="/usr/local/bin"
DOWNLOADED_BINARY=""
SERVICE_PORT="9300"
COLLECT_INTERVAL="120s"
FORCE_INSTALL=""
SKIP_VERSION_CHECK=""
DRY_RUN=""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Show usage information
show_usage() {
  echo -e "${GREEN}Disk Health Prometheus Exporter Installer${NC}"
  echo ""
  echo "Usage: $0 [OPTIONS]"
  echo ""
  echo "Options:"
  echo "  -v, --version VERSION    Install specific version (default: latest)"
  echo "  -s, --service           Install as system service (systemd/launchd)"
  echo "  -p, --port PORT         Set service port (default: $SERVICE_PORT)"
  echo "  -f, --force             Force installation/update without confirmation"
  echo "  --skip-version-check    Skip version comparison check"
  echo "  --dry-run              Show what would be done without executing"
  echo "  -h, --help              Show this help message"
  echo ""
  echo "Examples:"
  echo "  $0                      # Install latest version"
  echo "  $0 -v v1.0.0          # Install specific version"
  echo "  $0 -s                  # Install latest with service"
  echo "  $0 -v v1.0.0 -s       # Install specific version with service"
  echo "  $0 -s -p 9200         # Install with service on port 9200"
  echo "  $0 -f                  # Force install without confirmation"
  echo "  $0 --dry-run           # See what would be installed"
}

# Parse command line arguments
parse_args() {
  while [[ $# -gt 0 ]]; do
    case $1 in
    -v | --version)
      VERSION="$2"
      shift 2
      ;;
    -s | --service)
      INSTALL_SERVICE="true"
      shift
      ;;
    -p | --port)
      SERVICE_PORT="$2"
      shift 2
      ;;
    -f | --force)
      FORCE_INSTALL="true"
      shift
      ;;
    --skip-version-check)
      SKIP_VERSION_CHECK="true"
      shift
      ;;
    --dry-run)
      DRY_RUN="true"
      shift
      ;;
    -h | --help)
      show_usage
      exit 0
      ;;
    *)
      echo -e "${RED}Unknown option: $1${NC}"
      show_usage
      exit 1
      ;;
    esac
  done
}

# Detect operating system and architecture
detect_os() {
  # Detect OS
  case "$(uname -s)" in
  Linux*)
    OS="linux"
    DISTRO="linux"
    ;;
  Darwin*)
    OS="darwin"
    DISTRO="macOS"
    ;;
  CYGWIN* | MINGW* | MSYS*)
    OS="windows"
    DISTRO="windows"
    ;;
  *)
    echo -e "${RED}Unsupported operating system: $(uname -s)${NC}"
    echo -e "${RED}Supported: Linux, macOS, Windows${NC}"
    exit 1
    ;;
  esac

  # Detect architecture
  case "$(uname -m)" in
  x86_64 | amd64)
    ARCH="amd64"
    ;;
  arm64 | aarch64)
    ARCH="arm64"
    ;;
  *)
    echo -e "${RED}Unsupported architecture: $(uname -m)${NC}"
    echo -e "${RED}Supported: amd64, arm64${NC}"
    exit 1
    ;;
  esac

  echo -e "${YELLOW}Detected: $DISTRO ($OS-$ARCH)${NC}"
}

# Download binary from GitHub releases
download_binary() {
  local binary_name=""

  if [[ "$OS" == "linux" ]]; then
    binary_name="disk-health-exporter-linux-${ARCH}"
  elif [[ "$OS" == "darwin" ]]; then
    binary_name="disk-health-exporter-darwin-${ARCH}"
  else
    echo -e "${RED}Unsupported OS: $OS${NC}"
    exit 1
  fi

  # https://github.com/OWNER/REPO/releases/download/TAG/ASSET_NAME
  local download_url="https://github.com/${GITHUB_REPO}/releases/download/${VERSION}/${binary_name}"
  local temp_file="/tmp/${binary_name}"

  echo -e "${YELLOW}Downloading ${binary_name}...${NC}"
  echo -e "${BLUE}URL: $download_url${NC}"

  if [[ "$DRY_RUN" == "true" ]]; then
    echo "[DRY RUN] Would download from: $download_url"
    echo "[DRY RUN] Would save to: $temp_file"
    DOWNLOADED_BINARY="$temp_file"
    return 0
  fi

  # Check if curl or wget is available
  if command -v curl >/dev/null 2>&1; then
    if ! curl -L -f -o "$temp_file" "$download_url"; then
      echo -e "${RED}Failed to download binary. Please check if version $VERSION exists.${NC}"
      echo -e "${YELLOW}You can check available releases at: https://github.com/${GITHUB_REPO}/releases${NC}"
      exit 1
    fi
  elif command -v wget >/dev/null 2>&1; then
    if ! wget -O "$temp_file" "$download_url"; then
      echo -e "${RED}Failed to download binary. Please check if version $VERSION exists.${NC}"
      echo -e "${YELLOW}You can check available releases at: https://github.com/${GITHUB_REPO}/releases${NC}"
      exit 1
    fi
  else
    echo -e "${RED}Neither curl nor wget found. Please install one of them.${NC}"
    exit 1
  fi

  if [[ ! -f "$temp_file" ]] || [[ ! -s "$temp_file" ]]; then
    echo -e "${RED}Downloaded file is empty or missing${NC}"
    exit 1
  fi

  # Make it executable
  chmod +x "$temp_file"

  echo -e "${GREEN}Binary downloaded successfully${NC}"
  DOWNLOADED_BINARY="$temp_file"
}

# Check dependencies for Linux (warning only)
check_linux_deps() {
  if [[ "$INSTALL_SERVICE" != "true" ]]; then
    return 0
  fi

  echo -e "${YELLOW}Checking Linux dependencies for service setup...${NC}"

  # Check if running as root
  if [[ $EUID -eq 0 ]]; then
    echo -e "${RED}This script should not be run as root${NC}"
    echo "Please run as a regular user with sudo privileges"
    exit 1
  fi

  # Check for smartmontools
  if ! command -v smartctl &>/dev/null; then
    echo -e "${YELLOW}WARNING: smartmontools not found${NC}"
    echo "The exporter may have limited functionality without smartctl"
    echo "To install: sudo apt-get install smartmontools (Debian/Ubuntu)"
    echo "           sudo yum install smartmontools (RHEL/CentOS)"
    echo "           sudo dnf install smartmontools (Fedora)"
    echo "           sudo pacman -S smartmontools (Arch)"
  else
    echo -e "${GREEN}smartmontools found: $(smartctl --version | head -n1)${NC}"
  fi

  # Check for optional RAID tools
  local raid_tools_found=0

  if command -v megacli &>/dev/null || command -v MegaCli64 &>/dev/null; then
    echo -e "${GREEN}MegaCLI found${NC}"
    raid_tools_found=1
  fi

  if command -v storcli &>/dev/null || command -v storcli64 &>/dev/null; then
    echo -e "${GREEN}StorCLI found${NC}"
    raid_tools_found=1
  fi

  if command -v arcconf &>/dev/null; then
    echo -e "${GREEN}Arcconf found${NC}"
    raid_tools_found=1
  fi

  if command -v mdadm &>/dev/null; then
    echo -e "${GREEN}mdadm found${NC}"
    raid_tools_found=1
  fi

  if [[ $raid_tools_found -eq 0 ]]; then
    echo -e "${YELLOW}WARNING: No RAID management tools found${NC}"
    echo "RAID monitoring will be limited without MegaCLI, StorCLI, Arcconf, or mdadm"
  fi

  # Create exporter user if it doesn't exist (only for service)
  # if ! id -u exporter >/dev/null 2>&1; then
  #   echo -e "${YELLOW}Creating exporter user...${NC}"
  #   sudo useradd --system --no-create-home --shell /bin/false --user-group --comment "Disk Health Exporter Service User" exporter
  # fi
}

# Check dependencies for macOS (warning only)
check_macos_deps() {
  if [[ "$INSTALL_SERVICE" != "true" ]]; then
    return 0
  fi

  echo -e "${YELLOW}Checking macOS dependencies for service setup...${NC}"

  # Check for smartmontools
  if ! command -v smartctl &>/dev/null; then
    echo -e "${YELLOW}WARNING: smartmontools not found${NC}"
    echo "The exporter may have limited functionality without smartctl"
    if command -v brew &>/dev/null; then
      echo "To install: brew install smartmontools"
    else
      echo "Please install Homebrew first: https://brew.sh"
      echo "Then run: brew install smartmontools"
    fi
  else
    echo -e "${GREEN}smartmontools found: $(smartctl --version | head -n1)${NC}"
  fi

  # Check for diskutil (should be available on all macOS systems)
  if command -v diskutil &>/dev/null; then
    echo -e "${GREEN}diskutil found${NC}"
  else
    echo -e "${YELLOW}WARNING: diskutil not found (unusual for macOS)${NC}"
  fi

  echo -e "${YELLOW}Note: RAID monitoring on macOS is limited${NC}"
  echo "Hardware RAID controllers are rare on Mac systems"
}

# Setup Linux systemd service
setup_linux_service() {
  if [[ "$INSTALL_SERVICE" != "true" ]]; then
    return 0
  fi

  echo -e "${YELLOW}Creating systemd service...${NC}"

  if [[ "$DRY_RUN" == "true" ]]; then
    echo "[DRY RUN] Would create systemd service file at /etc/systemd/system/disk-health-exporter.service"
    echo "[DRY RUN] Would enable and configure service to run on port ${SERVICE_PORT}"
    return 0
  fi

  # Create systemd service file
  sudo tee /etc/systemd/system/disk-health-exporter.service >/dev/null <<EOF
[Unit]
Description=Disk Health Prometheus Exporter
Documentation=https://github.com/${GITHUB_REPO}
After=network.target

[Service]
Type=simple
User=root
Group=root
ExecStart=${BIN_DIR}/disk-health-exporter --port=${SERVICE_PORT} --collect-interval=${COLLECT_INTERVAL}
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

  # Reload systemd and enable service
  sudo systemctl daemon-reload
  sudo systemctl enable disk-health-exporter.service

  echo -e "${GREEN}Service configuration completed successfully!${NC}"
  echo ""
  echo "Service will be started after installation completes"
  echo "Metrics URL: http://localhost:${SERVICE_PORT}/metrics"
  echo ""
  echo "To check logs: sudo journalctl -u disk-health-exporter.service -f"
  echo "To restart: sudo systemctl restart disk-health-exporter.service"
  echo "To stop: sudo systemctl stop disk-health-exporter.service"
}

# Setup macOS LaunchAgent service
setup_macos_service() {
  if [[ "$INSTALL_SERVICE" != "true" ]]; then
    return 0
  fi

  # Create a simple launchd plist for macOS
  PLIST_FILE="com.diskhealth.exporter.plist"
  PLIST_PATH="$HOME/Library/LaunchAgents/$PLIST_FILE"

  echo -e "${YELLOW}Creating LaunchAgent plist...${NC}"

  if [[ "$DRY_RUN" == "true" ]]; then
    echo "[DRY RUN] Would create LaunchAgent plist at $PLIST_PATH"
    echo "[DRY RUN] Would configure service to run on port ${SERVICE_PORT}"
    return 0
  fi

  # Ensure LaunchAgents directory exists
  mkdir -p "$HOME/Library/LaunchAgents"

  # Create the plist file with proper structure
  cat >"$PLIST_PATH" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>com.diskhealth.exporter</string>
	<key>ProgramArguments</key>
	<array>
		<string>${BIN_DIR}/disk-health-exporter</string>
		<string>--port</string>
		<string>${SERVICE_PORT}</string>
	</array>
	<key>RunAtLoad</key>
	<true/>
	<key>KeepAlive</key>
	<true/>
	<key>StandardOutPath</key>
	<string>/tmp/disk-health-exporter.log</string>
	<key>StandardErrorPath</key>
	<string>/tmp/disk-health-exporter.err</string>
	<key>WorkingDirectory</key>
	<string>/tmp</string>
</dict>
</plist>
EOF

  # Set proper permissions on the plist file
  chmod 644 "$PLIST_PATH"

  # Validate the plist file
  if ! plutil -lint "$PLIST_PATH" >/dev/null 2>&1; then
    echo -e "${RED}Error: Invalid plist file created${NC}"
    return 1
  fi

  echo -e "${GREEN}LaunchAgent configuration completed successfully!${NC}"
  echo ""
  echo "Service will be started after installation completes"
  echo "Metrics URL: http://localhost:${SERVICE_PORT}/metrics"
  echo ""
  echo "Log files:"
  echo "  Output: /tmp/disk-health-exporter.log"
  echo "  Errors: /tmp/disk-health-exporter.err"
  echo ""
  echo "Service management:"
  echo "  Check logs: tail -f /tmp/disk-health-exporter.log"
  echo "  Restart: launchctl bootout gui/\$(id -u) $PLIST_PATH && launchctl bootstrap gui/\$(id -u) $PLIST_PATH"
  echo "  Stop: launchctl bootout gui/\$(id -u) $PLIST_PATH"
  echo "  Status: launchctl list | grep com.diskhealth.exporter"
  echo ""
  echo "To test manually: ${BIN_DIR}/disk-health-exporter --port=${SERVICE_PORT}"
}

# Check if application already exists and compare versions
check_existing_installation() {
  if [[ "$SKIP_VERSION_CHECK" == "true" ]]; then
    echo -e "${YELLOW}Skipping version check...${NC}"
    return 0
  fi

  local binary_path="${BIN_DIR}/disk-health-exporter"

  if [[ ! -f "$binary_path" ]]; then
    echo -e "${YELLOW}No existing installation found${NC}"
    return 0
  fi

  echo -e "${YELLOW}Found existing installation at $binary_path${NC}"

  # Get current version
  local current_version=""
  if current_version=$("$binary_path" --version 2>/dev/null | grep -oE 'v[0-9]+\.[0-9]+\.[0-9]+' | head -n1); then
    echo -e "${BLUE}Current version: $current_version${NC}"
  else
    echo -e "${YELLOW}Could not determine current version${NC}"
    current_version="unknown"
  fi

  # If user specified a version, compare it
  if [[ "$VERSION" != "latest" ]] && [[ "$current_version" == "$VERSION" ]]; then
    echo -e "${GREEN}Version $VERSION is already installed${NC}"
    if [[ "$FORCE_INSTALL" != "true" ]]; then
      echo "Use --force to reinstall"
      exit 0
    fi
  fi

  # If installing latest, we'll proceed (could be newer version available)
  if [[ "$VERSION" == "latest" ]] && [[ "$current_version" != "unknown" ]]; then
    echo -e "${YELLOW}Will attempt to update from $current_version to latest${NC}"
  fi

  # Ask for confirmation unless force is used
  if [[ "$FORCE_INSTALL" != "true" ]] && [[ "$DRY_RUN" != "true" ]]; then
    echo ""
    read -p "Do you want to proceed with the installation/update? (y/N): " -n 1 -r
    echo ""
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
      echo "Installation cancelled"
      exit 0
    fi
  fi
}

# Stop existing service if running
stop_existing_service() {
  echo -e "${YELLOW}Checking for running services...${NC}"

  case "$OS" in
  linux)
    if systemctl is-active --quiet disk-health-exporter.service 2>/dev/null; then
      echo -e "${YELLOW}Stopping systemd service...${NC}"
      if [[ "$DRY_RUN" != "true" ]]; then
        sudo systemctl stop disk-health-exporter.service
      else
        echo "[DRY RUN] Would stop systemd service"
      fi
    else
      echo -e "${BLUE}No systemd service running${NC}"
    fi
    ;;
  darwin)
    local plist_path="$HOME/Library/LaunchAgents/com.diskhealth.exporter.plist"
    # Check if service is loaded
    if launchctl list | grep -q com.diskhealth.exporter 2>/dev/null; then
      echo -e "${YELLOW}Stopping LaunchAgent...${NC}"
      if [[ "$DRY_RUN" != "true" ]]; then
        # Try modern bootout first, fallback to unload
        launchctl bootout "gui/$(id -u)" "$plist_path" 2>/dev/null ||
          launchctl unload "$plist_path" 2>/dev/null || true
      else
        echo "[DRY RUN] Would stop LaunchAgent"
      fi
    else
      echo -e "${BLUE}No LaunchAgent running${NC}"
    fi
    ;;
  esac
}

# Start/restart service after installation
restart_service() {
  if [[ "$INSTALL_SERVICE" != "true" ]]; then
    return 0
  fi

  echo -e "${YELLOW}Starting service...${NC}"

  case "$OS" in
  linux)
    if [[ "$DRY_RUN" != "true" ]]; then
      sudo systemctl start disk-health-exporter.service
    else
      echo "[DRY RUN] Would start systemd service"
    fi
    ;;
  darwin)
    local plist_path="$HOME/Library/LaunchAgents/com.diskhealth.exporter.plist"
    if [[ -f "$plist_path" ]]; then
      if [[ "$DRY_RUN" != "true" ]]; then
        # Try modern bootstrap first, fallback to load
        if ! launchctl bootstrap "gui/$(id -u)" "$plist_path" 2>/dev/null; then
          launchctl load "$plist_path" 2>/dev/null || echo "Failed to start service"
        fi
      else
        echo "[DRY RUN] Would start LaunchAgent"
      fi
    fi
    ;;
  esac
}

# Test connection to the application
test_connection() {
  local max_attempts=30
  local attempt=1
  local url="http://localhost:${SERVICE_PORT}/metrics"

  echo -e "${YELLOW}Testing connection to $url...${NC}"

  if [[ "$DRY_RUN" == "true" ]]; then
    echo "[DRY RUN] Would test connection to $url"
    return 0
  fi

  # Wait for service to start
  sleep 2

  while [[ $attempt -le $max_attempts ]]; do
    if command -v curl >/dev/null 2>&1; then
      if curl -s --connect-timeout 2 "$url" >/dev/null 2>&1; then
        echo -e "${GREEN}✓ Service is responding on port ${SERVICE_PORT}${NC}"
        echo -e "${BLUE}Metrics endpoint: $url${NC}"
        return 0
      fi
    elif command -v wget >/dev/null 2>&1; then
      if wget -q --timeout=2 --tries=1 -O /dev/null "$url" 2>/dev/null; then
        echo -e "${GREEN}✓ Service is responding on port ${SERVICE_PORT}${NC}"
        echo -e "${BLUE}Metrics endpoint: $url${NC}"
        return 0
      fi
    else
      echo -e "${YELLOW}Warning: Neither curl nor wget available for testing${NC}"
      return 1
    fi

    echo -n "."
    sleep 1
    ((attempt++))
  done

  echo ""
  echo -e "${RED}✗ Service is not responding after ${max_attempts} seconds${NC}"
  echo "Check logs or try starting manually:"
  case "$OS" in
  linux)
    echo "  sudo journalctl -u disk-health-exporter.service -f"
    echo "  sudo systemctl status disk-health-exporter.service"
    ;;
  darwin)
    echo "  tail -f /tmp/disk-health-exporter.log"
    echo "  ${BIN_DIR}/disk-health-exporter --port=${SERVICE_PORT}"
    ;;
  esac
  return 1
}

# Get latest version from releases page (without API)
get_latest_version_from_releases() {
  if [[ "$VERSION" != "latest" ]]; then
    echo -e "${YELLOW}Using specified version: $VERSION${NC}"
    return 0
  fi

  echo -e "${YELLOW}Getting latest release version from GitHub releases page...${NC}"
  local releases_url="https://github.com/${GITHUB_REPO}/releases/latest"

  if command -v curl >/dev/null 2>&1; then
    # Follow redirects and extract version from final URL
    VERSION=$(curl -s -L -I "$releases_url" | grep -i "location:" | tail -1 | sed -E 's/.*\/([^\/\r\n]+).*/\1/' | tr -d '\r\n')
  elif command -v wget >/dev/null 2>&1; then
    # Use wget to follow redirects and extract version
    VERSION=$(wget -qO- --max-redirect=1 "$releases_url" 2>&1 | grep -oE 'v[0-9]+\.[0-9]+\.[0-9]+' | head -1)
  else
    echo -e "${RED}Neither curl nor wget found. Please install one of them.${NC}"
    exit 1
  fi

  if [[ -z "$VERSION" ]] || [[ "$VERSION" == "latest" ]]; then
    # Fallback: try to parse from releases page HTML
    echo -e "${YELLOW}Fallback: parsing releases page...${NC}"
    local releases_page_url="https://github.com/${GITHUB_REPO}/releases"

    if command -v curl >/dev/null 2>&1; then
      VERSION=$(curl -s "$releases_page_url" | grep -oE '/releases/tag/v[0-9]+\.[0-9]+\.[0-9]+' | head -1 | sed 's/.*tag\///')
    elif command -v wget >/dev/null 2>&1; then
      VERSION=$(wget -qO- "$releases_page_url" | grep -oE '/releases/tag/v[0-9]+\.[0-9]+\.[0-9]+' | head -1 | sed 's/.*tag\///')
    fi
  fi

  if [[ -z "$VERSION" ]] || [[ "$VERSION" == "latest" ]]; then
    echo -e "${RED}Failed to get latest version. Please specify version with -v flag${NC}"
    echo -e "${YELLOW}Example: $0 -v v1.0.0${NC}"
    exit 1
  fi

  echo -e "${GREEN}Latest version: $VERSION${NC}"
}

# Main installation logic
main() {
  # Parse command line arguments
  parse_args "$@"

  echo -e "${GREEN}Installing Disk Health Prometheus Exporter${NC}"

  if [[ "$DRY_RUN" == "true" ]]; then
    echo -e "${YELLOW}DRY RUN MODE - No actual changes will be made${NC}"
    echo ""
  fi

  # Check for required tools
  if ! command -v curl >/dev/null 2>&1 && ! command -v wget >/dev/null 2>&1; then
    echo -e "${RED}Neither curl nor wget found. Please install one of them first.${NC}"
    exit 1
  fi

  # Detect OS and architecture
  detect_os

  # Check existing installation and compare versions
  check_existing_installation

  # Get version to install (without using GitHub API)
  get_latest_version_from_releases

  # Stop existing service if running
  stop_existing_service

  # Download binary from GitHub releases
  download_binary

  # Install binary
  echo -e "${YELLOW}Installing binary to ${BIN_DIR}...${NC}"

  if [[ "$DRY_RUN" == "true" ]]; then
    echo "[DRY RUN] Would install binary to ${BIN_DIR}/disk-health-exporter"
    echo "[DRY RUN] Would set appropriate permissions"
  else
    sudo mkdir -p "$BIN_DIR"
    sudo cp "$DOWNLOADED_BINARY" "${BIN_DIR}/disk-health-exporter"

    case "$OS" in
    linux)
      sudo chown root:root "${BIN_DIR}/disk-health-exporter"
      ;;
    darwin)
      sudo chown root:wheel "${BIN_DIR}/disk-health-exporter"
      ;;
    esac

    sudo chmod 755 "${BIN_DIR}/disk-health-exporter"

    # Clean up downloaded file
    rm -f "$DOWNLOADED_BINARY"

    echo -e "${GREEN}Binary installed successfully!${NC}"
  fi

  # Check dependencies and setup service if requested
  case "$OS" in
  linux)
    check_linux_deps
    setup_linux_service
    ;;
  darwin)
    check_macos_deps
    setup_macos_service
    ;;
  *)
    echo -e "${RED}Unsupported OS: $OS${NC}"
    exit 1
    ;;
  esac

  # Start/restart service after installation
  restart_service

  # Test connection to the application
  if [[ "$INSTALL_SERVICE" == "true" ]]; then
    test_connection
  fi

  # Show final installation summary
  echo ""
  echo -e "${GREEN}Installation completed successfully!${NC}"
  echo ""
  echo "Binary installed at: ${BIN_DIR}/disk-health-exporter"
  echo "Version: $VERSION"

  if [[ "$INSTALL_SERVICE" == "true" ]]; then
    echo "Service: Installed and configured"
    echo "Metrics URL: http://localhost:${SERVICE_PORT}/metrics"
  else
    echo "Service: Not installed (use -s flag to install service)"
    echo "To run manually: ${BIN_DIR}/disk-health-exporter --port=${SERVICE_PORT}"
  fi

  if [[ "$DRY_RUN" != "true" ]] && [[ "$INSTALL_SERVICE" != "true" ]]; then
    echo ""
    echo "To test manually: ${BIN_DIR}/disk-health-exporter --port=${SERVICE_PORT} &"
    echo "Then check: curl http://localhost:${SERVICE_PORT}/metrics"
  fi
}

# Run main function
main "$@"
