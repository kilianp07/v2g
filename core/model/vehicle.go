package model

import (
	"fmt"
	"math"
	"time"
)

// Vehicle represents an electric vehicle participating in V2X operations.
type Vehicle struct {
	ID         string
	SoC        float64   // State of charge between 0 and 1
	IsV2G      bool      // true if vehicle supports V2G
	MaxPower   float64   // max power in kW the vehicle can provide or consume
	BatteryKWh float64   // total battery capacity in kWh
	Available  bool      // whether the vehicle is currently connected and ready
	Charging   bool      // whether the vehicle is currently charging
	Priority   bool      // whether the charging session is marked as priority
	Departure  time.Time // planned departure time
	MinSoC     float64   // minimum required SoC at departure

	// Optional profile and metadata information used by advanced algorithms.
	Profile  UserProfile
	Metadata map[string]string

	// AvailabilityProb represents the probability the vehicle will remain
	// connected for the duration of the signal. It should be in the range
	// [0,1].
	AvailabilityProb float64

	// DegradationFactor estimates the fraction of capacity lost due to
	// battery degradation or temperature effects. 0 means no degradation,
	// 1 means completely unusable.
	DegradationFactor float64
}

// UserProfile contains user-specific data that can be leveraged
// to estimate availability or behaviour.
type UserProfile struct {
	ExpectedDeparture time.Time
	HistoricalUsage   []float64
}

// Validate checks that the vehicle configuration is sound.
// In particular BatteryKWh must be positive.
func (v Vehicle) Validate() error {
	if v.BatteryKWh <= 0 {
		return fmt.Errorf("battery capacity must be positive")
	}
	return nil
}

// CanProvidePower returns true if the vehicle can provide the requested power in kW.
func (v Vehicle) CanProvidePower(power float64) bool {
	return v.IsV2G && v.Available && v.SoC >= v.MinSoC && v.MaxPower >= power
}

// CanReduceCharge returns true if the vehicle can reduce its charging power.
func (v Vehicle) CanReduceCharge() bool {
	return v.Charging && !v.Priority
}

// EffectiveCapacity returns the estimated remaining power capacity for the
// vehicle considering SoC, availability probability and degradation factor.
// Fields left to their zero value are treated as neutral (e.g. probability 1).
func (v Vehicle) EffectiveCapacity(current float64) float64 {
	avail := v.AvailabilityProb
	if avail == 0 {
		avail = 1
	}
	degr := v.DegradationFactor
	if degr < 0 {
		degr = 0
	}
	if degr > 1 {
		degr = 1
	}

	cap := v.MaxPower*(1-degr) - math.Abs(current)
	if cap < 0 {
		cap = 0
	}
	if v.SoC < 0.3 {
		return 0
	}
	cap *= v.SoC * avail
	if cap < 0 {
		cap = 0
	}
	return cap
}
