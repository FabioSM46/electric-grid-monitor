package collector

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"electric-grid-monitor/internal/config"
)

// PowerReading represents a single power measurement
type PowerReading struct {
	PowerW   float64 `json:"power_w"`
	VoltageV float64 `json:"voltage_v"`
	CurrentA float64 `json:"current_a"`
	EnergyWh float64 `json:"energy_wh,omitempty"`
}

// Collector wraps chip-tool to collect power data from Matter device
type Collector struct {
	config *config.Config
}

// New creates a new collector instance
func New(cfg *config.Config) *Collector {
	return &Collector{
		config: cfg,
	}
}

// Collect reads power data from the device
func (c *Collector) Collect() (*PowerReading, error) {
	reading := &PowerReading{}

	// Read active power (in milliwatts, divide by 1000)
	power, err := c.readAttribute("electricalpowermeasurement", "active-power")
	if err != nil {
		return nil, fmt.Errorf("failed to read power: %w", err)
	}
	reading.PowerW = power / 1000.0

	// Read voltage (in millivolts, divide by 1000)
	voltage, err := c.readAttribute("electricalpowermeasurement", "voltage")
	if err != nil {
		return nil, fmt.Errorf("failed to read voltage: %w", err)
	}
	reading.VoltageV = voltage / 1000.0

	// Read current (in milliamps, divide by 1000)
	current, err := c.readAttribute("electricalpowermeasurement", "active-current")
	if err != nil {
		return nil, fmt.Errorf("failed to read current: %w", err)
	}
	reading.CurrentA = current / 1000.0

	return reading, nil
}

// readAttribute executes chip-tool to read a specific attribute
func (c *Collector) readAttribute(cluster, attribute string) (float64, error) {
	args := []string{
		cluster,
		"read",
		attribute,
		c.config.MatterNodeID,
		strconv.Itoa(c.config.MatterEndpoint),
	}

	cmd := exec.Command(c.config.ChipToolPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("chip-tool failed: %w, output: %s", err, string(output))
	}

	return c.parseOutput(string(output), attribute)
}

// parseOutput extracts the numeric value from chip-tool output
func (c *Collector) parseOutput(output string, attributeName string) (float64, error) {
	lines := strings.Split(output, "\n")

	// Look for specific patterns like "ActivePower: 123" or "Voltage: 230"
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Pattern 1: Look for "AttributeName: value" format
		// Example: "ActivePower: 123" or "Voltage: 230000"
		re := regexp.MustCompile(`(?i)` + attributeName + `:\s*(\d+)`)
		matches := re.FindStringSubmatch(line)
		if len(matches) > 1 {
			if val, err := strconv.ParseFloat(matches[1], 64); err == nil {
				return val, nil
			}
		}

		// Pattern 2: Look for "Data = X (signed)" which is the raw attribute value
		// This appears in the DMG output
		re = regexp.MustCompile(`Data = (\d+) \(signed\)`)
		matches = re.FindStringSubmatch(line)
		if len(matches) > 1 {
			if val, err := strconv.ParseFloat(matches[1], 64); err == nil {
				return val, nil
			}
		}
	}

	return 0, fmt.Errorf("could not parse value for %s from output", attributeName)
}

// GetDeviceInfo returns basic device information
func (c *Collector) GetDeviceInfo() (map[string]string, error) {
	args := []string{
		"basicinformation",
		"read",
		"product-name",
		c.config.MatterNodeID,
		strconv.Itoa(c.config.MatterEndpoint),
	}

	cmd := exec.Command(c.config.ChipToolPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to get device info: %w", err)
	}

	info := map[string]string{
		"raw_output": string(output),
	}

	return info, nil
}
