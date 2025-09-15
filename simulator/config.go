package main

import (
	"fmt"
	"time"
)

// Config holds parameters for the simulator.
type Config struct {
	Broker          string
	Count           int
	FleetSize       int
	AckLatency      time.Duration
	DropRate        float64
	DisconnectRate  float64
	CapacityKWh     float64
	ChargeRateKW    float64
	DischargeRateKW float64
	MaxPower        float64
	Interval        time.Duration

	TelemetryPush           bool
	TelemetryInterval       time.Duration
	TelemetryRequestTopic   string
	TelemetryResponsePrefix string
	StateTopicPrefix        string

	CommuterPct      float64
	AvailabilityFile string
	ScheduleFile     string
	TemplateFile     string

	BatteryProfile string
	Verbose        bool

	TopicPrefix string

	InfluxURL    string
	InfluxToken  string
	InfluxOrg    string
	InfluxBucket string
}

//gocyclo:ignore
func (c *Config) Validate() error {
	if c.Broker == "" {
		return fmt.Errorf("broker is required")
	}
	if c.FleetSize == 0 {
		c.FleetSize = c.Count
	}
	if c.FleetSize <= 0 {
		return fmt.Errorf("fleet-size must be positive")
	}
	if c.DropRate < 0 || c.DropRate > 1 {
		return fmt.Errorf("drop-rate must be between 0 and 1")
	}
	if c.DisconnectRate < 0 || c.DisconnectRate > 1 {
		return fmt.Errorf("disconnect-rate must be between 0 and 1")
	}
	if c.CapacityKWh <= 0 {
		return fmt.Errorf("capacity must be positive")
	}
	if c.ChargeRateKW <= 0 || c.DischargeRateKW <= 0 {
		return fmt.Errorf("charge and discharge rates must be positive")
	}
	if c.MaxPower <= 0 {
		return fmt.Errorf("max-power must be positive")
	}
	if c.Interval <= 0 {
		c.Interval = time.Second
	}
	if c.TelemetryInterval <= 0 {
		c.TelemetryInterval = c.Interval
	}
	if c.CommuterPct < 0 || c.CommuterPct > 1 {
		return fmt.Errorf("commuter percentage must be between 0 and 1")
	}
	return nil
}
