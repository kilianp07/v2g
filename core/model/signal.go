package model

import "time"

// SignalType defines the type of flexibility signal received.
type SignalType int

const (
	SignalFCR SignalType = iota
	SignalAFRR
	SignalMA
	SignalNEBEF
	SignalEcoWatt
)

// FlexibilitySignal represents a dispatch request from the grid.
type FlexibilitySignal struct {
	Type      SignalType
	PowerKW   float64       // requested power in kW (positive for injection, negative for consumption reduction)
	Duration  time.Duration // duration of the signal
	Timestamp time.Time     // time at which signal was received
}

// String returns a human-readable representation of the signal type.
func (t SignalType) String() string {
	switch t {
	case SignalFCR:
		return "FCR"
	case SignalAFRR:
		return "aFRR"
	case SignalMA:
		return "MA"
	case SignalNEBEF:
		return "NEBEF"
	case SignalEcoWatt:
		return "EcoWatt"
	default:
		return "unknown"
	}
}

// IsStrict returns true for signals that require exact power delivery.
func (s FlexibilitySignal) IsStrict() bool {
	return s.Type == SignalFCR
}
