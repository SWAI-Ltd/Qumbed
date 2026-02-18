package proto

import (
	"encoding/json"
	"fmt"
)

// Schema IDs for typed topics
const (
	SchemaTemperature = "sensor.Temperature"
	SchemaHumidity    = "sensor.Humidity"
	SchemaCommand     = "control.Command"
)

// Temperature sensor reading
type Temperature struct {
	Celsius    float64 `json:"celsius"`
	TimestampMs int64  `json:"timestamp_ms"`
	SensorID   string `json:"sensor_id"`
}

// Humidity sensor reading
type Humidity struct {
	Percent    float64 `json:"percent"`
	TimestampMs int64  `json:"timestamp_ms"`
	SensorID   string `json:"sensor_id"`
}

// Command for actuators
type Command struct {
	Action string            `json:"action"`
	Params map[string]string `json:"params"`
}

// ValidatePayload checks that payload matches the expected schema
func ValidatePayload(schemaID string, payload []byte) error {
	switch schemaID {
	case SchemaTemperature:
		var t Temperature
		if err := json.Unmarshal(payload, &t); err != nil {
			return fmt.Errorf("invalid Temperature: %w", err)
		}
		return nil
	case SchemaHumidity:
		var h Humidity
		if err := json.Unmarshal(payload, &h); err != nil {
			return fmt.Errorf("invalid Humidity: %w", err)
		}
		return nil
	case SchemaCommand:
		var c Command
		if err := json.Unmarshal(payload, &c); err != nil {
			return fmt.Errorf("invalid Command: %w", err)
		}
		if c.Action == "" {
			return fmt.Errorf("Command.action required")
		}
		return nil
	default:
		return fmt.Errorf("unknown schema: %s", schemaID)
	}
}

// KnownSchemas returns all registered schema IDs
func KnownSchemas() []string {
	return []string{SchemaTemperature, SchemaHumidity, SchemaCommand}
}
