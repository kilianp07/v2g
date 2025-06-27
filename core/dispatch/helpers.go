package dispatch

import "github.com/kilianp07/v2g/core/model"

// prepareVehicles filters vehicles with positive energy capacity and power
// capability and returns candidate structs with their score and capacity.
func prepareVehicles(vehicles []model.Vehicle, signal model.FlexibilitySignal, ctx *DispatchContext, scorer func(model.Vehicle, *DispatchContext) float64) []candidate {
	var list []candidate
	for _, v := range vehicles {
		energy := (v.SoC - v.MinSoC) * v.BatteryKWh
		if energy <= 0 {
			continue
		}
		cap := v.MaxPower
		if signal.Duration > 0 {
			maxFromEnergy := energy / signal.Duration.Hours()
			if maxFromEnergy < cap {
				cap = maxFromEnergy
			}
		}
		if cap <= 0 {
			continue
		}
		list = append(list, candidate{v: v, score: scorer(v, ctx), capacity: cap})
	}
	return list
}
