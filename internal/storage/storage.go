package storage

import (
	"database/sql"
	"fmt"
	"time"

	"electric-grid-monitor/internal/collector"
	_ "modernc.org/sqlite"
)

// Store handles database operations
type Store struct {
	db *sql.DB
}

// New creates a new storage instance
func New(dbPath string) (*Store, error) {
	// Enable WAL mode and set a busy timeout so concurrent readers (Grafana)
	// don't block the writer and vice-versa.
	dsn := fmt.Sprintf("%s?_pragma=journal_mode%%3Dwal&_pragma=busy_timeout%%3D5000", dbPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	store := &Store{db: db}
	if err := store.createTables(); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	return store, nil
}

// createTables initializes the database schema
func (s *Store) createTables() error {
	query := `
	CREATE TABLE IF NOT EXISTS power_readings (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		power_w REAL NOT NULL,
		voltage_v REAL NOT NULL,
		current_a REAL NOT NULL,
		energy_wh REAL DEFAULT 0
	);

	CREATE INDEX IF NOT EXISTS idx_timestamp ON power_readings(timestamp);
	CREATE INDEX IF NOT EXISTS idx_date ON power_readings(date(timestamp, 'unixepoch'));

	CREATE TABLE IF NOT EXISTS settings (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);
	`

	_, err := s.db.Exec(query)
	return err
}

// SaveSetting upserts a key/value pair in the settings table.
func (s *Store) SaveSetting(key, value string) error {
	query := `INSERT INTO settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value`
	_, err := s.db.Exec(query, key, value)
	return err
}

// SaveReading stores a power reading in the database
func (s *Store) SaveReading(reading *collector.PowerReading) error {
	query := `
	INSERT INTO power_readings (timestamp, power_w, voltage_v, current_a, energy_wh)
	VALUES (?, ?, ?, ?, ?)
	`

	_, err := s.db.Exec(query,
		time.Now().Unix(),
		reading.PowerW,
		reading.VoltageV,
		reading.CurrentA,
		reading.EnergyWh,
	)

	return err
}

// GetHistory returns power readings between two timestamps
func (s *Store) GetHistory(from, to time.Time) ([]PowerReadingRecord, error) {
	query := `
	SELECT id, timestamp, power_w, voltage_v, current_a, energy_wh
	FROM power_readings
	WHERE timestamp BETWEEN ? AND ?
	ORDER BY timestamp DESC
	`

	rows, err := s.db.Query(query, from.Unix(), to.Unix())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var readings []PowerReadingRecord
	for rows.Next() {
		var r PowerReadingRecord
		var ts int64
		if err := rows.Scan(&r.ID, &ts, &r.PowerW, &r.VoltageV, &r.CurrentA, &r.EnergyWh); err != nil {
			return nil, err
		}
		r.Timestamp = time.Unix(ts, 0)
		readings = append(readings, r)
	}

	return readings, rows.Err()
}

// GetLatest returns the most recent power reading
func (s *Store) GetLatest() (*PowerReadingRecord, error) {
	query := `
	SELECT id, timestamp, power_w, voltage_v, current_a, energy_wh
	FROM power_readings
	ORDER BY timestamp DESC
	LIMIT 1
	`

	var r PowerReadingRecord
	var ts int64
	err := s.db.QueryRow(query).Scan(&r.ID, &ts, &r.PowerW, &r.VoltageV, &r.CurrentA, &r.EnergyWh)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	r.Timestamp = time.Unix(ts, 0)

	return &r, nil
}

// GetDailyStats returns aggregated statistics for a specific date
func (s *Store) GetDailyStats(date time.Time) (*DailyStats, error) {
	// Timestamps are stored as Unix epoch integers, so convert them via
	// datetime(timestamp, 'unixepoch') before applying date() for grouping.
	query := `
	SELECT 
		AVG(power_w) as avg_power,
		MAX(power_w) as max_power,
		MIN(power_w) as min_power,
		AVG(voltage_v) as avg_voltage,
		AVG(current_a) as avg_current,
		COUNT(*) as reading_count
	FROM power_readings
	WHERE date(timestamp, 'unixepoch') = date(?)
	`

	var stats DailyStats
	stats.Date = date.Format("2006-01-02")

	err := s.db.QueryRow(query, date.Format("2006-01-02")).Scan(
		&stats.AvgPower,
		&stats.MaxPower,
		&stats.MinPower,
		&stats.AvgVoltage,
		&stats.AvgCurrent,
		&stats.ReadingCount,
	)
	if err != nil {
		return nil, err
	}

	return &stats, nil
}

// CleanupOldData removes data older than retention days
func (s *Store) CleanupOldData(retentionDays int) error {
	// Timestamps are stored as Unix epoch integers, so use strftime('%s', ...)
	// to produce a comparable integer instead of datetime() which returns an ISO string.
	query := `
	DELETE FROM power_readings
	WHERE timestamp < strftime('%s', 'now', ?)
	`

	_, err := s.db.Exec(query, fmt.Sprintf("-%d days", retentionDays))
	return err
}

// Close closes the database connection
func (s *Store) Close() error {
	return s.db.Close()
}

// PowerReadingRecord represents a database record
type PowerReadingRecord struct {
	ID        int64     `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	PowerW    float64   `json:"power_w"`
	VoltageV  float64   `json:"voltage_v"`
	CurrentA  float64   `json:"current_a"`
	EnergyWh  float64   `json:"energy_wh"`
}

// DailyStats represents aggregated statistics for a day
type DailyStats struct {
	Date         string  `json:"date"`
	AvgPower     float64 `json:"avg_power_w"`
	MaxPower     float64 `json:"max_power_w"`
	MinPower     float64 `json:"min_power_w"`
	AvgVoltage   float64 `json:"avg_voltage_v"`
	AvgCurrent   float64 `json:"avg_current_a"`
	ReadingCount int64   `json:"reading_count"`
}
