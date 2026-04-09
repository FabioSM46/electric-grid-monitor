package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	// Matter device settings
	MatterNodeID   string
	MatterEndpoint int
	MatterIP       string

	// Collection settings
	CollectionInterval time.Duration
	DataRetentionDays  int

	// Database
	DBPath string

	// API Server
	APIPort string
	APIHost string

	// chip-tool
	ChipToolPath string

	// Logging
	LogLevel string
	LogFile  string
}

func Load() (*Config, error) {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		// It's okay if .env doesn't exist, we'll use defaults or env vars
		fmt.Println("No .env file found, using environment variables or defaults")
	}

	cfg := &Config{
		// Matter defaults
		MatterNodeID:   getEnv("MATTER_NODE_ID", "0x1234"),
		MatterEndpoint: getEnvInt("MATTER_ENDPOINT", 1),
		MatterIP:       getEnv("MATTER_IP", ""),

		// Collection defaults
		CollectionInterval: getEnvDuration("COLLECTION_INTERVAL", 30*time.Second),
		DataRetentionDays:  getEnvInt("DATA_RETENTION_DAYS", 365),

		// Database defaults
		DBPath: getEnv("DB_PATH", "./data/grid_monitor.db"),

		// API defaults
		APIPort: getEnv("API_PORT", "8080"),
		APIHost: getEnv("API_HOST", "0.0.0.0"),

		// chip-tool defaults
		ChipToolPath: getEnv("CHIP_TOOL_PATH", "chip-tool"),

		// Logging defaults
		LogLevel: getEnv("LOG_LEVEL", "info"),
		LogFile:  getEnv("LOG_FILE", "./logs/monitor.log"),
	}

	// Validate required fields
	if cfg.MatterIP == "" {
		return nil, fmt.Errorf("MATTER_IP is required")
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	return defaultValue
}
