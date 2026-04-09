#!/bin/bash

# Helper script to discover and commission Sonoff S61s from Raspberry Pi

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Sonoff S61s Discovery & Commissioning${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Check if chip-tool is installed
if ! command -v chip-tool &> /dev/null; then
    echo -e "${RED}Error: chip-tool not found!${NC}"
    echo "Please install chip-tool first:"
    echo "  See COMMISSIONING.md for instructions"
    exit 1
fi

echo -e "${GREEN}✓ chip-tool found${NC}"
echo ""

# Function to discover devices
discover_devices() {
    echo -e "${BLUE}Scanning for Matter devices...${NC}"
    echo "Make sure your Sonoff S61s is plugged in and in pairing mode"
    echo "(LED should be blinking)"
    echo ""
    
    chip-tool discover commissionables
}

# Function to commission with QR code
commission_qr() {
    echo ""
    echo -e "${BLUE}Commission with QR Code${NC}"
    echo "Find the QR code sticker on your Sonoff S61s device"
    echo "It looks like: MT:-24J0AFN00SIQ663000"
    echo ""
    
    read -p "Enter QR code (MT:...): " qr_code
    read -p "Enter Node ID to assign (default: 0x1234): " node_id
    node_id=${node_id:-0x1234}
    
    echo ""
    echo -e "${YELLOW}Commissioning device...${NC}"
    chip-tool pairing qrcode $node_id "$qr_code"
    
    if [ $? -eq 0 ]; then
        echo ""
        echo -e "${GREEN}✓ Commissioning successful!${NC}"
        echo "Node ID: $node_id"
        
        # Test the device
        echo ""
        echo -e "${BLUE}Testing device connection...${NC}"
        chip-tool electricalmeasurement read active-power $node_id 1
    else
        echo -e "${RED}✗ Commissioning failed${NC}"
        exit 1
    fi
}

# Function to commission with manual code
commission_manual() {
    echo ""
    echo -e "${BLUE}Commission with Manual Pairing Code${NC}"
    echo "Find the manual pairing code on your Sonoff S61s device"
    echo "It looks like: 357-920-000-79 or 35792000079"
    echo ""
    
    read -p "Enter manual pairing code: " manual_code
    read -p "Enter Node ID to assign (default: 0x1234): " node_id
    node_id=${node_id:-0x1234}
    
    # Remove dashes if present
    manual_code=$(echo $manual_code | tr -d '-')
    
    echo ""
    echo -e "${YELLOW}Commissioning device...${NC}"
    chip-tool pairing code $node_id $manual_code
    
    if [ $? -eq 0 ]; then
        echo ""
        echo -e "${GREEN}✓ Commissioning successful!${NC}"
        echo "Node ID: $node_id"
        
        # Test the device
        echo ""
        echo -e "${BLUE}Testing device connection...${NC}"
        chip-tool electricalmeasurement read active-power $node_id 1
    else
        echo -e "${RED}✗ Commissioning failed${NC}"
        exit 1
    fi
}

# Function to test commissioned device
test_device() {
    echo ""
    read -p "Enter Node ID (default: 0x1234): " node_id
    node_id=${node_id:-0x1234}
    
    echo ""
    echo -e "${BLUE}Testing device $node_id...${NC}"
    
    echo "Reading power..."
    chip-tool electricalmeasurement read active-power $node_id 1
    
    echo ""
    echo "Reading voltage..."
    chip-tool electricalmeasurement read rms-voltage $node_id 1
    
    echo ""
    echo "Reading current..."
    chip-tool electricalmeasurement read rms-current $node_id 1
}

# Function to find device IP
find_ip() {
    echo ""
    echo -e "${BLUE}Finding device IP address...${NC}"
    echo ""
    
    echo "Method 1: Using arp-scan"
    if command -v arp-scan &> /dev/null; then
        sudo arp-scan --localnet | grep -i -E "(sonoff|itead)" || echo "No Sonoff devices found with arp-scan"
    else
        echo "arp-scan not installed. Install with: sudo apt install arp-scan"
    fi
    
    echo ""
    echo "Method 2: Check router"
    echo "Log into your router's admin panel and look for connected devices"
    echo "Look for device named 'ITEAD' or 'Sonoff'"
    
    echo ""
    echo "Method 3: Check this computer's ARP table"
    ip neigh show | grep -i -E "(192\.168|10\.|172\.)" | head -20
}

# Main menu
show_menu() {
    echo ""
    echo "What would you like to do?"
    echo ""
    echo "1) Discover commissionable devices"
    echo "2) Commission with QR code"
    echo "3) Commission with manual pairing code"
    echo "4) Test commissioned device"
    echo "5) Find device IP address"
    echo "6) Exit"
    echo ""
}

# Main loop
while true; do
    show_menu
    read -p "Select option (1-6): " choice
    
    case $choice in
        1)
            discover_devices
            ;;
        2)
            commission_qr
            ;;
        3)
            commission_manual
            ;;
        4)
            test_device
            ;;
        5)
            find_ip
            ;;
        6)
            echo "Goodbye!"
            exit 0
            ;;
        *)
            echo -e "${RED}Invalid option${NC}"
            ;;
    esac
done
