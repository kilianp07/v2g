package dispatch

import (
	"math"

	"github.com/kilianp07/v2g/core/logger"
	"github.com/kilianp07/v2g/core/model"
)

// ProbabilisticFallback redistributes power from failed vehicles to the others
// using availability probabilities and degradation factors.
type ProbabilisticFallback struct {
	vehicles map[string]model.Vehicle
	logger   logger.Logger
}

// NewProbabilisticFallback returns a fallback strategy based on probabilistic allocation.
func NewProbabilisticFallback(log logger.Logger) *ProbabilisticFallback {
	return &ProbabilisticFallback{vehicles: make(map[string]model.Vehicle), logger: log}
}

// SetVehicles stores the vehicles used for a dispatch so that the fallback can
// compute remaining capacity.
func (p *ProbabilisticFallback) SetVehicles(vs []model.Vehicle) {
	p.vehicles = make(map[string]model.Vehicle, len(vs))
	for _, v := range vs {
		p.vehicles[v.ID] = v
	}
}

// Reallocate implements FallbackStrategy. It redistributes residual power
// according to a weighted capacity accounting for SoC, availability and
// degradation.
func (p *ProbabilisticFallback) Reallocate(failed []model.Vehicle, current map[string]float64, signal model.FlexibilitySignal) map[string]float64 {
	res := make(map[string]float64, len(current))
	for id, pwr := range current {
		res[id] = pwr
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

	p.logger.Infof("fallback: reallocating %.2f kW after %d failures", residual*sign, len(failed))

	avail := p.availableCapacity(current, failedIDs, res)
	remaining := allocatePower(avail, res, residual, sign)

	efficiency := 1.0
	if residual > 0 {
		efficiency = (residual - remaining) / residual
	}
	if remaining > 0 {
		p.logger.Warnf("fallback: %.2f kW could not be reallocated (%.0f%% efficiency)", remaining*sign, efficiency*100)
		p.logger.Errorf("alert: fallback deficit %.2f kW for signal %.2f kW", remaining*sign, signal.PowerKW)
	} else {
		p.logger.Infof("fallback completed with %.0f%% efficiency", efficiency*100)
	}

	return res
}

func (p *ProbabilisticFallback) availableCapacity(current map[string]float64, failed map[string]struct{}, res map[string]float64) []alloc {
	var avail []alloc
	for id := range current {
		if _, ok := failed[id]; ok {
			continue
		}
		veh, ok := p.vehicles[id]
		if !ok {
			continue
		}

		cap := veh.EffectiveCapacity(res[id])
		if cap > 0 {
			avail = append(avail, alloc{id: id, capacity: cap})
		}
	}
	return avail
}
