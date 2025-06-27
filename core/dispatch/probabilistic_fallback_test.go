package dispatch

import (
	"testing"
	"time"

	"github.com/kilianp07/v2g/core/model"
	"github.com/kilianp07/v2g/infra/logger"
	"github.com/kilianp07/v2g/infra/mqtt"
	"github.com/kilianp07/v2g/internal/eventbus"
)

func TestProbabilisticFallback_Reallocate(t *testing.T) {
	fb := NewProbabilisticFallback(logger.NopLogger{})
	vs := []model.Vehicle{
		{ID: "v1", MaxPower: 50, SoC: 1, AvailabilityProb: 1},
		{ID: "v2", MaxPower: 100, SoC: 1, AvailabilityProb: 1, DegradationFactor: 0.1},
	}
	fb.SetVehicles(vs)
	current := map[string]float64{"v1": 30, "v2": 60}
	failed := []model.Vehicle{vs[0]}
	sig := model.FlexibilitySignal{PowerKW: 90}

	res := fb.Reallocate(failed, current, sig)
	if res["v1"] != 0 {
		t.Fatalf("expected v1=0 got %v", res["v1"])
	}
	if res["v2"] != 90 {
		t.Fatalf("expected v2=90 got %v", res["v2"])
	}
}

func TestProbabilisticFallback_SkipLowSoC(t *testing.T) {
	fb := NewProbabilisticFallback(logger.NopLogger{})
	vs := []model.Vehicle{
		{ID: "v1", MaxPower: 50, SoC: 1, AvailabilityProb: 1},
		{ID: "v2", MaxPower: 50, SoC: 0.25, AvailabilityProb: 1},
	}
	fb.SetVehicles(vs)
	current := map[string]float64{"v1": 30, "v2": 30}
	failed := []model.Vehicle{vs[0]}
	sig := model.FlexibilitySignal{PowerKW: 60}

	res := fb.Reallocate(failed, current, sig)
	if res["v2"] != 30 {
		t.Fatalf("expected v2 unchanged got %v", res["v2"])
	}
	if res["v1"] != 0 {
		t.Fatalf("expected v1 zero allocation")
	}
}

func TestProbabilisticFallback_AvailabilityWeight(t *testing.T) {
	fb := NewProbabilisticFallback(logger.NopLogger{})
	vs := []model.Vehicle{
		{ID: "v1", MaxPower: 50, SoC: 1, AvailabilityProb: 1},
		{ID: "v2", MaxPower: 50, SoC: 1, AvailabilityProb: 0.5},
	}
	fb.SetVehicles(vs)
	current := map[string]float64{"v1": 25, "v2": 25}
	failed := []model.Vehicle{vs[0]}
	sig := model.FlexibilitySignal{PowerKW: 50}

	res := fb.Reallocate(failed, current, sig)
	if res["v2"] != 37.5 {
		t.Fatalf("expected v2=37.5 got %v", res["v2"])
	}
}

func TestDispatchManager_ProbabilisticFallback(t *testing.T) {
	vehicles := []model.Vehicle{
		{ID: "v1", SoC: 0.8, IsV2G: true, Available: true, MaxPower: 50, AvailabilityProb: 1},
		{ID: "v2", SoC: 0.8, IsV2G: true, Available: true, MaxPower: 50, AvailabilityProb: 1},
	}
	publisher := mqtt.NewMockPublisher()
	publisher.FailIDs["v1"] = true
	fb := NewProbabilisticFallback(logger.NopLogger{})
	bus := eventbus.New()
	mgr, err := NewDispatchManager(SimpleVehicleFilter{}, EqualDispatcher{}, fb, publisher, time.Second, nil, bus, nil, logger.NopLogger{})
	if err != nil {
		t.Fatalf("manager: %v", err)
	}
	sig := model.FlexibilitySignal{Type: model.SignalFCR, PowerKW: 80, Timestamp: time.Now()}

	res := mgr.Dispatch(sig, vehicles)
	if res.FallbackAssignments["v2"] != 48 {
		t.Fatalf("expected reallocated 48 got %v", res.FallbackAssignments["v2"])
	}
	if res.FallbackAssignments["v1"] != 0 {
		t.Fatalf("expected v1 zero allocation")
	}
}
