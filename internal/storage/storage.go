package storage

import (
	"database/sql"
	"fmt"
	"time"

	"electric-grid-monitor/internal/collector"
	_ "github.com/mattn/go-sqlite3"
)

// Store handles database operations
type Store struct {
	db *sql.DB
}

// New creates a new storage instance
func New(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite3", dbPath)
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
	CREATE INDEX IF NOT EXISTS idx_date ON power_readings(date(timestamp));
	`

	_, err := s.db.Exec(query)
	return err
}

// SaveReading stores a power reading in the database
func (s *Store) SaveReading(reading *collector.PowerReading) error {
	query := `
	INSERT INTO power_readings (timestamp, power_w, voltage_v, current_a, energy_wh)
	VALUES (?, ?, ?, ?, ?)
	`

	_, err := s.db.Exec(query,
		time.Now(),
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

	rows, err := s.db.Query(query, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var readings []PowerReadingRecord
	for rows.Next() {
		var r PowerReadingRecord
		if err := rows.Scan(&r.ID, &r.Timestamp, &r.PowerW, &r.VoltageV, &r.CurrentA, &r.EnergyWh); err != nil {
			return nil, err
		}
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
	err := s.db.QueryRow(query).Scan(&r.ID, &r.Timestamp, &r.PowerW, &r.VoltageV, &r.CurrentA, &r.EnergyWh)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &r, nil
}

// GetDailyStats returns aggregated statistics for a specific date
func (s *Store) GetDailyStats(date time.Time) (*DailyStats, error) {
	query := `
	SELECT 
		AVG(power_w) as avg_power,
		MAX(power_w) as max_power,
		MIN(power_w) as min_power,
		AVG(voltage_v) as avg_voltage,
		AVG(current_a) as avg_current,
		COUNT(*) as reading_count
	FROM power_readings
	WHERE date(timestamp) = date(?)
	`

	var stats DailyStats
	stats.Date = date.Format("2006-01-02")

	err := s.db.QueryRow(query, date).Scan(
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
	query := `
	DELETE FROM power_readings
	WHERE timestamp < datetime('now', ?)
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
