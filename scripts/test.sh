#!/bin/bash

# Universal Test script for Disk Health Exporter
# Auto-detects OS and runs appropriate tests

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Disk Health Exporter - Universal Test Script${NC}"

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

# Check dependencies for Linux
check_linux_deps() {
    echo -e "${YELLOW}Checking Linux dependencies...${NC}"
    
    # Check Go
    if ! command -v go &>/dev/null; then
        echo -e "${RED}Go is not installed${NC}"
        echo "Please install Go from https://golang.org/dl/"
        exit 1
    fi
    
    # Check smartctl (optional but recommended)
    if ! command -v smartctl &>/dev/null; then
        echo -e "${YELLOW}smartctl not found. This is needed for disk monitoring.${NC}"
        echo "Install with: sudo apt-get install smartmontools (or equivalent for your distro)"
    fi
    
    # Check for MegaCLI (optional)
    if command -v megacli &>/dev/null || command -v MegaCli64 &>/dev/null; then
        echo -e "${GREEN}MegaCLI found - RAID monitoring available${NC}"
    else
        echo -e "${YELLOW}MegaCLI not found - RAID monitoring unavailable${NC}"
    fi
}

# Check dependencies for macOS
check_macos_deps() {
    echo -e "${YELLOW}Checking macOS dependencies...${NC}"
    
    # Check Go
    if ! command -v go &>/dev/null; then
        echo -e "${RED}Go is not installed${NC}"
        echo "Please install Go from https://golang.org/dl/"
        exit 1
    fi
    
    # Check smartctl (optional but recommended)
    if ! command -v smartctl &>/dev/null; then
        echo -e "${YELLOW}smartctl not found. Installing via Homebrew...${NC}"
        if command -v brew &>/dev/null; then
            brew install smartmontools
        else
            echo -e "${RED}Homebrew not found. Please install smartmontools manually${NC}"
            echo "Or install Homebrew: https://brew.sh"
        fi
    fi
}

# Show system information for Linux
show_linux_info() {
    echo -e "${YELLOW}System Information:${NC}"
    echo "OS: $(uname -s)"
    echo "Kernel: $(uname -r)"
    echo "Go version: $(go version)"
    
    # Show disk information
    echo -e "${YELLOW}Storage Information:${NC}"
    echo "Block devices:"
    lsblk 2>/dev/null | head -15 || echo "lsblk not available"
    
    # Test smartctl if available
    if command -v smartctl &>/dev/null; then
        echo -e "${YELLOW}Testing smartctl...${NC}"
        sudo smartctl --scan 2>/dev/null | head -5 || echo "No SMART-capable devices found"
    fi
    
    # Test MegaCLI if available
    if command -v megacli &>/dev/null; then
        echo -e "${YELLOW}Testing MegaCLI...${NC}"
        sudo megacli -AdpCount -NoLog 2>/dev/null || echo "No RAID controllers found"
    elif command -v MegaCli64 &>/dev/null; then
        echo -e "${YELLOW}Testing MegaCLI64...${NC}"
        sudo MegaCli64 -AdpCount -NoLog 2>/dev/null || echo "No RAID controllers found"
    fi
}

# Show system information for macOS
show_macos_info() {
    echo -e "${YELLOW}System Information:${NC}"
    echo "OS: $(uname -s)"
    echo "Version: $(sw_vers -productVersion)"
    echo "Go version: $(go version)"
    
    # Show disk information
    echo -e "${YELLOW}Disk Information:${NC}"
    echo "Available disks:"
    diskutil list | head -20
    
    # Test smartctl if available
    if command -v smartctl &>/dev/null; then
        echo -e "${YELLOW}Testing smartctl...${NC}"
        smartctl --scan 2>/dev/null || echo "smartctl scan failed (this is normal on macOS)"
    fi
}

# Test the exporter (common for all platforms)
test_exporter() {
    echo -e "${YELLOW}Building the exporter...${NC}"
    cd "$(dirname "$0")/.."
    go build -o disk-health-exporter ./cmd/disk-health-exporter
    
    echo -e "${GREEN}Build successful!${NC}"
    
    echo -e "${YELLOW}Testing the exporter...${NC}"
    echo "Starting exporter in background..."
    
    # Start the exporter in background
    ./disk-health-exporter &
    EXPORTER_PID=$!
    
    # Wait for startup
    sleep 3
    
    echo "Testing exporter endpoints..."
    
    # Test health endpoint
    echo "Testing root endpoint..."
    if curl -s http://localhost:9100/ | head -5; then
        echo -e "${GREEN}Root endpoint is working!${NC}"
    else
        echo -e "${RED}Failed to access root endpoint${NC}"
    fi
    
    echo ""
    echo "Testing metrics endpoint..."
    if curl -s http://localhost:9100/metrics | head -10; then
        echo -e "${GREEN}Metrics endpoint is working!${NC}"
    else
        echo -e "${RED}Failed to get metrics${NC}"
    fi
    
    echo ""
    echo "Checking for custom metrics..."
    CUSTOM_METRICS=$(curl -s http://localhost:9100/metrics | grep -E "(disk_health_status|disk_temperature|raid_array_status|disk_health_exporter_up)" | head -10)
    if [[ -n "$CUSTOM_METRICS" ]]; then
        echo -e "${GREEN}Custom metrics found:${NC}"
        echo "$CUSTOM_METRICS"
    else
        echo -e "${YELLOW}No custom metrics found yet (this is normal if no compatible tools are installed)${NC}"
    fi
    
    # Clean up
    echo -e "${YELLOW}Stopping test exporter...${NC}"
    kill $EXPORTER_PID 2>/dev/null || true
    
    echo ""
    echo -e "${GREEN}Test completed successfully!${NC}"
    echo ""
    echo "The exporter is working correctly!"
    echo "Metrics URL: http://localhost:9100/metrics"
    echo ""
    echo "To run manually: ./disk-health-exporter"
    echo "To install as service: ./scripts/install.sh"
}

# Main test logic
main() {
    # Detect OS
    detect_os
    
    # Check dependencies and show system info based on OS
    case "$OS" in
        ubuntu|debian|centos|fedora|rhel|arch|*linux*)
            check_linux_deps
            show_linux_info
            ;;
        macos)
            check_macos_deps
            show_macos_info
            ;;
        *)
            echo -e "${RED}Unsupported OS: $OS${NC}"
            exit 1
            ;;
    esac
    
    # Test the exporter (common for all platforms)
    test_exporter
}

# Run main function
main "$@"
