package dispatch

import (
	"math"

	"github.com/kilianp07/v2g/core/logger"
	"github.com/kilianp07/v2g/core/model"
)

// BalancedFallback redistributes power from failed vehicles to the remaining
// ones while respecting their max power capacity.
type BalancedFallback struct {
	vehicles map[string]model.Vehicle
	logger   logger.Logger
}

// NewBalancedFallback returns a new BalancedFallback instance.
func NewBalancedFallback(log logger.Logger) *BalancedFallback {
	return &BalancedFallback{vehicles: make(map[string]model.Vehicle), logger: log}
}

// SetVehicles stores the vehicles used for a dispatch so that the fallback can
// compute remaining capacity.
func (b *BalancedFallback) SetVehicles(vs []model.Vehicle) {
	b.vehicles = make(map[string]model.Vehicle, len(vs))
	for _, v := range vs {
		b.vehicles[v.ID] = v
	}
}

// Reallocate implements FallbackStrategy. It uses the remaining capacity of
// successful vehicles to meet the original power target as much as possible.
func (b *BalancedFallback) Reallocate(failed []model.Vehicle, current map[string]float64, signal model.FlexibilitySignal) map[string]float64 {
	res := make(map[string]float64, len(current))
	for id, p := range current {
		res[id] = p
	}

	if len(failed) == 0 {
		return res
	}

	sign := 1.0
	if signal.PowerKW < 0 {
		sign = -1
	}

	failedIDs := make(map[string]struct{}, len(failed))
	var residual float64
	for _, v := range failed {
		failedIDs[v.ID] = struct{}{}
		residual += math.Abs(res[v.ID])
		res[v.ID] = 0
	}
	if residual == 0 {
		return res
	}

	b.logger.Infof("fallback: reallocating %.2f kW after %d failures", residual*sign, len(failed))

	avail := b.availableCapacity(current, failedIDs, res)
	remaining := allocatePower(avail, res, residual, sign)

	efficiency := 1.0
	if residual > 0 {
		efficiency = (residual - remaining) / residual
	}
	if remaining > 0 {
		b.logger.Warnf("fallback: %.2f kW could not be reallocated (%.0f%% efficiency)", remaining, efficiency*100)
	} else {
		b.logger.Infof("fallback completed with %.0f%% efficiency", efficiency*100)
	}
	return res
}

type alloc struct {
	id       string
	capacity float64
}

func (b *BalancedFallback) availableCapacity(current map[string]float64, failed map[string]struct{}, res map[string]float64) []alloc {
	var avail []alloc
	for id := range current {
		if _, ok := failed[id]; ok {
			continue
		}
		veh, ok := b.vehicles[id]
		cap := 0.0 // capacity weighted by state of charge
		if ok {
			cap = veh.MaxPower - math.Abs(res[id])
			if cap < 0 {
				cap = 0
			}
			if veh.SoC < 0.3 {
				cap = 0
			}
			cap *= veh.SoC
		}
		if cap > 0 {
			avail = append(avail, alloc{id: id, capacity: cap})
		}
	}
	return avail
}

func allocatePower(avail []alloc, res map[string]float64, residual, sign float64) float64 {
	remaining := residual
	for remaining > 0 && len(avail) > 0 {
		var totalCap float64
		for _, a := range avail {
			totalCap += a.capacity
		}
		if totalCap == 0 {
			break
		}
		next := avail[:0]
		for _, a := range avail {
			share := remaining * (a.capacity / totalCap)
			if share > a.capacity {
				share = a.capacity
			}
			res[a.id] += sign * share
			remaining -= share
			a.capacity -= share
			if a.capacity > 0 {
				next = append(next, a)
			}
		}
		avail = next
	}
	return remaining
}
