package dispatch

import (
	"testing"
	"time"

	"github.com/kilianp07/v2g/core/model"
)

func TestSmartDispatcher_FairnessPenalty(t *testing.T) {
	now := time.Now()
	vehicles := []model.Vehicle{
		{ID: "v1", SoC: 0.9, MinSoC: 0.5, BatteryKWh: 60, IsV2G: true, Available: true, MaxPower: 40, Departure: now.Add(4 * time.Hour)},
		{ID: "v2", SoC: 0.8, MinSoC: 0.5, BatteryKWh: 60, IsV2G: true, Available: true, MaxPower: 40, Departure: now.Add(1 * time.Hour)},
	}
	sig := model.FlexibilitySignal{PowerKW: 60, Duration: time.Hour, Timestamp: now}

	base := NewSmartDispatcher()
	baseAssignments := base.Dispatch(vehicles, sig)

	dispatcher := NewSmartDispatcher()
	dispatcher.FairnessWeight = 0.5
	dispatcher.Participation["v1"] = 10

	assignments := dispatcher.Dispatch(vehicles, sig)
	if assignments["v1"] >= baseAssignments["v1"] {
		t.Errorf("participation penalty should reduce v1 allocation")
	}
}

func TestSmartDispatcher_SoCWeight(t *testing.T) {
	now := time.Now()
	vehicles := []model.Vehicle{
		{ID: "h", SoC: 1.0, MinSoC: 0.2, BatteryKWh: 100, IsV2G: true, Available: true, MaxPower: 40, Departure: now.Add(2 * time.Hour)},
		{ID: "l", SoC: 0.7, MinSoC: 0.5, BatteryKWh: 100, IsV2G: true, Available: true, MaxPower: 40, Departure: now.Add(2 * time.Hour)},
	}
	sig := model.FlexibilitySignal{PowerKW: 50, Duration: time.Hour, Timestamp: now}

	disp := NewSmartDispatcher()
	disp.SocWeight = 1
	disp.TimeWeight = 0
	asn := disp.Dispatch(vehicles, sig)
	if asn["h"] <= asn["l"] {
		t.Fatalf("expected high SoC vehicle to get more power")
	}
}

func TestSmartDispatcher_TimeWeight(t *testing.T) {
	now := time.Now()
	vehicles := []model.Vehicle{
		{ID: "soon", SoC: 1.0, MinSoC: 0.2, BatteryKWh: 100, IsV2G: true, Available: true, MaxPower: 40, Departure: now.Add(time.Hour)},
		{ID: "later", SoC: 1.0, MinSoC: 0.2, BatteryKWh: 100, IsV2G: true, Available: true, MaxPower: 40, Departure: now.Add(6 * time.Hour)},
	}
	sig := model.FlexibilitySignal{PowerKW: 40, Duration: time.Hour, Timestamp: now}

	disp := NewSmartDispatcher()
	disp.TimeWeight = 1
	disp.SocWeight = 0
	asn := disp.Dispatch(vehicles, sig)
	if asn["soon"] <= asn["later"] {
		t.Fatalf("expected vehicle leaving sooner to get more power")
	}
}

func TestSmartDispatcher_PriorityWeight(t *testing.T) {
	now := time.Now()
	vehicles := []model.Vehicle{
		{ID: "prio", SoC: 1.0, MinSoC: 0.2, BatteryKWh: 100, IsV2G: true, Available: true, MaxPower: 40, Priority: true, Departure: now.Add(2 * time.Hour)},
		{ID: "norm", SoC: 1.0, MinSoC: 0.2, BatteryKWh: 100, IsV2G: true, Available: true, MaxPower: 40, Departure: now.Add(2 * time.Hour)},
	}
	sig := model.FlexibilitySignal{PowerKW: 40, Duration: time.Hour, Timestamp: now}

	disp := NewSmartDispatcher()
	disp.PriorityWeight = 1
	disp.SocWeight = 0
	disp.TimeWeight = 0
	asn := disp.Dispatch(vehicles, sig)
	if asn["prio"] <= asn["norm"] {
		t.Fatalf("expected priority vehicle to get more power")
	}
}

func TestSmartDispatcher_MaxRounds(t *testing.T) {
	now := time.Now()
	vehicles := []model.Vehicle{
		{ID: "v1", SoC: 1, MinSoC: 0.2, BatteryKWh: 100, IsV2G: true, Available: true, MaxPower: 5, Departure: now.Add(2 * time.Hour)},
		{ID: "v2", SoC: 1, MinSoC: 0.2, BatteryKWh: 100, IsV2G: true, Available: true, MaxPower: 5, Departure: now.Add(2 * time.Hour)},
		{ID: "v3", SoC: 1, MinSoC: 0.2, BatteryKWh: 100, IsV2G: true, Available: true, MaxPower: 20, Departure: now.Add(2 * time.Hour)},
	}
	sig := model.FlexibilitySignal{PowerKW: 40, Duration: time.Hour, Timestamp: now}

	disp := NewSmartDispatcher()
	disp.MaxRounds = 1
	asn := disp.Dispatch(vehicles, sig)
	total := asn["v1"] + asn["v2"] + asn["v3"]
	if total != 30 {
		t.Fatalf("expected allocation capped at capacity got %v", total)
	}
}
