package main

import "time"

// Config holds parameters for the simulator.
type Config struct {
	Broker     string
	Count      int
	AckLatency time.Duration
	DropRate   float64
}
