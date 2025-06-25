#!/bin/bash

# Universal Installation script for Disk Health Exporter
# Auto-detects OS and installs accordingly

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Installing Disk Health Prometheus Exporter${NC}"

# Detect operating system
detect_os() {
  case "$OSTYPE" in
  linux*)
    if [[ -f /etc/os-release ]]; then
      . /etc/os-release
      OS="$ID"
      VER="$VERSION_ID"
    else
      OS="linux"
      VER="unknown"
    fi
    ;;
  darwin*)
    OS="macos"
    VER=$(sw_vers -productVersion)
    ;;
  *)
    echo -e "${RED}Unsupported operating system: $OSTYPE${NC}"
    exit 1
    ;;
  esac
  echo -e "${YELLOW}Detected OS: $OS $VER${NC}"
}

# Install dependencies for Linux
install_linux_deps() {
  echo -e "${YELLOW}Installing Linux dependencies...${NC}"

  # Check if running as root
  if [[ $EUID -eq 0 ]]; then
    echo -e "${RED}This script should not be run as root${NC}"
    echo "Please run as a regular user with sudo privileges"
    exit 1
  fi

  # Detect package manager and install dependencies
  if command -v apt-get &>/dev/null; then
    sudo apt-get update
    sudo apt-get install -y smartmontools curl
  elif command -v yum &>/dev/null; then
    sudo yum install -y smartmontools curl
  elif command -v dnf &>/dev/null; then
    sudo dnf install -y smartmontools curl
  elif command -v pacman &>/dev/null; then
    sudo pacman -Sy --noconfirm smartmontools curl
  else
    echo -e "${RED}No supported package manager found${NC}"
    echo "Please install smartmontools and curl manually"
    exit 1
  fi

  # Try to install MegaCLI (optional)
  echo -e "${YELLOW}Checking for MegaCLI...${NC}"
  if ! command -v megacli &>/dev/null && ! command -v MegaCli64 &>/dev/null; then
    echo -e "${YELLOW}MegaCLI not found. This is optional for RAID monitoring.${NC}"
    echo "To install MegaCLI, download it from Broadcom's website"
  fi

  # Create prometheus user if it doesn't exist
  if ! id -u prometheus >/dev/null 2>&1; then
    echo -e "${YELLOW}Creating prometheus user...${NC}"
    sudo useradd --no-create-home --shell /bin/false prometheus
  fi
}

# Install dependencies for macOS
install_macos_deps() {
  echo -e "${YELLOW}Installing macOS dependencies...${NC}"

  # Check if Homebrew is installed
  if ! command -v brew &>/dev/null; then
    echo -e "${RED}Homebrew is required but not installed${NC}"
    echo "Please install Homebrew first: https://brew.sh"
    exit 1
  fi

  # Install system dependencies
  brew update
  brew install smartmontools

  # Note about RAID support on macOS
  echo -e "${YELLOW}Note: RAID monitoring on macOS is limited${NC}"
  echo "MegaCLI is not typically available on macOS"
  echo "This exporter will focus on regular disk monitoring via smartctl"
}

# Setup Linux systemd service
setup_linux_service() {
  echo -e "${YELLOW}Installing systemd service...${NC}"
  sudo cp deployments/disk-health-exporter.service /etc/systemd/system/
  sudo systemctl daemon-reload
  sudo systemctl enable disk-health-exporter.service

  # Start the service
  echo -e "${YELLOW}Starting the service...${NC}"
  sudo systemctl start disk-health-exporter.service

  # Check status
  echo -e "${YELLOW}Checking service status...${NC}"
  sudo systemctl status disk-health-exporter.service --no-pager

  echo ""
  echo -e "${GREEN}Installation completed successfully!${NC}"
  echo ""
  echo "The exporter is running on port 9100"
  echo "Metrics URL: http://localhost:9100/metrics"
  echo ""
  echo "To check logs: sudo journalctl -u disk-health-exporter.service -f"
  echo "To restart: sudo systemctl restart disk-health-exporter.service"
  echo "To stop: sudo systemctl stop disk-health-exporter.service"
}

# Setup macOS LaunchAgent service
setup_macos_service() {
  # Create a simple launchd plist for macOS
  PLIST_FILE="com.diskhealth.exporter.plist"
  PLIST_PATH="$HOME/Library/LaunchAgents/$PLIST_FILE"
  BIN_DIR="/usr/local/bin"

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
        <string>9100</string>
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
  if curl -s http://localhost:9100/metrics >/dev/null; then
    echo -e "${GREEN}Service is running successfully!${NC}"
  else
    echo -e "${YELLOW}Service might be starting up, checking logs...${NC}"
    tail -5 /tmp/disk-health-exporter.log 2>/dev/null || echo "No logs yet"
  fi

  echo ""
  echo -e "${GREEN}Installation completed successfully!${NC}"
  echo ""
  echo "The exporter is running on port 9100"
  echo "Metrics URL: http://localhost:9100/metrics"
  echo ""
  echo "To check logs: tail -f /tmp/disk-health-exporter.log"
  echo "To restart: launchctl unload $PLIST_PATH && launchctl load $PLIST_PATH"
  echo "To stop: launchctl unload $PLIST_PATH"
  echo ""
  echo "To test manually: ${BIN_DIR}/disk-health-exporter"
}

# Main installation logic
main() {
  # Detect OS
  detect_os

  # Build the exporter (common for all platforms)
  echo -e "${YELLOW}Building the exporter...${NC}"
  cd "$(dirname "$0")/.."
  go build -o disk-health-exporter ./cmd/disk-health-exporter

  # Install binary based on OS
  BIN_DIR="/usr/local/bin"
  echo -e "${YELLOW}Installing binary to ${BIN_DIR}...${NC}"

  case "$OS" in
  ubuntu | debian | centos | fedora | rhel | arch | *linux*)
    install_linux_deps
    sudo cp disk-health-exporter ${BIN_DIR}/
    sudo chown root:root ${BIN_DIR}/disk-health-exporter
    sudo chmod 755 ${BIN_DIR}/disk-health-exporter
    setup_linux_service
    ;;
  macos)
    install_macos_deps
    sudo cp disk-health-exporter ${BIN_DIR}/
    sudo chown root:wheel ${BIN_DIR}/disk-health-exporter
    sudo chmod 755 ${BIN_DIR}/disk-health-exporter
    setup_macos_service
    ;;
  *)
    echo -e "${RED}Unsupported OS: $OS${NC}"
    exit 1
    ;;
  esac
}

# Run main function
main "$@"
