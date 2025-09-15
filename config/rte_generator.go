package config

import (
	"fmt"
	"time"
)

// RTEGeneratorConfig configures the internal RTE signal generator.
type RTEGeneratorConfig struct {
	Enabled            bool     `json:"enabled"`
	Mode               string   `json:"mode"`
	HTTPEndpoint       string   `json:"httpEndpoint"`
	Scenario           string   `json:"scenario"`
	MinIntervalSeconds int      `json:"min_interval_seconds"`
	MaxIntervalSeconds int      `json:"max_interval_seconds"`
	MinDurationSeconds int      `json:"min_duration_seconds"`
	MaxDurationSeconds int      `json:"max_duration_seconds"`
	MinPowerKW         float64  `json:"min_power_kw"`
	MaxPowerKW         float64  `json:"max_power_kw"`
	SignalTypes        []string `json:"signal_types"`
	JitterPct          float64  `json:"jitter_pct"`
	Seed               int64    `json:"seed"`
	TimeZone           string   `json:"time_zone"`
}

// SetDefaults applies fallback values for optional fields.
func (c *RTEGeneratorConfig) SetDefaults() {
	if c.MinIntervalSeconds <= 0 {
		c.MinIntervalSeconds = 120
	}
	if c.MaxIntervalSeconds <= 0 {
		c.MaxIntervalSeconds = 300
	}
	if c.MinDurationSeconds <= 0 {
		c.MinDurationSeconds = 120
	}
	if c.MaxDurationSeconds <= 0 {
		c.MaxDurationSeconds = 600
	}
	if c.MinPowerKW == 0 {
		c.MinPowerKW = 5
	}
	if c.MaxPowerKW == 0 {
		c.MaxPowerKW = 25
	}
	if c.JitterPct == 0 {
		c.JitterPct = 0.15
	}
	if c.Scenario == "" {
		c.Scenario = "steady"
	}
	if c.Mode == "" {
		c.Mode = "internal"
	}
	if len(c.SignalTypes) == 0 {
		c.SignalTypes = []string{"FCR"}
	}
	if c.TimeZone == "" {
		c.TimeZone = time.Local.String()
	}
}

// Validate checks the configuration ranges.
func (c RTEGeneratorConfig) Validate() error {
	if c.MinIntervalSeconds < 0 || c.MaxIntervalSeconds < 0 {
		return fmt.Errorf("interval seconds must be positive")
	}
	if c.MinIntervalSeconds > c.MaxIntervalSeconds {
		return fmt.Errorf("min_interval_seconds > max_interval_seconds")
	}
	if c.MinDurationSeconds <= 0 || c.MaxDurationSeconds <= 0 {
		return fmt.Errorf("duration seconds must be >0")
	}
	if c.MinDurationSeconds > c.MaxDurationSeconds {
		return fmt.Errorf("min_duration_seconds > max_duration_seconds")
	}
	if c.MinPowerKW > c.MaxPowerKW {
		return fmt.Errorf("min_power_kw > max_power_kw")
	}
	if c.Mode != "internal" && c.Mode != "http" && c.Mode != "" {
		return fmt.Errorf("unknown mode %s", c.Mode)
	}
	return nil
}

func (c RTEGeneratorConfig) minInterval() time.Duration {
	return time.Duration(c.MinIntervalSeconds) * time.Second
}

func (c RTEGeneratorConfig) maxInterval() time.Duration {
	return time.Duration(c.MaxIntervalSeconds) * time.Second
}

func (c RTEGeneratorConfig) minDuration() time.Duration {
	return time.Duration(c.MinDurationSeconds) * time.Second
}

func (c RTEGeneratorConfig) maxDuration() time.Duration {
	return time.Duration(c.MaxDurationSeconds) * time.Second
}
