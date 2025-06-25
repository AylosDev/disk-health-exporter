#!/bin/bash

# Demo script for Disk Health Exporter
# Shows OS detection and script capabilities without full installation

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${GREEN}=== Disk Health Exporter Demo ===${NC}"
echo ""

# Detect operating system
detect_os() {
  echo -e "${BLUE}1. Operating System Detection:${NC}"
  case "$OSTYPE" in
  linux*)
    if [[ -f /etc/os-release ]]; then
      . /etc/os-release
      OS="$ID"
      VER="$VERSION_ID"
      PRETTY_NAME="$PRETTY_NAME"
    else
      OS="linux"
      VER="unknown"
      PRETTY_NAME="Linux (unknown distribution)"
    fi
    echo -e "   Detected: ${GREEN}$PRETTY_NAME${NC}"
    echo -e "   OS Type: ${GREEN}Linux${NC}"
    echo -e "   OS ID: ${GREEN}$OS${NC}"
    echo -e "   Version: ${GREEN}$VER${NC}"
    ;;
  darwin*)
    OS="macos"
    VER=$(sw_vers -productVersion)
    PRETTY_NAME="macOS $VER"
    echo -e "   Detected: ${GREEN}$PRETTY_NAME${NC}"
    echo -e "   OS Type: ${GREEN}macOS${NC}"
    echo -e "   Version: ${GREEN}$VER${NC}"
    ;;
  *)
    echo -e "   ${RED}Unsupported operating system: $OSTYPE${NC}"
    exit 1
    ;;
  esac
  echo ""
}

# Show available tools
show_tools() {
  echo -e "${BLUE}2. Available Tools:${NC}"

  # Check Go
  if command -v go &>/dev/null; then
    GO_VERSION=$(go version | cut -d' ' -f3)
    echo -e "   ✓ Go: ${GREEN}$GO_VERSION${NC}"
  else
    echo -e "   ✗ Go: ${RED}Not installed${NC}"
  fi

  # Check smartctl
  if command -v smartctl &>/dev/null; then
    SMARTCTL_VERSION=$(smartctl --version | head -1 | cut -d' ' -f2)
    echo -e "   ✓ smartctl: ${GREEN}$SMARTCTL_VERSION${NC}"
  else
    echo -e "   ✗ smartctl: ${RED}Not installed${NC}"
  fi

  # Check MegaCLI (Linux only)
  if [[ "$OS" != "macos" ]]; then
    if command -v megacli &>/dev/null; then
      echo -e "   ✓ MegaCLI: ${GREEN}Available${NC}"
    elif command -v MegaCli64 &>/dev/null; then
      echo -e "   ✓ MegaCLI64: ${GREEN}Available${NC}"
    else
      echo -e "   ✗ MegaCLI: ${YELLOW}Not installed (optional for RAID)${NC}"
    fi
  else
    echo -e "   - MegaCLI: ${YELLOW}Not applicable on macOS${NC}"
  fi

  # Check curl
  if command -v curl &>/dev/null; then
    echo -e "   ✓ curl: ${GREEN}Available${NC}"
  else
    echo -e "   ✗ curl: ${RED}Not installed${NC}"
  fi

  echo ""
}

# Show system information
show_system_info() {
  echo -e "${BLUE}3. System Information:${NC}"
  echo -e "   Hostname: ${GREEN}$(hostname)${NC}"
  echo -e "   Kernel: ${GREEN}$(uname -r)${NC}"
  echo -e "   Architecture: ${GREEN}$(uname -m)${NC}"

  if [[ "$OS" == "macos" ]]; then
    echo -e "   Model: ${GREEN}$(system_profiler SPHardwareDataType | grep 'Model Name' | cut -d: -f2 | xargs)${NC}"
  else
    if [[ -f /proc/cpuinfo ]]; then
      CPU_MODEL=$(grep 'model name' /proc/cpuinfo | head -1 | cut -d: -f2 | xargs)
      echo -e "   CPU: ${GREEN}$CPU_MODEL${NC}"
    fi
  fi
  echo ""
}

# Show disk information
show_disk_info() {
  echo -e "${BLUE}4. Storage Information:${NC}"

  if [[ "$OS" == "macos" ]]; then
    echo -e "   ${YELLOW}Disks (via diskutil):${NC}"
    diskutil list | grep -E "(/dev/disk|TYPE)" | head -10
  else
    echo -e "   ${YELLOW}Block devices (via lsblk):${NC}"
    if command -v lsblk &>/dev/null; then
      lsblk | head -10
    else
      echo -e "   ${RED}lsblk not available${NC}"
    fi
  fi
  echo ""
}

# Show what the scripts would do
show_script_capabilities() {
  echo -e "${BLUE}5. Script Capabilities:${NC}"
  echo ""

  echo -e "${YELLOW}   install.sh (Universal Install Script):${NC}"
  echo -e "   • Auto-detects operating system ($OS)"

  if [[ "$OS" == "macos" ]]; then
    echo -e "   • Installs dependencies via Homebrew"
    echo -e "   • Creates macOS LaunchAgent service"
    echo -e "   • Manages service via launchctl"
  else
    echo -e "   • Installs dependencies via package manager"
    echo -e "   • Creates Linux systemd service"
    echo -e "   • Manages service via systemctl"
  fi

  echo -e "   • Builds and installs binary to /usr/local/bin"
  echo -e "   • Configures automatic startup"
  echo ""

  echo -e "${YELLOW}   test.sh (Universal Test Script):${NC}"
  echo -e "   • Auto-detects operating system ($OS)"
  echo -e "   • Checks dependencies and tools"
  echo -e "   • Shows system and disk information"
  echo -e "   • Builds and tests the exporter"
  echo -e "   • Validates metrics endpoint"
  echo -e "   • Provides usage instructions"
  echo ""
}

# Show sample build and test
show_sample_build() {
  echo -e "${BLUE}6. Sample Build Test:${NC}"

  cd "$(dirname "$0")/.."

  # Check if we can build
  if command -v go &>/dev/null; then
    echo -e "   ${YELLOW}Building exporter...${NC}"
    if go build -o disk-health-exporter-demo ./cmd/disk-health-exporter; then
      echo -e "   ✓ Build: ${GREEN}Success${NC}"

      # Quick test
      echo -e "   ${YELLOW}Testing binary...${NC}"
      if ./disk-health-exporter-demo --help >/dev/null 2>&1; then
        echo -e "   ✓ Binary: ${GREEN}Working${NC}"
      else
        echo -e "   ✓ Binary: ${GREEN}Created (--help not implemented)${NC}"
      fi

      # Clean up
      rm -f disk-health-exporter-demo
    else
      echo -e "   ✗ Build: ${RED}Failed${NC}"
    fi
  else
    echo -e "   ✗ Build: ${RED}Go not installed${NC}"
  fi
  echo ""
}

# Show usage instructions
show_usage() {
  echo -e "${BLUE}7. Usage Instructions:${NC}"
  echo ""
  echo -e "${YELLOW}   To test the exporter:${NC}"
  echo -e "   ./scripts/test.sh"
  echo ""
  echo -e "${YELLOW}   To install as a service:${NC}"
  echo -e "   ./scripts/install.sh"
  echo ""
  echo -e "${YELLOW}   To run manually:${NC}"
  echo -e "   go run ./cmd/disk-health-exporter"
  echo ""
  echo -e "${YELLOW}   To build manually:${NC}"
  echo -e "   go build -o disk-health-exporter ./cmd/disk-health-exporter"
  echo ""
  echo -e "${YELLOW}   Access metrics:${NC}"
  echo -e "   http://localhost:9100/metrics"
  echo ""
}

# Main demo function
main() {
  detect_os
  show_tools
  show_system_info
  show_disk_info
  show_script_capabilities
  show_sample_build
  show_usage

  echo -e "${GREEN}=== Demo Complete ===${NC}"
  echo ""
  echo -e "The universal scripts will handle installation and testing"
  echo -e "automatically based on your operating system (${GREEN}$OS${NC})."
  echo ""
}

# Run the demo
main "$@"
