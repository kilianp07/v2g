package dispatch

import (
	"math"

	"github.com/kilianp07/v2g/model"
	"gonum.org/v1/gonum/mat"
	"gonum.org/v1/gonum/optimize/convex/lp"
)

// LPDispatcher solves a linear program to optimally distribute power
// according to SmartDispatcher scores.
type LPDispatcher struct {
	SmartDispatcher
}

type lpData struct {
	ids    []string
	scores []float64
	caps   []float64
}

func (d LPDispatcher) buildData(vehicles []model.Vehicle, signal model.FlexibilitySignal, ctx DispatchContext) lpData {
	var data lpData
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
		data.ids = append(data.ids, v.ID)
		data.scores = append(data.scores, d.vehicleScore(v, ctx))
		data.caps = append(data.caps, cap)
	}
	return data
}

func solveLP(scores, caps []float64, target float64) ([]float64, error) {
	c := make([]float64, len(scores))
	for i, s := range scores {
		c[i] = -s
	}

	g := mat.NewDense(len(caps), len(caps), nil)
	h := make([]float64, len(caps))
	for i, cap := range caps {
		g.Set(i, i, 1)
		h[i] = cap
	}

	A := mat.NewDense(1, len(caps), nil)
	for i := range caps {
		A.Set(0, i, 1)
	}
	b := []float64{target}

	cStd, AStd, bStd := lp.Convert(c, g, h, A, b)
	_, sol, err := lp.Simplex(cStd, AStd, bStd, 1e-7, nil)
	return sol, err
}

// NewLPDispatcher returns an LP-based dispatcher with default weights.
func NewLPDispatcher() LPDispatcher {
	return LPDispatcher{SmartDispatcher: NewSmartDispatcher()}
}

// Dispatch implements the Dispatcher interface. It solves
// a linear program maximizing the weighted score while meeting
// the power target.
func (d LPDispatcher) Dispatch(vehicles []model.Vehicle, signal model.FlexibilitySignal) map[string]float64 {
	assignments := make(map[string]float64)
	if len(vehicles) == 0 || signal.PowerKW == 0 {
		return assignments
	}

	ctx := DispatchContext{
		Signal:             signal,
		Now:                signal.Timestamp,
		MarketPrice:        d.MarketPrice,
		ParticipationScore: d.Participation,
	}

	data := d.buildData(vehicles, signal, ctx)
	if len(data.ids) == 0 {
		return assignments
	}

	target := math.Abs(signal.PowerKW)
	sign := 1.0
	if signal.PowerKW < 0 {
		sign = -1
	}

	sol, err := solveLP(data.scores, data.caps, target)
	if err != nil {
		gd := d.SmartDispatcher
		return gd.Dispatch(vehicles, signal)
	}

	n := len(data.scores)
	for i, id := range data.ids {
		power := sol[i]
		if i+n < len(sol) {
			power -= sol[i+n]
		}
		assignments[id] = sign * power
	}
	return assignments
}
