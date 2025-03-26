package dispatch

import "time"

// FlexibilitySignal represents a request from the grid operator for power adjustment.
type FlexibilitySignal struct {
	Type        string  // "PRIMARY_RESERVE", "SECONDARY_RESERVE", "LOAD_SHEDDING"
	Power       float64 // Requested power adjustment (kW)
	Duration    int     // Duration in minutes
	Timestamp   time.Time
	MarketPrice float64 // Market price of energy in €/kWh
}
