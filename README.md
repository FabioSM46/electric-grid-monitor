# Electric Grid Monitor

Go application to monitor Sonoff S61s power consumption using Matter protocol (chip-tool).

## Architecture

```
┌─────────────────────────────────────────────┐
│           Electric Grid Monitor             │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  │
│  │ Config   │  │Collector │  │  SQLite  │  │
│  │ (.env)   │  │(chip-tool│  │ Storage  │  │
│  └──────────┘  │ wrapper) │  └──────────┘  │
│                └────┬─────┘       │        │
│                     │             │        │
│                ┌────▼─────┐       │        │
│                │ Sonoff   │       │        │
│                │ S61s     │       │        │
│                └──────────┘       │        │
│                                   ▼        │
│                          ┌──────────────┐  │
│                          │  HTTP API    │  │
│                          │  (Grafana)   │  │
│                          └──────────────┘  │
└─────────────────────────────────────────────┘
```

## Prerequisites

1. **chip-tool** installed on Raspberry Pi
2. **Sonoff S61s** already commissioned to your network (Google Home + eWeLink)
3. **Go 1.21+** installed

## Installation

### Step 1: Install chip-tool on Raspberry Pi

```bash
# Update system
sudo apt update && sudo apt upgrade -y

# Install dependencies
sudo apt install -y git gcc g++ python3 python3-pip pkg-config libssl-dev libdbus-1-dev \
    libglib2.0-dev libavahi-client-dev ninja-build python3-venv python3-dev \
    python3-pip unzip libgirepository1.0-dev libcairo2-dev

# Clone connectedhomeip repository
git clone https://github.com/project-chip/connectedhomeip.git
cd connectedhomeip

# Setup build environment
source scripts/bootstrap.sh

# Build chip-tool
./scripts/build/build_examples.py --target linux-arm64-chip-tool build

# Copy chip-tool to PATH
sudo cp out/linux-arm64-chip-tool/chip-tool /usr/local/bin/

# Verify installation
chip-tool --version
```

### Step 2: Commission Your Device (if not already done)

If your device is already commissioned, skip to Step 3.

```bash
# Commission using QR code
chip-tool pairing qrcode 0x1234 MT:-24J0AFN00SIQ663000

# OR commission using manual pairing code
chip-tool pairing code 0x1234 35792000079

# Test connection - read power
chip-tool electricalmeasurement read active-power 0x1234 1 --ip YOUR_DEVICE_IP
```

### Step 3: Install Electric Grid Monitor

```bash
# Clone repository
git clone <repository-url>
cd electric-grid-monitor

# Create directories
mkdir -p data logs

# Copy and configure environment
cp .env.example .env
nano .env  # Edit with your settings
```

### Step 4: Configure .env File

Edit `.env` with your specific values:

```bash
# Matter Device Configuration
MATTER_NODE_ID=0x1234          # Your commissioned node ID
MATTER_ENDPOINT=1              # Usually 1 for Sonoff S61s
MATTER_IP=192.168.1.100        # Your Sonoff S61s IP address

# Collection Settings
COLLECTION_INTERVAL=30s        # How often to collect data
DATA_RETENTION_DAYS=365        # Keep data for 1 year

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

### Step 5: Build and Run

```bash
# Build
make build

# Run in development mode
make dev

# OR run the binary
./build/electric-grid-monitor
```

### Step 6: Install as System Service (Production)

```bash
# Install binary and service
sudo make install

# Copy configuration to system location
sudo mkdir -p /etc/electric-grid-monitor
sudo cp .env /etc/electric-grid-monitor/
sudo nano /etc/electric-grid-monitor/.env  # Edit with correct paths

# Update service file to use system config
sudo sed -i 's|WorkingDirectory=.*|WorkingDirectory=/var/lib/electric-grid-monitor|' /etc/systemd/system/electric-grid-monitor.service

# Enable and start service
sudo systemctl daemon-reload
sudo systemctl enable electric-grid-monitor
sudo systemctl start electric-grid-monitor

# Check status
sudo systemctl status electric-grid-monitor
sudo journalctl -u electric-grid-monitor -f
```

## Docker Deployment (Alternative)

```bash
# Build and run with docker-compose
docker-compose up -d

# View logs
docker-compose logs -f grid-monitor

# Access Grafana at http://localhost:3000 (admin/admin)
```

## API Endpoints

- `GET /health` - Health check
- `GET /api/v1/power/current` - Current power consumption (live from device)
- `GET /api/v1/history?from=2024-01-01&to=2024-01-31` - Historical data
- `GET /api/v1/stats/daily?date=2024-01-01` - Daily statistics

### Example API Usage

```bash
# Get current power
curl http://localhost:8080/api/v1/power/current

# Get history for last 24 hours (default)
curl http://localhost:8080/api/v1/history

# Get history for specific date range
curl "http://localhost:8080/api/v1/history?from=2024-01-01&to=2024-01-31"

# Get daily stats
curl "http://localhost:8080/api/v1/stats/daily?date=2024-01-01"
```

## Database Schema

Table: `power_readings`
- `id` INTEGER PRIMARY KEY
- `timestamp` DATETIME
- `power_w` REAL (Power in Watts)
- `voltage_v` REAL (Voltage in Volts)
- `current_a` REAL (Current in Amps)
- `energy_wh` REAL (Energy in Watt-hours)

## Grafana Setup

### Option 1: Docker Compose (Recommended)
Grafana is included in docker-compose.yml with pre-configured dashboard.

### Option 2: Manual Setup

1. **Install SQLite Datasource Plugin:**
   ```bash
   grafana-cli plugins install frser-sqlite-datasource
   sudo systemctl restart grafana-server
   ```

2. **Add Data Source:**
   - URL: `http://localhost:8080`
   - Or use SQLite: Path to `/var/lib/electric-grid-monitor/data/grid_monitor.db`

3. **Import Dashboard:**
   - Copy JSON from `grafana/dashboards/grid-monitor.json`
   - Import in Grafana UI

## Troubleshooting

### chip-tool not found
```bash
which chip-tool
# If not found, add to PATH or set CHIP_TOOL_PATH in .env
```

### Permission denied
```bash
# Make sure chip-tool is executable
sudo chmod +x /usr/local/bin/chip-tool

# For systemd service, ensure correct permissions
sudo chown -R root:root /var/lib/electric-grid-monitor
```

### Device not responding
```bash
# Test chip-tool manually
chip-tool electricalmeasurement read active-power 0x1234 1 --ip YOUR_DEVICE_IP

# Check device is online
ping YOUR_DEVICE_IP
```

### Database locked
```bash
# Stop service
sudo systemctl stop electric-grid-monitor

# Check database integrity
sqlite3 data/grid_monitor.db "PRAGMA integrity_check;"

# Restart service
sudo systemctl start electric-grid-monitor
```

## Development

```bash
# Run tests
make test

# Format code
make fmt

# Clean build
make clean

# Build for Raspberry Pi (ARM64)
GOOS=linux GOARCH=arm64 make build
```

## Project Structure

```
.
├── cmd/monitor/main.go           # Application entry point
├── internal/
│   ├── api/server.go             # HTTP API server
│   ├── collector/collector.go    # chip-tool wrapper
│   ├── config/config.go          # Configuration management
│   └── storage/storage.go        # SQLite database operations
├── grafana/
│   ├── dashboards/               # Grafana dashboard JSON
│   └── provisioning/             # Grafana datasource config
├── systemd/                      # Systemd service files
├── .env.example                  # Example configuration
├── docker-compose.yml            # Docker deployment
├── Dockerfile                    # Container build
└── Makefile                      # Build automation
```

## License

MIT License
