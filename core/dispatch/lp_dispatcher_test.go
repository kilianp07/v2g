package dispatch

import (
	"errors"
	"testing"
	"time"

	"github.com/kilianp07/v2g/core/model"
)

func TestLPDispatcher_Basic(t *testing.T) {
	now := time.Now()
	vehicles := []model.Vehicle{
		{ID: "v1", SoC: 1.0, MinSoC: 0.3, BatteryKWh: 100, IsV2G: true, Available: true, MaxPower: 70, Departure: now.Add(2 * time.Hour)},
		{ID: "v2", SoC: 1.0, MinSoC: 0.3, BatteryKWh: 100, IsV2G: true, Available: true, MaxPower: 70, Departure: now.Add(2 * time.Hour)},
	}
	sig := model.FlexibilitySignal{PowerKW: 60, Duration: time.Hour, Timestamp: now}

	disp := NewLPDispatcher()
	assignments := disp.Dispatch(vehicles, sig)
	if len(assignments) != 2 {
		t.Fatalf("expected 2 assignments got %d", len(assignments))
	}
	if assignments["v1"]+assignments["v2"] != 60 {
		t.Fatalf("expected total 60 got %v", assignments)
	}
}

func TestLPDispatcher_VaryingCapacity(t *testing.T) {
	now := time.Now()
	vehicles := []model.Vehicle{
		{ID: "v1", SoC: 1, MinSoC: 0.2, BatteryKWh: 50, IsV2G: true, Available: true, MaxPower: 20, Departure: now.Add(3 * time.Hour)},
		{ID: "v2", SoC: 0.8, MinSoC: 0.3, BatteryKWh: 60, IsV2G: true, Available: true, MaxPower: 40, Departure: now.Add(3 * time.Hour)},
	}
	sig := model.FlexibilitySignal{PowerKW: 50, Duration: time.Hour, Timestamp: now}

	disp := NewLPDispatcher()
	asn := disp.Dispatch(vehicles, sig)
	total := asn["v1"] + asn["v2"]
	if total != 50 {
		t.Fatalf("expected total 50 got %v", total)
	}
	if asn["v1"] > 20 || asn["v2"] > 40 {
		t.Fatalf("assignment exceeds capacity: %v", asn)
	}
}

func TestLPDispatcher_InsufficientCapacity(t *testing.T) {
	now := time.Now()
	vehicles := []model.Vehicle{
		{ID: "v1", SoC: 0.9, MinSoC: 0.4, BatteryKWh: 40, IsV2G: true, Available: true, MaxPower: 20, Departure: now.Add(2 * time.Hour)},
	}
	sig := model.FlexibilitySignal{PowerKW: 50, Duration: time.Hour, Timestamp: now}

	disp := NewLPDispatcher()
	asn := disp.Dispatch(vehicles, sig)
	if asn["v1"] > 20 {
		t.Fatalf("expected at most 20 got %v", asn["v1"])
	}
}

func TestLPDispatcher_NoVehicles(t *testing.T) {
	disp := NewLPDispatcher()
	asn := disp.Dispatch(nil, model.FlexibilitySignal{PowerKW: 10})
	if len(asn) != 0 {
		t.Fatalf("expected empty map")
	}
}

func TestLPDispatcher_SolverErrorFallback(t *testing.T) {
	old := lpSolve
	lpSolve = func(_, _ []float64, _ float64) ([]float64, error) { return nil, errors.New("fail") }
	defer func() { lpSolve = old }()

	now := time.Now()
	vehicles := []model.Vehicle{{ID: "v1", SoC: 1, MinSoC: 0.5, BatteryKWh: 50, IsV2G: true, Available: true, MaxPower: 30, Departure: now.Add(time.Hour)}}
	sig := model.FlexibilitySignal{PowerKW: 20, Duration: time.Hour, Timestamp: now}

	disp := NewLPDispatcher()
	asn := disp.Dispatch(vehicles, sig)
	if asn["v1"] == 0 {
		t.Fatalf("fallback should still allocate")
	}
}
