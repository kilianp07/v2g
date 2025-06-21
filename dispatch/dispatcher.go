package dispatch

import "github.com/kilianp07/v2g/model"

// EqualDispatcher distributes power equally between all vehicles.
type EqualDispatcher struct{}

func (d EqualDispatcher) Dispatch(vehicles []model.Vehicle, signal model.FlexibilitySignal) map[string]float64 {
	assignments := make(map[string]float64)
	if len(vehicles) == 0 {
		return assignments
	}
	powerPerVehicle := signal.PowerKW / float64(len(vehicles))
	for _, v := range vehicles {
		if v.MaxPower < powerPerVehicle {
			assignments[v.ID] = v.MaxPower
		} else {
			assignments[v.ID] = powerPerVehicle
		}
	}
	return assignments
}
