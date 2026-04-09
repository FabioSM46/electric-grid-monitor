package collector

import (
	"encoding/json"
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

	// Read active power (in deci-watts, divide by 10)
	power, err := c.readAttribute("electricalmeasurement", "active-power")
	if err != nil {
		return nil, fmt.Errorf("failed to read power: %w", err)
	}
	reading.PowerW = power / 10.0

	// Read RMS voltage (in millivolts, divide by 1000)
	voltage, err := c.readAttribute("electricalmeasurement", "rms-voltage")
	if err != nil {
		return nil, fmt.Errorf("failed to read voltage: %w", err)
	}
	reading.VoltageV = voltage / 1000.0

	// Read RMS current (in milliamps, divide by 1000)
	current, err := c.readAttribute("electricalmeasurement", "rms-current")
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
		"--ip",
		c.config.MatterIP,
	}

	cmd := exec.Command(c.config.ChipToolPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("chip-tool failed: %w, output: %s", err, string(output))
	}

	return c.parseOutput(string(output))
}

// parseOutput extracts the numeric value from chip-tool output
func (c *Collector) parseOutput(output string) (float64, error) {
	// Look for pattern like: "ActivePower: 1234" or just the number
	// chip-tool output varies, but usually contains the value in the response

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Try to find JSON-like output
		if strings.Contains(line, "{") && strings.Contains(line, "}") {
			// Try to parse as JSON
			var result map[string]interface{}
			if err := json.Unmarshal([]byte(line), &result); err == nil {
				// Look for value field
				if val, ok := result["value"]; ok {
					switch v := val.(type) {
					case float64:
						return v, nil
					case int:
						return float64(v), nil
					}
				}
			}
		}

		// Try regex pattern for "AttributeName: value"
		re := regexp.MustCompile(`:\s*(\d+)`)
		matches := re.FindStringSubmatch(line)
		if len(matches) > 1 {
			if val, err := strconv.ParseFloat(matches[1], 64); err == nil {
				return val, nil
			}
		}

		// Try to extract just a number
		re = regexp.MustCompile(`\b(\d+)\b`)
		matches = re.FindStringSubmatch(line)
		if len(matches) > 1 {
			if val, err := strconv.ParseFloat(matches[1], 64); err == nil {
				return val, nil
			}
		}
	}

	return 0, fmt.Errorf("could not parse value from output: %s", output)
}

// GetDeviceInfo returns basic device information
func (c *Collector) GetDeviceInfo() (map[string]string, error) {
	args := []string{
		"basicinformation",
		"read",
		"product-name",
		c.config.MatterNodeID,
		strconv.Itoa(c.config.MatterEndpoint),
		"--ip",
		c.config.MatterIP,
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
