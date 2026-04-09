# Electric Grid Monitor - Direct Commissioning Guide

## Overview

This guide covers commissioning your Sonoff S61s **directly from Raspberry Pi** using chip-tool, bypassing Google Home entirely.

## Prerequisites

1. **Raspberry Pi** (3B+ or 4 recommended) with Raspberry Pi OS
2. **Sonoff S61s** smart plug (Matter-enabled)
3. **WiFi network** (2.4GHz - Sonoff doesn't support 5GHz)
4. **chip-tool** installed on Raspberry Pi

## Step 1: Install chip-tool on Raspberry Pi

```bash
# Update system
sudo apt update && sudo apt upgrade -y

# Install dependencies (takes a while)
sudo apt install -y git gcc g++ python3 python3-pip pkg-config libssl-dev \
    libdbus-1-dev libglib2.0-dev libavahi-client-dev ninja-build \
    python3-venv python3-dev unzip libgirepository1.0-dev libcairo2-dev

# Clone connectedhomeip repository
git clone https://github.com/project-chip/connectedhomeip.git
cd connectedhomeip

# Setup build environment
source scripts/bootstrap.sh

# Build chip-tool (this takes 30-60 minutes on Raspberry Pi!)
./scripts/build/build_examples.py --target linux-arm64-chip-tool build

# Copy chip-tool to PATH
sudo cp out/linux-arm64-chip-tool/chip-tool /usr/local/bin/

# Verify installation
chip-tool --version
```

## Step 2: Prepare Sonoff S61s for Commissioning

1. **Plug in** the Sonoff S61s to power
2. **Reset to factory defaults** if previously commissioned:
   - Press and hold the button for **10-15 seconds**
   - LED will blink rapidly, then stop
   - Device is now in commissioning mode
3. **Wait 30 seconds** for device to boot and start broadcasting

## Step 3: Find Device Information

### Get Pairing Code from Device

Look at the Sonoff S61s:
- **QR Code sticker** on the device
- **Manual pairing code** printed nearby (format: `XXX-XXX-XXX`)

### Alternative: Use chip-tool to discover

```bash
# Discover commissionable devices
chip-tool discover commissionables

# Output will show something like:
# [TOO] Discovered Device:
# [TOO]   Hostname: xxx
# [TOO]   IP Address: 192.168.x.x
# [TOO]   Vendor ID: xxx
# [TOO]   Product ID: xxx
# [TOO]   Device Type: xxx
```

## Step 4: Commission the Device

You have two options:

### Option A: Commission with QR Code (Recommended)

```bash
# Use the QR code text from the device sticker
# Example: MT:-24J0AFN00SIQ663000
chip-tool pairing qrcode 0x1234 MT:XXXXXXXXXXX

# Replace MT:XXXXXXXXXXX with your actual QR code
```

### Option B: Commission with Manual Pairing Code

```bash
# Use the manual pairing code from the device sticker
# Example: 35792000079
chip-tool pairing code 0x1234 YOUR_PAIRING_CODE

# Replace YOUR_PAIRING_CODE with your actual code
```

### Option C: Commission with IP Address (If you know it)

```bash
# If device already connected to WiFi
chip-tool pairing onnetwork 0x1234 20202021 --ip YOUR_DEVICE_IP
```

**Note:** `0x1234` is the Node ID we're assigning to the device. You can use any hex value.

## Step 5: Verify Commissioning

```bash
# Test reading power consumption
chip-tool electricalmeasurement read active-power 0x1234 1

# Test reading voltage
chip-tool electricalmeasurement read rms-voltage 0x1234 1

# Test reading current
chip-tool electricalmeasurement read rms-current 0x1234 1
```

Expected output:
```
[TOO] Response: 1234
```

## Step 6: Find Device IP Address

```bash
# Method 1: Check your router's admin panel
# Method 2: Use arp-scan
sudo apt install arp-scan
sudo arp-scan --localnet | grep -i sonoff

# Method 3: Use chip-tool discovery
chip-tool discover commissionables

# Method 4: Check WiFi router connected devices
# Usually shows up as "ITEAD" or "Sonoff"
```

## Step 7: Configure Environment

Create your `.env` file:

```bash
# Matter Device Configuration
MATTER_NODE_ID=0x1234          # The node ID you used during commissioning
MATTER_ENDPOINT=1              # Usually 1 for Sonoff S61s
MATTER_IP=192.168.1.XXX        # Your Sonoff's IP address

# Collection Settings
COLLECTION_INTERVAL=30s
DATA_RETENTION_DAYS=365

# Database
DB_PATH=./data/grid_monitor.db

# API Server
API_PORT=8080
API_HOST=0.0.0.0

# chip-tool path
CHIP_TOOL_PATH=/usr/local/bin/chip-tool

# Logging
LOG_LEVEL=info
LOG_FILE=./logs/monitor.log
```

## Step 8: Install Electric Grid Monitor

### Option A: Automated Installation

```bash
# Clone repository
git clone <your-repo-url>
cd electric-grid-monitor

# Run automated setup
./scripts/install-raspberry-pi.sh

# Edit configuration
sudo nano /etc/electric-grid-monitor/.env

# Start service
sudo systemctl enable electric-grid-monitor
sudo systemctl start electric-grid-monitor
```

### Option B: Manual Installation

```bash
# Build application
go mod tidy
go build -o electric-grid-monitor cmd/monitor/main.go

# Create directories
mkdir -p data logs

# Create .env file
cp .env.example .env
nano .env  # Edit with your values

# Run directly
./electric-grid-monitor
```

### Option C: Docker

```bash
# Create .env
cp .env.example .env
nano .env

# Run with docker-compose
docker-compose up -d
```

## Step 9: Verify Everything Works

```bash
# Check service status
sudo systemctl status electric-grid-monitor

# View logs
sudo journalctl -u electric-grid-monitor -f

# Test API
curl http://localhost:8080/health
curl http://localhost:8080/api/v1/power/current
```

## Troubleshooting

### Commissioning Fails

**Problem:** Device not found
```bash
# Make sure device is in pairing mode
# LED should be blinking

# Try factory reset again
# Hold button for 15+ seconds
```

**Problem:** Wrong PIN
```bash
# Check the QR code/manual pairing code on device
# Make sure you're reading it correctly
```

### chip-tool Not Working

```bash
# Verify chip-tool installation
which chip-tool
chip-tool --version

# Check if binary exists
ls -la /usr/local/bin/chip-tool
```

### Device Responds to chip-tool but not to App

```bash
# Make sure you're using the same Node ID
# Check if device IP is correct

# Test manually:
chip-tool electricalmeasurement read active-power 0x1234 1 --ip DEVICE_IP
```

### Service Won't Start

```bash
# Check logs
sudo journalctl -u electric-grid-monitor -n 50

# Verify .env file exists and is readable
sudo cat /etc/electric-grid-monitor/.env

# Test binary manually
sudo /usr/local/bin/electric-grid-monitor
```

## Complete Setup Checklist

- [ ] Installed chip-tool on Raspberry Pi
- [ ] Reset Sonoff S61s to factory defaults
- [ ] Retrieved QR code or manual pairing code from device
- [ ] Commissioned device with chip-tool
- [ ] Verified device responds to chip-tool commands
- [ ] Found device IP address
- [ ] Installed Electric Grid Monitor
- [ ] Created and configured .env file
- [ ] Started the service
- [ ] Tested API endpoints
- [ ] Setup Grafana (optional)

## Quick Reference Commands

```bash
# Commission device
chip-tool pairing qrcode 0x1234 MT:XXXXXXXXXXX

# Test device
chip-tool electricalmeasurement read active-power 0x1234 1

# Check service
sudo systemctl status electric-grid-monitor

# View logs
sudo journalctl -u electric-grid-monitor -f

# Test API
curl http://localhost:8080/api/v1/power/current
```

## Next Steps

After successful setup:
1. **Setup Grafana** for visualization (see main README)
2. **Configure data retention** in .env (default: 365 days)
3. **Adjust collection interval** (default: 30 seconds)
4. **Add more devices** (each with unique Node ID)
