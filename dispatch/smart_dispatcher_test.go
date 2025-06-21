package dispatch

import (
	"testing"
	"time"

	"github.com/kilianp07/v2g/model"
)

func TestSmartDispatcher_Dispatch(t *testing.T) {
	now := time.Now()
	vehicles := []model.Vehicle{
		{ID: "v1", SoC: 0.9, MinSoC: 0.5, BatteryKWh: 60, IsV2G: true, Available: true, MaxPower: 40, Departure: now.Add(4 * time.Hour)},
		{ID: "v2", SoC: 0.8, MinSoC: 0.5, BatteryKWh: 60, IsV2G: true, Available: true, MaxPower: 40, Departure: now.Add(1 * time.Hour)},
		{ID: "v3", SoC: 0.7, MinSoC: 0.5, BatteryKWh: 60, IsV2G: true, Available: true, MaxPower: 40, Departure: now.Add(6 * time.Hour)},
	}
	sig := model.FlexibilitySignal{PowerKW: 60, Duration: time.Hour, Timestamp: now}

	base := NewSmartDispatcher()
	baseAssignments := base.Dispatch(vehicles, sig)

	dispatcher := NewSmartDispatcher()
	dispatcher.FairnessWeight = 0.5
	dispatcher.Participation["v1"] = 10 // simulate high previous use

	assignments := dispatcher.Dispatch(vehicles, sig)
	if len(assignments) != 3 {
		t.Fatalf("expected assignments for 3 vehicles")
	}
	if assignments["v1"] >= baseAssignments["v1"] {
		t.Errorf("participation penalty should reduce v1 allocation")
	}
}
