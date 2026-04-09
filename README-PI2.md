# Electric Grid Monitor - Raspberry Pi 2

Pre-built binary for ARMv7 (Raspberry Pi 2).

## Files

- `electric-grid-monitor` - Main application binary
- `.env` - Configuration file
- `scripts/` - Helper scripts
  - `commission-helper.sh` - Interactive device commissioning
  - `build-chip-tool-pi2.sh` - Build chip-tool from source

## Quick Start

### 1. Build chip-tool (30-60 minutes)

**⚠️ IMPORTANT:** Use `screen` or `tmux` to prevent SSH timeout!

```bash
cd ~/electric-grid-monitor-app/scripts

# Using screen (recommended):
screen -S chipbuild
./build-chip-tool-pi2.sh
# Press Ctrl+A then D to detach (build continues in background)
# Reconnect later: screen -r chipbuild

# OR using tmux:
tmux new -s chipbuild
./build-chip-tool-pi2.sh
# Press Ctrl+B then D to detach
# Reconnect later: tmux attach -t chipbuild
```

### 2. Commission Your Sonoff S61s

```bash
cd ~/electric-grid-monitor-app
./scripts/commission-helper.sh
```

Follow the interactive menu to commission your device.

### 3. Update Configuration

Edit `.env` with your device settings:

```bash
nano .env
```

Update:
- `MATTER_NODE_ID` (from commissioning, e.g., 0x1234)
- `MATTER_IP` (find with: `sudo arp-scan --localnet | grep -i sonoff`)

### 4. Create Directories

```bash
mkdir -p data logs
```

### 5. Run the Application

```bash
./electric-grid-monitor
```

The application will:
- Start collecting data every 30 seconds
- Store in SQLite database (data/grid_monitor.db)
- Expose API on port 8080

### 6. Test

```bash
# Health check
curl http://localhost:8080/health

# Current power
curl http://localhost:8080/api/v1/power/current

# Historical data
curl http://localhost:8080/api/v1/history
```

## Install as Service

```bash
# Create systemd service
sudo tee /etc/systemd/system/electric-grid-monitor.service > /dev/null << 'EOF'
[Unit]
Description=Electric Grid Monitor
After=network.target

[Service]
Type=simple
User=fopea
WorkingDirectory=/home/fopea/electric-grid-monitor-app
ExecStart=/home/fopea/electric-grid-monitor-app/electric-grid-monitor
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF

# Enable and start
sudo systemctl daemon-reload
sudo systemctl enable electric-grid-monitor
sudo systemctl start electric-grid-monitor

# Check status
sudo systemctl status electric-grid-monitor
sudo journalctl -u electric-grid-monitor -f
```

## API Endpoints

- `GET /health` - Health check
- `GET /api/v1/power/current` - Current power consumption
- `GET /api/v1/history` - Historical data (last 24h default)
- `GET /api/v1/history?from=2024-01-01&to=2024-01-31` - Date range
- `GET /api/v1/stats/daily?date=2024-01-01` - Daily statistics

## Troubleshooting

### chip-tool not found
Build it first:
```bash
cd scripts
./build-chip-tool-pi2.sh
```

### Permission denied
```bash
chmod +x electric-grid-monitor
chmod +x scripts/*.sh
```

### Database locked
```bash
# Stop service if running
sudo systemctl stop electric-grid-monitor

# Check database
sqlite3 data/grid_monitor.db "PRAGMA integrity_check;"

# Restart
sudo systemctl start electric-grid-monitor
```

## Database

SQLite database location: `data/grid_monitor.db`

View data manually:
```bash
sqlite3 data/grid_monitor.db "SELECT * FROM power_readings ORDER BY timestamp DESC LIMIT 10;"
```

## Grafana Integration

See COMMISSIONING.md for Grafana setup instructions.
