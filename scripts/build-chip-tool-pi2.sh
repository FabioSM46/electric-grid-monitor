#!/bin/bash
# Build chip-tool for Raspberry Pi 2 (ARMv7)

set -e

echo "Installing dependencies for chip-tool build..."
echo ""
echo "⚠️  WARNING: This build takes 30-60 minutes!"
echo "If running via SSH, use screen or tmux to prevent timeout:"
echo "   screen -S chipbuild"
echo "   ./build-chip-tool-pi2.sh"
echo "   # Press Ctrl+A then D to detach"
echo ""
read -p "Press Enter to continue or Ctrl+C to cancel..."
echo ""
sudo apt-get update
sudo apt-get install -y git gcc g++ python3 python3-pip pkg-config libssl-dev \
    libdbus-1-dev libglib2.0-dev libavahi-client-dev ninja-build \
    python3-venv python3-dev unzip libgirepository1.0-dev libcairo2-dev

echo "Installing depot_tools..."
cd ~
if [ ! -d "depot_tools" ]; then
    git clone https://chromium.googlesource.com/chromium/tools/depot_tools.git
fi
export PATH="$HOME/depot_tools:$PATH"

echo "Cloning connectedhomeip..."
if [ ! -d "connectedhomeip" ]; then
    git clone https://github.com/project-chip/connectedhomeip.git
fi
cd connectedhomeip

echo "Setting up Python environment..."
python3 -m venv venv
source venv/bin/activate
pip install coloredlogs click

echo "Updating submodules..."
./scripts/checkout_submodules.py --shallow --platform linux

echo "Bootstrapping build environment..."
source scripts/bootstrap.sh

echo "Building chip-tool (this will take 30-60 minutes on Pi 2)..."
./scripts/build/build_examples.py --target linux-arm-chip-tool build

echo "Installing chip-tool..."
sudo cp out/linux-arm-chip-tool/chip-tool /usr/local/bin/
sudo chmod +x /usr/local/bin/chip-tool

echo "chip-tool installed successfully!"
chip-tool --version
