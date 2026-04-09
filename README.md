# Electric Grid Monitor

Go application to monitor Sonoff S61s power consumption using Matter protocol (chip-tool).

## 🆕 Quick Start - Direct Commissioning from Raspberry Pi

**New:** You can now commission the Sonoff S61s **directly from your Raspberry Pi** without needing Google Home!

See [COMMISSIONING.md](COMMISSIONING.md) for detailed commissioning instructions.

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

1. **Raspberry Pi** (3B+ or 4 recommended)
2. **Sonoff S61s** smart plug
3. **chip-tool** installed on Raspberry Pi
4. **Go 1.21+** (for building from source)

## Installation Options

### Option 1: Automated Setup (Recommended)

```bash
# On Raspberry Pi:
git clone <repository-url>
cd electric-grid-monitor
./scripts/install-raspberry-pi.sh

# Follow the prompts to configure
```

### Option 2: Docker Compose

```bash
# Copy and edit configuration
cp .env.example .env
nano .env

# Run everything
docker-compose up -d

# Access Grafana at http://raspberry-pi-ip:3000
```

### Option 3: Manual Build

```bash
# Install dependencies
go mod tidy

# Build
make build

# Create directories
mkdir -p data logs

# Configure
cp .env.example .env
nano .env

# Run
./build/electric-grid-monitor
```

## Configuration

Copy `.env.example` to `.env` and configure:

```bash
# Matter Device Configuration (see COMMISSIONING.md)
MATTER_NODE_ID=0x1234          # Node ID from commissioning
MATTER_ENDPOINT=1              # Usually 1 for Sonoff S61s
MATTER_IP=192.168.1.100        # Sonoff S61s IP address

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
```

## Commissioning Your Device

**New!** Commission directly from Raspberry Pi:

1. **Install chip-tool** (see COMMISSIONING.md)
2. **Reset Sonoff S61s** (hold button 10-15 seconds)
3. **Commission:**
   ```bash
   # Using QR code from device sticker
   chip-tool pairing qrcode 0x1234 MT:XXXXXXXXXXX
   
   # OR using manual pairing code
   chip-tool pairing code 0x1234 35792000079
   ```
4. **Test:**
   ```bash
   chip-tool electricalmeasurement read active-power 0x1234 1
   ```

See [COMMISSIONING.md](COMMISSIONING.md) for complete guide.

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

Access at: `http://raspberry-pi-ip:3000` (admin/admin)

### Option 2: Manual Setup

1. **Install SQLite Datasource Plugin:**
   ```bash
   grafana-cli plugins install frser-sqlite-datasource
   sudo systemctl restart grafana-server
   ```

2. **Add Data Source:**
   - URL: `http://localhost:8080`
   - Or use SQLite: Path to database file

3. **Import Dashboard:**
   - Copy JSON from `grafana/dashboards/grid-monitor.json`
   - Import in Grafana UI

## Systemd Service Management

```bash
# Enable service to start on boot
sudo systemctl enable electric-grid-monitor

# Start service
sudo systemctl start electric-grid-monitor

# Stop service
sudo systemctl stop electric-grid-monitor

# Check status
sudo systemctl status electric-grid-monitor

# View logs
sudo journalctl -u electric-grid-monitor -f

# Restart after config changes
sudo systemctl restart electric-grid-monitor
```

## Troubleshooting

### chip-tool not found
```bash
which chip-tool
# If not found, install it first (see COMMISSIONING.md)
```

### Device not responding
```bash
# Test chip-tool manually
chip-tool electricalmeasurement read active-power 0x1234 1 --ip YOUR_DEVICE_IP

# Check device is online
ping YOUR_DEVICE_IP

# Verify commissioning worked
chip-tool discover commissionables
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

### Permission denied
```bash
# Fix permissions
sudo chown -R $USER:$USER /var/lib/electric-grid-monitor
sudo chown -R $USER:$USER /var/log/electric-grid-monitor
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

# Run in development mode
make dev
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
├── scripts/
│   └── install-raspberry-pi.sh   # Automated setup script
├── grafana/
│   ├── dashboards/               # Grafana dashboard JSON
│   └── provisioning/             # Grafana datasource config
├── systemd/                      # Systemd service files
├── COMMISSIONING.md              # Detailed commissioning guide
├── .env.example                  # Example configuration
├── docker-compose.yml            # Docker deployment
├── Dockerfile                    # Container build
└── Makefile                      # Build automation
```

## Documentation

- [COMMISSIONING.md](COMMISSIONING.md) - Step-by-step device commissioning guide
- This README - General setup and usage

## Features

- ✅ **Direct Matter protocol** via chip-tool CLI wrapper
- ✅ **No cloud dependency** - fully local
- ✅ **SQLite storage** with automatic cleanup
- ✅ **HTTP API** for Grafana integration
- ✅ **Configurable via .env file**
- ✅ **Systemd service** for production deployment
- ✅ **Docker support** with docker-compose
- ✅ **Pre-configured Grafana dashboards**
- ✅ **Commission directly from Raspberry Pi**

## License

MIT License
