package dispatch

import "time"

// Vehicle represents an electric vehicle with V2G capabilities.
type Vehicle struct {
	ID            string
	StateOfCharge float64   // Battery charge level in percentage
	MaxPower      float64   // Maximum power available for charging/discharging (kW)
	Available     bool      // Whether the vehicle is available for dispatch
	Priority      int       // Priority level for dispatch selection
	Tariff        float64   // Electricity tariff in €/kWh
	LastUpdate    time.Time // Last state update timestamp
}
