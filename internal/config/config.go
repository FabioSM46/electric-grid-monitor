package config

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Duration wraps time.Duration to support JSON string unmarshaling (e.g. "30s", "5m").
type Duration struct {
	time.Duration
}

func (d *Duration) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	dur, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", s, err)
	}
	d.Duration = dur
	return nil
}

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.Duration.String())
}

type Config struct {
	// Matter device settings
	MatterNodeID   string `json:"matter_node_id"`
	MatterEndpoint int    `json:"matter_endpoint"`
	MatterIP       string `json:"matter_ip"`

	// Collection settings
	CollectionInterval Duration `json:"collection_interval"`
	DataRetentionDays  int      `json:"data_retention_days"`

	// Database
	DBPath string `json:"db_path"`

	// API Server
	APIPort string `json:"api_port"`
	APIHost string `json:"api_host"`

	// chip-tool
	ChipToolPath string `json:"chip_tool_path"`

	// Energy cost
	KwhCost float64 `json:"kwh_cost"`

	// Logging
	LogLevel string `json:"log_level"`
}

// defaults returns a Config with sensible default values.
func defaults() *Config {
	return &Config{
		MatterNodeID:       "0x1234",
		MatterEndpoint:     1,
		CollectionInterval: Duration{30 * time.Second},
		DataRetentionDays:  365,
		DBPath:             "./data/grid_monitor.db",
		APIPort:            "8080",
		APIHost:            "0.0.0.0",
		ChipToolPath:       "chip-tool",
		LogLevel:           "info",
	}
}

// Load reads config.json from the working directory (or the path given by
// CONFIG_PATH env var) and returns a validated Config. Fields not present in
// the JSON file keep their default values.
func Load() (*Config, error) {
	cfg := defaults()

	path := "config.json"
	if p := os.Getenv("CONFIG_PATH"); p != "" {
		path = p
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config file not found: %s", path)
		}
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	// Validate required fields
	if cfg.MatterIP == "" {
		return nil, fmt.Errorf("matter_ip is required in %s", path)
	}

	return cfg, nil
}
