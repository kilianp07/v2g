package model

import "time"

// Vehicle represents an electric vehicle participating in V2X operations.
type Vehicle struct {
	ID        string
	SoC       float64   // State of charge between 0 and 1
	IsV2G     bool      // true if vehicle supports V2G
	MaxPower  float64   // max power in kW the vehicle can provide or consume
	Available bool      // whether the vehicle is currently connected and ready
	Charging  bool      // whether the vehicle is currently charging
	Priority  bool      // whether the charging session is marked as priority
	Departure time.Time // planned departure time
	MinSoC    float64   // minimum required SoC at departure
}

// CanProvidePower returns true if the vehicle can provide the requested power in kW.
func (v Vehicle) CanProvidePower(power float64) bool {
	return v.IsV2G && v.Available && v.SoC >= v.MinSoC && v.MaxPower >= power
}

// CanReduceCharge returns true if the vehicle can reduce its charging power.
func (v Vehicle) CanReduceCharge() bool {
	return v.Charging && !v.Priority
}
