package dispatch

import (
	"errors"
	"testing"
	"time"

	"github.com/kilianp07/v2g/core/model"
)

func TestSegmentedDispatcher_DefaultSegments(t *testing.T) {
	sd := NewSegmentedSmartDispatcher(nil)
	now := time.Now()
	vehicles := []model.Vehicle{
		{ID: "c1", Segment: "commuter", SoC: 0.9, BatteryKWh: 40, MaxPower: 10, IsV2G: true, Available: true, Departure: now.Add(time.Hour)},
		{ID: "f1", Segment: "captive_fleet", SoC: 0.9, BatteryKWh: 40, MaxPower: 10, IsV2G: true, Available: true, Departure: now.Add(time.Hour)},
		{ID: "o1", Segment: "opportunistic_charger", SoC: 0.9, BatteryKWh: 40, MaxPower: 10, IsV2G: true, Available: true, Departure: now.Add(time.Hour)},
	}
	called := 0
	old := lpSolve
	lpSolve = func(scores, caps []float64, target float64) ([]float64, error) {
		called++
		sol := make([]float64, len(scores))
		share := target / float64(len(scores))
		for i := range sol {
			sol[i] = share
		}
		return sol, nil
	}
	defer func() { lpSolve = old }()

	sig := model.FlexibilitySignal{PowerKW: 9, Duration: time.Hour, Timestamp: now}
	asn := sd.Dispatch(vehicles, sig)
	if len(asn) != 3 {
		t.Fatalf("expected assignments for 3 vehicles got %d", len(asn))
	}
	if called != 1 {
		t.Fatalf("expected LP solver called once got %d", called)
	}
}

func TestSegmentedDispatcher_LPFallback(t *testing.T) {
	cfg := DefaultSegmentConfigs()
	d := NewSegmentedSmartDispatcher(cfg)
	now := time.Now()
	vehicles := []model.Vehicle{
		{ID: "f1", Segment: "captive_fleet", SoC: 0.9, BatteryKWh: 40, MaxPower: 10, IsV2G: true, Available: true, Departure: now.Add(time.Hour)},
	}
	old := lpSolve
	lpSolve = func(_, _ []float64, _ float64) ([]float64, error) { return nil, errors.New("fail") }
	defer func() { lpSolve = old }()
	sig := model.FlexibilitySignal{PowerKW: 5, Duration: time.Hour, Timestamp: now}
	asn := d.Dispatch(vehicles, sig)
	if asn["f1"] == 0 {
		t.Fatalf("expected fallback allocation")
	}
}

func TestSegmentedDispatcher_UnknownSegment(t *testing.T) {
	d := NewSegmentedSmartDispatcher(nil)
	now := time.Now()
	vehicles := []model.Vehicle{{ID: "u1", Segment: "unknown", SoC: 0.8, BatteryKWh: 40, MaxPower: 10, IsV2G: true, Available: true, Departure: now.Add(time.Hour)}}
	sig := model.FlexibilitySignal{PowerKW: 5, Duration: time.Hour, Timestamp: now}
	asn := d.Dispatch(vehicles, sig)
	if asn["u1"] == 0 {
		t.Fatalf("expected default smart allocation")
	}
}

func TestApplyWeights(t *testing.T) {
	sd := NewSmartDispatcher()
	ws := map[string]float64{
		"soc":          1,
		"time":         2,
		"priority":     3,
		"price":        4,
		"wear":         5,
		"fairness":     6,
		"availability": 7,
		"market_price": 9,
	}
	applyWeights(&sd, ws)
	if sd.SocWeight != 1 || sd.TimeWeight != 2 || sd.PriorityWeight != 3 || sd.PriceWeight != 4 || sd.WearWeight != 5 || sd.FairnessWeight != 6 || sd.AvailabilityWeight != 7 || sd.MarketPrice != 9 {
		t.Fatalf("weights not applied correctly")
	}
}
