#!/bin/bash

# Electric Grid Monitor - Raspberry Pi Setup Script
# This script automates the installation on Raspberry Pi

set -e

echo "=========================================="
echo "Electric Grid Monitor - Raspberry Pi Setup"
echo "=========================================="
echo ""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if running as root
if [ "$EUID" -eq 0 ]; then 
   echo -e "${RED}Please do not run as root${NC}"
   exit 1
fi

# Variables
INSTALL_DIR="/opt/electric-grid-monitor"
CONFIG_DIR="/etc/electric-grid-monitor"
DATA_DIR="/var/lib/electric-grid-monitor/data"
LOG_DIR="/var/log/electric-grid-monitor"
USER="${SUDO_USER:-$USER}"

echo "Step 1: Installing system dependencies..."
sudo apt update
sudo apt install -y git gcc g++ python3 python3-pip pkg-config libssl-dev \
    libdbus-1-dev libglib2.0-dev libavahi-client-dev ninja-build \
    python3-venv python3-dev unzip libgirepository1.0-dev libcairo2-dev \
    sqlite3

echo ""
echo "Step 2: Installing Go (if not present)..."
if ! command -v go &> /dev/null; then
    GO_VERSION="1.21.5"
    echo "Installing Go ${GO_VERSION}..."
    wget https://golang.org/dl/go${GO_VERSION}.linux-arm64.tar.gz
    sudo tar -C /usr/local -xzf go${GO_VERSION}.linux-arm64.tar.gz
    rm go${GO_VERSION}.linux-arm64.tar.gz
    echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
    export PATH=$PATH:/usr/local/go/bin
else
    echo "Go is already installed: $(go version)"
fi

echo ""
echo "Step 3: Building and installing Electric Grid Monitor..."

# Create directories
sudo mkdir -p "$INSTALL_DIR"
sudo mkdir -p "$CONFIG_DIR"
sudo mkdir -p "$DATA_DIR"
sudo mkdir -p "$LOG_DIR"

# Build the application
go mod download
go build -o /tmp/electric-grid-monitor cmd/monitor/main.go

# Install binary
sudo cp /tmp/electric-grid-monitor /usr/local/bin/
sudo chmod +x /usr/local/bin/electric-grid-monitor

echo ""
echo "Step 4: Installing systemd service..."
sudo cp systemd/electric-grid-monitor.service /etc/systemd/system/
sudo sed -i "s|WorkingDirectory=.*|WorkingDirectory=$DATA_DIR|" /etc/systemd/system/electric-grid-monitor.service
sudo systemctl daemon-reload

echo ""
echo "Step 5: Creating default configuration..."
if [ ! -f "$CONFIG_DIR/.env" ]; then
    sudo tee "$CONFIG_DIR/.env" > /dev/null <<EOF
# Matter Device Configuration
MATTER_NODE_ID=0x1234
MATTER_ENDPOINT=1
MATTER_IP=192.168.1.100

# Collection Settings
COLLECTION_INTERVAL=30s
DATA_RETENTION_DAYS=365

# Database
DB_PATH=$DATA_DIR/grid_monitor.db

# API Server
API_PORT=8080
API_HOST=0.0.0.0

# chip-tool path
CHIP_TOOL_PATH=/usr/local/bin/chip-tool

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
echo "Step 6: Setting permissions..."
sudo chown -R "$USER:$USER" "$DATA_DIR"
sudo chown -R "$USER:$USER" "$LOG_DIR"
sudo chmod 755 "$CONFIG_DIR"
sudo chmod 644 "$CONFIG_DIR/.env"

echo ""
echo "=========================================="
echo -e "${GREEN}Installation Complete!${NC}"
echo "=========================================="
echo ""
echo "Next steps:"
echo ""
echo "1. Edit the configuration file:"
echo "   sudo nano $CONFIG_DIR/.env"
echo ""
echo "2. Update these settings:"
echo "   - MATTER_NODE_ID (your commissioned device ID)"
echo "   - MATTER_IP (your Sonoff S61s IP address)"
echo ""
echo "3. Enable and start the service:"
echo "   sudo systemctl enable electric-grid-monitor"
echo "   sudo systemctl start electric-grid-monitor"
echo ""
echo "4. Check service status:"
echo "   sudo systemctl status electric-grid-monitor"
echo "   sudo journalctl -u electric-grid-monitor -f"
echo ""
echo "5. Test the API:"
echo "   curl http://localhost:8080/health"
echo ""
echo "The application will collect data every 30 seconds (configurable)"
echo "and store it in SQLite database at: $DATA_DIR/grid_monitor.db"
echo ""
