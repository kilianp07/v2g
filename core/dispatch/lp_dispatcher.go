package dispatch

import (
	"errors"
	"math"

	"github.com/kilianp07/v2g/core/model"
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

func (d LPDispatcher) buildData(vehicles []model.Vehicle, signal model.FlexibilitySignal, ctx *DispatchContext) lpData {
	cands := prepareVehicles(vehicles, signal, ctx, d.vehicleScore)
	data := lpData{ids: make([]string, len(cands)), scores: make([]float64, len(cands)), caps: make([]float64, len(cands))}
	for i, c := range cands {
		data.ids[i] = c.v.ID
		data.scores[i] = c.score
		data.caps[i] = c.capacity
	}
	return data
}

// solveLP runs the simplex algorithm to maximise the weighted score subject to
// capacity constraints.
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

// lpSolve points to the function used to solve the LP. It can be overridden in
// tests to simulate solver failures.
var lpSolve = solveLP

// ErrInfeasible indicates the LP had no feasible solution meeting the target.
var ErrInfeasible = errors.New("lp infeasible")

// NewLPDispatcher returns an LP-based dispatcher with default weights.
func NewLPDispatcher() LPDispatcher {
	return LPDispatcher{SmartDispatcher: NewSmartDispatcher()}
}

// DispatchStrict solves the LP and returns an error if the solver fails or the
// target cannot be met. No fallback to SmartDispatcher is applied.
func (d *LPDispatcher) DispatchStrict(vehicles []model.Vehicle, signal model.FlexibilitySignal) (map[string]float64, error) {
	assignments := make(map[string]float64)
	if len(vehicles) == 0 || signal.PowerKW == 0 {
		return assignments, nil
	}

	ctx := &DispatchContext{
		Signal:             signal,
		Now:                signal.Timestamp,
		MarketPrice:        d.MarketPrice,
		ParticipationScore: d.Participation,
	}

	data := d.buildData(vehicles, signal, ctx)
	d.scores = make(map[string]float64, len(data.ids))
	for i, id := range data.ids {
		d.scores[id] = data.scores[i]
	}
	if len(data.ids) == 0 {
		return assignments, nil
	}

	target := math.Abs(signal.PowerKW)
	sign := 1.0
	if signal.PowerKW < 0 {
		sign = -1
	}

	sol, err := lpSolve(data.scores, data.caps, target)
	if err != nil {
		return nil, err
	}

	var sum float64
	for i, id := range data.ids {
		power := sol[i]
		if power < 0 {
			power = 0
		}
		if power > data.caps[i] {
			power = data.caps[i]
		}
		assignments[id] = sign * power
		sum += power
	}
	if math.Abs(sum-target) > 1e-3 {
		return assignments, ErrInfeasible
	}
	return assignments, nil
}

// Dispatch implements the Dispatcher interface. It solves
// a linear program maximizing the weighted score while meeting
// the power target.
func (d *LPDispatcher) Dispatch(vehicles []model.Vehicle, signal model.FlexibilitySignal) map[string]float64 {
	asn, err := d.DispatchStrict(vehicles, signal)
	if err != nil {
		gd := d.SmartDispatcher
		return gd.Dispatch(vehicles, signal)
	}
	return asn
}

// GetScores returns the last computed scores for vehicles.
func (d *LPDispatcher) GetScores() map[string]float64 {
	cp := make(map[string]float64, len(d.scores))
	for k, v := range d.scores {
		cp[k] = v
	}
	return cp
}
