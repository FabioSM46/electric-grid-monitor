#!/bin/bash

# Electric Grid Monitor - Raspberry Pi Setup Script
# Optimized for Raspberry Pi 4/5 (ARM64) with native chip-tool building

set -e

echo "=========================================="
echo "Electric Grid Monitor - Raspberry Pi Setup"
echo "=========================================="
echo ""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Check if running as root
if [ "$EUID" -eq 0 ]; then 
   echo -e "${RED}Please do not run as root (use sudo for commands that need it)${NC}"
   exit 1
fi

# Detect architecture and Pi model
echo "Detecting system architecture..."
ARCH=$(uname -m)
PI_MODEL=""
TOTAL_RAM_MB=$(free -m | awk '/^Mem:/{print $2}')

# Detect Pi model
if [ -f /proc/device-tree/model ]; then
    PI_MODEL=$(cat /proc/device-tree/model | tr '\0' '\n')
elif [ -f /sys/firmware/devicetree/base/model ]; then
    PI_MODEL=$(cat /sys/firmware/devicetree/base/model | tr '\0' '\n')
fi

echo -e "${BLUE}Architecture: $ARCH${NC}"
echo -e "${BLUE}Pi Model: $PI_MODEL${NC}"
echo -e "${BLUE}Total RAM: ${TOTAL_RAM_MB}MB${NC}"

# Validate architecture
if [ "$ARCH" != "aarch64" ] && [ "$ARCH" != "arm64" ]; then
    echo -e "${YELLOW}Warning: Architecture $ARCH detected.${NC}"
    echo "This script is optimized for ARM64 (Raspberry Pi 4/5)."
    echo "For 32-bit ARM (armv7l), consider using the 32-bit version."
    read -p "Continue anyway? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

# Variables
INSTALL_DIR="/opt/electric-grid-monitor"
CONFIG_DIR="/etc/electric-grid-monitor"
DATA_DIR="/var/lib/electric-grid-monitor/data"
LOG_DIR="/var/log/electric-grid-monitor"
USER="${SUDO_USER:-$USER}"
CHIP_TOOL_PATH="/usr/local/bin/chip-tool"

# Go version (updated to latest stable)
GO_VERSION="1.23.4"
GO_ARCH="arm64"

echo ""
echo "Step 1: Updating package lists and installing system dependencies..."
sudo apt update

# Core dependencies for both Pi 4 and Pi 5
sudo apt install -y \
    git gcc g++ python3 python3-pip python3-venv python3-dev \
    pkg-config libssl-dev libdbus-1-dev libglib2.0-dev \
    libavahi-client-dev libavahi-common-dev ninja-build \
    build-essential cmake automake autoconf libtool \
    unzip zip wget curl sqlite3 \
    libgirepository1.0-dev libcairo2-dev libreadline-dev \
    libical-dev libconfig-dev libmosquitto-dev

# Pi 5 specific optimizations
if echo "$PI_MODEL" | grep -q "Pi 5"; then
    echo -e "${GREEN}Raspberry Pi 5 detected - applying optimizations...${NC}"
    # Additional Pi 5 optimizations if needed
    sudo apt install -y libraspberrypi-dev 2>/dev/null || true
fi

echo ""
echo "Step 2: Installing/Updating Go..."
if command -v go &> /dev/null; then
    CURRENT_GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    echo "Current Go version: $CURRENT_GO_VERSION"
    
    # Check if update is needed (simple major.minor comparison)
    CURRENT_MAJOR=$(echo $CURRENT_GO_VERSION | cut -d. -f1)
    CURRENT_MINOR=$(echo $CURRENT_GO_VERSION | cut -d. -f2)
    TARGET_MAJOR=$(echo $GO_VERSION | cut -d. -f1)
    TARGET_MINOR=$(echo $GO_VERSION | cut -d. -f2)
    
    if [ "$CURRENT_MAJOR" -lt "$TARGET_MAJOR" ] || ([ "$CURRENT_MAJOR" -eq "$TARGET_MAJOR" ] && [ "$CURRENT_MINOR" -lt "$TARGET_MINOR" ]); then
        echo "Updating Go to $GO_VERSION..."
        sudo rm -rf /usr/local/go
        wget -q --show-progress "https://golang.org/dl/go${GO_VERSION}.linux-${GO_ARCH}.tar.gz"
        sudo tar -C /usr/local -xzf "go${GO_VERSION}.linux-${GO_ARCH}.tar.gz"
        rm "go${GO_VERSION}.linux-${GO_ARCH}.tar.gz"
    else
        echo "Go version is up to date."
    fi
else
    echo "Installing Go ${GO_VERSION}..."
    wget -q --show-progress "https://golang.org/dl/go${GO_VERSION}.linux-${GO_ARCH}.tar.gz"
    sudo tar -C /usr/local -xzf "go${GO_VERSION}.linux-${GO_ARCH}.tar.gz"
    rm "go${GO_VERSION}.linux-${GO_ARCH}.tar.gz"
fi

# Add Go to PATH if not already there
if ! grep -q "/usr/local/go/bin" ~/.bashrc; then
    echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
fi
export PATH=$PATH:/usr/local/go/bin

# Verify Go installation
echo -e "${GREEN}Go installed: $(go version)${NC}"

echo ""
echo "Step 3: Building and installing chip-tool (Matter SDK)..."

# Check if chip-tool already exists and is working
if [ -f "$CHIP_TOOL_PATH" ] && $CHIP_TOOL_PATH --help &> /dev/null; then
    echo -e "${GREEN}chip-tool already installed at $CHIP_TOOL_PATH${NC}"
    echo "Skipping chip-tool build. To rebuild, delete $CHIP_TOOL_PATH first."
else
    # Determine build approach based on RAM
    if [ "$TOTAL_RAM_MB" -lt 4096 ]; then
        echo -e "${YELLOW}Warning: Less than 4GB RAM detected. Building chip-tool may be slow or fail.${NC}"
        echo "Consider increasing swap space or using a pre-built binary."
        read -p "Continue building chip-tool? (y/N) " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            echo "Skipping chip-tool build. Please install it manually."
        fi
    fi
    
    echo "Cloning connectedhomeip repository..."
    CHIP_TEMP_DIR=$(mktemp -d)
    cd "$CHIP_TEMP_DIR"
    
    # Shallow clone for faster download
    git clone --depth 1 --branch v1.3.0.0 https://github.com/project-chip/connectedhomeip.git
    cd connectedhomeip
    
    echo "Bootstrapping Matter SDK..."
    ./scripts/checkout_submodules.py --shallow --platform linux
    
    # Install build dependencies
    sudo ./scripts/setup/ubuntu.sh
    
    # Bootstrap
    source scripts/activate.sh
    
    echo "Building chip-tool for ARM64..."
    # Use gn to build just chip-tool
    gn gen out/debug --args='chip_build_tool=true'
    ninja -C out/debug chip-tool
    
    # Install chip-tool
    sudo cp out/debug/chip-tool "$CHIP_TOOL_PATH"
    sudo chmod +x "$CHIP_TOOL_PATH"
    
    # Cleanup
    cd "$HOME"
    rm -rf "$CHIP_TEMP_DIR"
    
    echo -e "${GREEN}chip-tool built and installed successfully!${NC}"
fi

echo ""
echo "Step 4: Building Electric Grid Monitor..."

# Create directories
sudo mkdir -p "$INSTALL_DIR"
sudo mkdir -p "$CONFIG_DIR"
sudo mkdir -p "$DATA_DIR"
sudo mkdir -p "$LOG_DIR"

# Navigate to project root (assume script is in scripts/)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
cd "$PROJECT_ROOT"

# Download Go dependencies
echo "Downloading Go dependencies..."
go mod download

# Build for ARM64 with optimizations
export GOOS=linux
export GOARCH=arm64
export CGO_ENABLED=1

echo "Building Go application for ARM64..."
go build -ldflags="-s -w" -o /tmp/electric-grid-monitor cmd/monitor/main.go

# Install binary
sudo cp /tmp/electric-grid-monitor /usr/local/bin/
sudo chmod +x /usr/local/bin/electric-grid-monitor

echo -e "${GREEN}Electric Grid Monitor built successfully!${NC}"

echo ""
echo "Step 5: Installing systemd service..."

# Check if systemd directory exists
if [ -d "$PROJECT_ROOT/systemd" ]; then
    sudo cp "$PROJECT_ROOT/systemd/electric-grid-monitor.service" /etc/systemd/system/
else
    # Create service file inline
    sudo tee /etc/systemd/system/electric-grid-monitor.service > /dev/null <<EOF
[Unit]
Description=Electric Grid Monitor
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=/var/lib/electric-grid-monitor/data
Environment="HOME=/root"
Environment="PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
ExecStart=/usr/local/bin/electric-grid-monitor
Restart=always
RestartSec=10

# Logging
StandardOutput=append:/var/log/electric-grid-monitor/output.log
StandardError=append:/var/log/electric-grid-monitor/error.log

[Install]
WantedBy=multi-user.target
EOF
fi

# Update service file with correct paths
sudo sed -i "s|WorkingDirectory=.*|WorkingDirectory=$DATA_DIR|" /etc/systemd/system/electric-grid-monitor.service
sudo systemctl daemon-reload

echo ""
echo "Step 6: Creating configuration..."

if [ ! -f "$CONFIG_DIR/.env" ]; then
    sudo tee "$CONFIG_DIR/.env" > /dev/null <<EOF
# Electric Grid Monitor Configuration
# Generated on $(date)

# Matter Device Configuration
MATTER_NODE_ID=0x1234
MATTER_ENDPOINT=1
MATTER_IP=192.168.1.100
MATTER_PORT=5540

# Collection Settings
COLLECTION_INTERVAL=30s
DATA_RETENTION_DAYS=365

# Database
DB_PATH=$DATA_DIR/grid_monitor.db

# API Server
API_PORT=8080
API_HOST=0.0.0.0

# chip-tool path
CHIP_TOOL_PATH=$CHIP_TOOL_PATH

# Logging
LOG_LEVEL=info
LOG_FILE=$LOG_DIR/monitor.log
EOF
    echo -e "${GREEN}Created default config at $CONFIG_DIR/.env${NC}"
    echo -e "${YELLOW}IMPORTANT: Edit this file with your device settings!${NC}"
else
    echo "Configuration already exists at $CONFIG_DIR/.env"
fi

echo ""
echo "Step 7: Setting permissions..."
sudo chown -R root:root "$DATA_DIR"
sudo chown -R root:root "$LOG_DIR"
sudo chmod 755 "$DATA_DIR"
sudo chmod 755 "$LOG_DIR"
sudo chmod 755 "$CONFIG_DIR"
sudo chmod 644 "$CONFIG_DIR/.env"

# Add user to gpio group for hardware access if needed
if ! groups "$USER" | grep -q '\bgpio\b'; then
    echo "Adding $USER to gpio group..."
    sudo usermod -a -G gpio "$USER"
fi

echo ""
echo "=========================================="
echo -e "${GREEN}Installation Complete!${NC}"
echo "=========================================="
echo ""
echo "Configuration:"
echo "  - Config file: $CONFIG_DIR/.env"
echo "  - Data directory: $DATA_DIR"
echo "  - Log directory: $LOG_DIR"
echo "  - Binary: /usr/local/bin/electric-grid-monitor"
echo "  - chip-tool: $CHIP_TOOL_PATH"
echo ""
echo "Next steps:"
echo ""
echo "1. Edit the configuration file:"
echo "   sudo nano $CONFIG_DIR/.env"
echo ""
echo "2. Update these settings:"
echo "   - MATTER_NODE_ID (your commissioned device ID)"
echo "   - MATTER_IP (your Sonoff S61s IP address)"
echo "   - MATTER_PORT (usually 5540)"
echo ""
echo "3. Commission your Matter device (if not already done):"
echo "   $CHIP_TOOL_PATH pairing onnetwork <node-id> <pin-code>"
echo ""
echo "4. Enable and start the service:"
echo "   sudo systemctl enable electric-grid-monitor"
echo "   sudo systemctl start electric-grid-monitor"
echo ""
echo "5. Check service status:"
echo "   sudo systemctl status electric-grid-monitor"
echo "   sudo journalctl -u electric-grid-monitor -f"
echo ""
echo "6. Test the API:"
echo "   curl http://localhost:8080/health"
echo ""
echo "The application will collect data every 30 seconds (configurable)"
echo "and store it in SQLite database at: $DATA_DIR/grid_monitor.db"
echo ""
