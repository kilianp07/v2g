package main

import (
	"fmt"
	"time"
)

// Config holds parameters for the simulator.
type Config struct {
	Broker     string
	Count      int
	AckLatency time.Duration
	DropRate   float64
}

func (c Config) Validate() error {
	if c.Broker == "" {
		return fmt.Errorf("broker is required")
	}
	if c.Count <= 0 {
		return fmt.Errorf("count must be positive")
	}
	if c.DropRate < 0 || c.DropRate > 1 {
		return fmt.Errorf("drop-rate must be between 0 and 1")
	}
	return nil
}
