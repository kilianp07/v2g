package dispatch

import (
	"testing"
	"time"

	"github.com/kilianp07/v2g/model"
)

func TestLPDispatcher_Dispatch(t *testing.T) {
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
