package dispatch

import (
	"math"

	"github.com/kilianp07/v2g/core/logger"
	"github.com/kilianp07/v2g/core/model"
)

// SmartDispatcher allocates power using a weighted greedy strategy. Scores are
// computed from energy slack, time until departure and charging priority. The
// weights can be tuned and dynamically adapted based on the signal type. A
// participation score allows fairness between vehicles.
type SmartDispatcher struct {
	SocWeight            float64
	TimeWeight           float64
	PriorityWeight       float64
	PriceWeight          float64
	WearWeight           float64
	FairnessWeight       float64
	AvailabilityWeight   float64
	MarketPrice          float64
	Participation        map[string]float64
	MaxRounds            int
	scores               map[string]float64
	EnableSoCConstraints bool
	MinSoC               float64
	SafeDischargeFloor   float64
	Logger               logger.Logger
}

type candidate struct {
	v        model.Vehicle
	score    float64
	capacity float64
}

func (d SmartDispatcher) filterBySoC(vehicles []model.Vehicle, signal model.FlexibilitySignal) ([]model.Vehicle, []model.Vehicle) {
	if !d.EnableSoCConstraints {
		return vehicles, nil
	}
	var eligible []model.Vehicle
	var excluded []model.Vehicle
	for _, v := range vehicles {
		if v.SoC < d.MinSoC || v.BatteryKWh <= 0 {
			excluded = append(excluded, v)
			continue
		}
		energy, cap := availableEnergyAndCapacity(v, signal, true, d.SafeDischargeFloor)
		if signal.PowerKW < 0 && energy <= 0 {
			excluded = append(excluded, v)
			continue
		}
		// When only one vehicle is available and it cannot meet the full
		// request, skip it so fallback strategies can handle the deficit.
		if len(vehicles) == 1 && cap < math.Abs(signal.PowerKW) {
			excluded = append(excluded, v)
			continue
		}
		eligible = append(eligible, v)
	}
	return eligible, excluded
}

func (d SmartDispatcher) buildCandidates(vehicles []model.Vehicle, signal model.FlexibilitySignal, ctx *DispatchContext) ([]candidate, []model.Vehicle) {
	eligible, excluded := d.filterBySoC(vehicles, signal)
	return prepareVehicles(eligible, signal, ctx, d.vehicleScore, d.EnableSoCConstraints, d.SafeDischargeFloor), excluded
}

// NewSmartDispatcher returns a dispatcher with sensible default weights.
func NewSmartDispatcher() SmartDispatcher {
	return SmartDispatcher{
		SocWeight:            0.5,
		TimeWeight:           0.3,
		PriorityWeight:       0.1,
		PriceWeight:          0.05,
		WearWeight:           0.05,
		FairnessWeight:       0.05,
		AvailabilityWeight:   0.1,
		Participation:        make(map[string]float64),
		MaxRounds:            10,
		scores:               make(map[string]float64),
		EnableSoCConstraints: true,
		MinSoC:               0.1,
		SafeDischargeFloor:   0.1,
	}
}

func (d SmartDispatcher) weightsForSignal(t model.SignalType) (float64, float64, float64, float64, float64, float64, float64) {
	soc := d.SocWeight
	tm := d.TimeWeight
	prio := d.PriorityWeight
	price := d.PriceWeight
	wear := d.WearWeight
	fair := d.FairnessWeight
	avail := d.AvailabilityWeight
	switch t {
	case model.SignalFCR:
		// Emphasise immediate power capability
		soc += 0.2
		prio += 0.1
	case model.SignalNEBEF:
		// Availability over a longer window
		soc += 0.1
		tm += 0.2
	case model.SignalMA, model.SignalEcoWatt:
		soc += 0.1
	}
	return soc, tm, prio, price, wear, fair, avail
}

// vehicleScore computes the weighted score for a vehicle.
func (d SmartDispatcher) vehicleScore(v model.Vehicle, ctx *DispatchContext) float64 {
	socW, timeW, prioW, priceW, wearW, fairW, availW := d.weightsForSignal(ctx.Signal.Type)
	if v.BatteryKWh <= 0 {
		return 0
	}
	var energyNorm float64
	denom := 1 - v.MinSoC
	if denom == 0 {
		energyNorm = 0
	} else {
		energyNorm = (v.SoC - v.MinSoC) / denom
	}
	if energyNorm < 0 {
		energyNorm = 0
	}
	if energyNorm > 1 {
		energyNorm = 1
	}
	minutes := v.Departure.Sub(ctx.Now).Minutes()
	timeScore := 0.0
	if minutes > 0 {
		timeScore = math.Exp(-minutes / 30.0)
	}
	priority := 0.0
	if v.Priority {
		priority = 1.0
	}
	wear := ctx.GetParticipation(v.ID)
	score := energyNorm*socW + timeScore*timeW + priority*prioW + energyNorm*ctx.MarketPrice*priceW
	score += v.AvailabilityProb * availW
	score -= wear*wearW + wear*fairW
	if score < 0 {
		return 0
	}
	return score
}

// Dispatch implements the Dispatcher interface using the greedy weighted scores.
//
//gocyclo:ignore
func (d *SmartDispatcher) Dispatch(vehicles []model.Vehicle, signal model.FlexibilitySignal) map[string]float64 {
	assignments := make(map[string]float64)
	if len(vehicles) == 0 || signal.PowerKW == 0 {
		return assignments
	}

	ctx := &DispatchContext{Signal: signal, Now: signal.Timestamp, MarketPrice: d.MarketPrice, ParticipationScore: d.Participation}

	list, excluded := d.buildCandidates(vehicles, signal, ctx)
	if d.Logger != nil {
		for _, v := range excluded {
			if sl, ok := d.Logger.(logger.StructuredLogger); ok {
				sl.Debugw("soc_excluded", map[string]any{"vehicle": v.ID, "soc": v.SoC})
			}
			d.Logger.Infof("vehicle %s skipped due to SoC %.2f", v.ID, v.SoC)
		}
	}
	candCopy := append([]candidate(nil), list...)
	d.scores = make(map[string]float64, len(list))
	for _, c := range list {
		d.scores[c.v.ID] = c.score
	}
	if len(list) == 0 {
		return assignments
	}

	remaining := math.Abs(signal.PowerKW)
	sign := 1.0
	if signal.PowerKW < 0 {
		sign = -1
	}

	var weightSum float64
	for _, c := range list {
		weightSum += c.score
	}

	rounds := 0
	for remaining > 0 && len(list) > 0 && weightSum > 0 && (d.MaxRounds == 0 || rounds < d.MaxRounds) {
		var consumed float64
		list, weightSum, remaining, consumed = d.allocateRound(list, weightSum, sign, remaining, assignments)
		if consumed == 0 {
			break
		}
		rounds++
	}
	if d.Logger != nil {
		for _, c := range candCopy {
			if p, ok := assignments[c.v.ID]; ok && p != 0 {
				if sl, ok := d.Logger.(logger.StructuredLogger); ok {
					sl.Debugw("vehicle_selected", map[string]any{"vehicle": c.v.ID, "score": c.score, "power": p})
				}
				d.Logger.Infof("vehicle %s selected soc=%.2f power=%.2f", c.v.ID, c.v.SoC, p)
			}
		}
	}
	return assignments
}

func (d SmartDispatcher) allocateRound(list []candidate, weightSum, sign, remaining float64, assignments map[string]float64) ([]candidate, float64, float64, float64) {
	consumed := 0.0
	next := list[:0]
	for _, c := range list {
		if remaining <= 0 || weightSum <= 0 {
			break
		}
		share := remaining * (c.score / weightSum)
		if share >= c.capacity {
			assignments[c.v.ID] += sign * c.capacity
			consumed += c.capacity
			remaining -= c.capacity
			weightSum -= c.score
		} else {
			assignments[c.v.ID] += sign * share
			c.capacity -= share
			consumed += share
			remaining -= share
			next = append(next, c)
		}
	}
	return next, weightSum, remaining, consumed
}

// GetScores implements ScoringDispatcher by returning the last computed scores.
func (d *SmartDispatcher) GetScores() map[string]float64 {
	cp := make(map[string]float64, len(d.scores))
	for k, v := range d.scores {
		cp[k] = v
	}
	return cp
}

// GetMarketPrice implements MarketPriceProvider by returning the configured market price.
func (d *SmartDispatcher) GetMarketPrice() float64 {
	return d.MarketPrice
}
