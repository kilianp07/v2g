package dispatch

import "github.com/kilianp07/v2g/core/model"

// NoopFallback does not reallocate power.
type NoopFallback struct{}

func (NoopFallback) Reallocate(failed []model.Vehicle, current map[string]float64, signal model.FlexibilitySignal) map[string]float64 {
	return current
}

// VehicleAwareFallback can receive the list of vehicles used in a dispatch.
// Implementations can use this information to compute remaining capacity.
type VehicleAwareFallback interface {
	SetVehicles([]model.Vehicle)
}
