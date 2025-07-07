package eco

import "time"

// Record aggregates ecological metrics for a vehicle and day.
type Record struct {
	VehicleID   string
	Date        time.Time
	InjectedKWh float64
	ConsumedKWh float64
}

// CO2Avoided returns the grams of CO2 avoided using the emission factor.
func (r Record) CO2Avoided(factor float64) float64 {
	return r.InjectedKWh * factor
}

// EnergyRatio returns the ratio of injected to consumed energy.
func (r Record) EnergyRatio() float64 {
	if r.ConsumedKWh == 0 {
		if r.InjectedKWh == 0 {
			return 0
		}
		return r.InjectedKWh
	}
	return r.InjectedKWh / r.ConsumedKWh
}
