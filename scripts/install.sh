#!/bin/bash

# Universal Installation script for Disk Health Exporter
# Downloads latest release from GitHub and installs accordingly

set -e

# Default configuration
GITHUB_REPO="AylosDev/disk-health-exporter"
VERSION=""
INSTALL_SERVICE=""
BIN_DIR="/usr/local/bin"
DOWNLOADED_BINARY=""
SERVICE_PORT="9300"

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
  echo "  -h, --help              Show this help message"
  echo ""
  echo "Examples:"
  echo "  $0                      # Install latest version"
  echo "  $0 -v v1.0.0          # Install specific version"
  echo "  $0 -s                  # Install latest with service"
  echo "  $0 -v v1.0.0 -s       # Install specific version with service"
  echo "  $0 -s -p 9200         # Install with service on port 9200"
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

echo -e "${GREEN}Installing Disk Health Prometheus Exporter${NC}"

# Detect operating system and architecture
detect_os() {
  case "$OSTYPE" in
  linux*)
    if [[ -f /etc/os-release ]]; then
      . /etc/os-release
      OS="linux"
      DISTRO="$ID"
      VER="$VERSION_ID"
    else
      OS="linux"
      DISTRO="unknown"
      VER="unknown"
    fi
    ;;
  darwin*)
    OS="darwin"
    DISTRO="macos"
    VER=$(sw_vers -productVersion)
    ;;
  *)
    echo -e "${RED}Unsupported operating system: $OSTYPE${NC}"
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
    exit 1
    ;;
  esac

  echo -e "${YELLOW}Detected: $DISTRO $VER ($OS-$ARCH)${NC}"
}

# Get latest release version from GitHub
get_latest_version() {
  echo -e "${YELLOW}Getting latest release version...${NC}"
  local latest_url="https://api.github.com/repos/${GITHUB_REPO}/releases/latest"

  if command -v curl >/dev/null 2>&1; then
    VERSION=$(curl -s "$latest_url" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
  elif command -v wget >/dev/null 2>&1; then
    VERSION=$(wget -qO- "$latest_url" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
  else
    echo -e "${RED}Neither curl nor wget found. Please install one of them.${NC}"
    exit 1
  fi

  if [[ -z "$VERSION" ]]; then
    echo -e "${RED}Failed to get latest version${NC}"
    exit 1
  fi

  echo -e "${GREEN}Latest version: $VERSION${NC}"
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

  # Correct GitHub releases download URL format:
  # https://github.com/OWNER/REPO/releases/download/TAG/ASSET_NAME
  local download_url="https://github.com/${GITHUB_REPO}/releases/download/${VERSION}/${binary_name}"
  local temp_file="/tmp/${binary_name}"

  echo -e "${YELLOW}Downloading ${binary_name}...${NC}"
  echo -e "${BLUE}URL: $download_url${NC}"

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
  if ! id -u exporter >/dev/null 2>&1; then
    echo -e "${YELLOW}Creating exporter user...${NC}"
    sudo useradd --system --no-create-home --shell /bin/false --user-group --comment "Disk Health Exporter Service User" exporter
  fi
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

  # Create systemd service file
  sudo tee /etc/systemd/system/disk-health-exporter.service >/dev/null <<EOF
[Unit]
Description=Disk Health Prometheus Exporter
Documentation=https://github.com/${GITHUB_REPO}
After=network.target

[Service]
Type=simple
User=exporter
Group=exporter
ExecStart=${BIN_DIR}/disk-health-exporter --port=${SERVICE_PORT}
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

  # Start the service
  echo -e "${YELLOW}Starting the service...${NC}"
  sudo systemctl start disk-health-exporter.service

  # Check status
  echo -e "${YELLOW}Checking service status...${NC}"
  sudo systemctl status disk-health-exporter.service --no-pager

  echo ""
  echo -e "${GREEN}Service installation completed successfully!${NC}"
  echo ""
  echo "The exporter is running on port ${SERVICE_PORT}"
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
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/tmp/disk-health-exporter.log</string>
    <key>StandardErrorPath</key>
    <string>/tmp/disk-health-exporter.error.log</string>
    <key>EnvironmentVariables</key>
    <dict>
        <key>PORT</key>
        <string>${SERVICE_PORT}</string>
    </dict>
</dict>
</plist>
EOF

  # Load the LaunchAgent
  echo -e "${YELLOW}Loading LaunchAgent...${NC}"
  launchctl load "$PLIST_PATH"

  # Wait a moment for the service to start
  sleep 2

  # Check if the service is running
  echo -e "${YELLOW}Checking if service is running...${NC}"
  if curl -s http://localhost:"${SERVICE_PORT}"/metrics >/dev/null; then
    echo -e "${GREEN}Service is running successfully!${NC}"
  else
    echo -e "${YELLOW}Service might be starting up, checking logs...${NC}"
    tail -5 /tmp/disk-health-exporter.log 2>/dev/null || echo "No logs yet"
  fi

  echo ""
  echo -e "${GREEN}Service installation completed successfully!${NC}"
  echo ""
  echo "The exporter is running on port ${SERVICE_PORT}"
  echo "Metrics URL: http://localhost:${SERVICE_PORT}/metrics"
  echo ""
  echo "To check logs: tail -f /tmp/disk-health-exporter.log"
  echo "To restart: launchctl unload $PLIST_PATH && launchctl load $PLIST_PATH"
  echo "To stop: launchctl unload $PLIST_PATH"
  echo ""
  echo "To test manually: ${BIN_DIR}/disk-health-exporter"
}

# Main installation logic
main() {
  # Parse command line arguments
  parse_args "$@"

  # Check for required tools
  if ! command -v curl >/dev/null 2>&1 && ! command -v wget >/dev/null 2>&1; then
    echo -e "${RED}Neither curl nor wget found. Please install one of them first.${NC}"
    exit 1
  fi

  # Detect OS and architecture
  detect_os

  # Get version to install
  if [[ -z "$VERSION" ]]; then
    get_latest_version
  else
    echo -e "${YELLOW}Installing version: $VERSION${NC}"
  fi

  # Download binary from GitHub releases
  download_binary

  # Install binary
  echo -e "${YELLOW}Installing binary to ${BIN_DIR}...${NC}"
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

  # Show final installation summary
  echo ""
  echo -e "${GREEN}Installation completed successfully!${NC}"
  echo ""
  echo "Binary installed at: ${BIN_DIR}/disk-health-exporter"
  echo "Version: $VERSION"

  if [[ "$INSTALL_SERVICE" == "true" ]]; then
    echo "Service: Installed and running"
    echo "Metrics URL: http://localhost:${SERVICE_PORT}/metrics"
  else
    echo "Service: Not installed (use -s flag to install service)"
    echo "To run manually: ${BIN_DIR}/disk-health-exporter"
  fi

  echo ""
  echo "To test: curl http://localhost:${SERVICE_PORT}/metrics"
}

# Run main function
main "$@"
