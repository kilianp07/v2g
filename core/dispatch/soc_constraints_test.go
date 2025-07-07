package dispatch

import (
	"testing"
	"time"

	"github.com/kilianp07/v2g/core/model"
)

func TestSmartDispatcher_SoCThresholds(t *testing.T) {
	now := time.Now()
	disp := NewSmartDispatcher()
	disp.MinSoC = 0.1
	disp.SafeDischargeFloor = 0.1
	vehicles := []model.Vehicle{
		{ID: "v1", SoC: 0.1, BatteryKWh: 50, IsV2G: true, Available: true, MaxPower: 10, Departure: now.Add(time.Hour)},
		{ID: "v2", SoC: 0.09, BatteryKWh: 50, IsV2G: true, Available: true, MaxPower: 10, Departure: now.Add(time.Hour)},
	}
	sig := model.FlexibilitySignal{PowerKW: 5, Duration: time.Hour, Timestamp: now}
	asn := disp.Dispatch(vehicles, sig)
	if _, ok := asn["v1"]; !ok {
		t.Errorf("v1 should be included at threshold")
	}
	if _, ok := asn["v2"]; ok {
		t.Errorf("v2 should be excluded below threshold")
	}
}

func TestSmartDispatcher_SoCInsufficientEnergy(t *testing.T) {
	now := time.Now()
	disp := NewSmartDispatcher()
	disp.MinSoC = 0.0
	disp.SafeDischargeFloor = 0.1
	vehicles := []model.Vehicle{
		{ID: "v1", SoC: 0.2, BatteryKWh: 10, IsV2G: true, Available: true, MaxPower: 5, Departure: now.Add(time.Hour)},
	}
	sig := model.FlexibilitySignal{PowerKW: -5, Duration: 2 * time.Hour, Timestamp: now}
	asn := disp.Dispatch(vehicles, sig)
	if len(asn) != 0 {
		t.Errorf("vehicle should be excluded due to insufficient energy")
	}
}

func TestSmartDispatcher_SoCDisabled(t *testing.T) {
	now := time.Now()
	disp := NewSmartDispatcher()
	disp.EnableSoCConstraints = false
	disp.MinSoC = 0.1
	vehicles := []model.Vehicle{
		{ID: "v1", SoC: 0.05, BatteryKWh: 50, IsV2G: true, Available: true, MaxPower: 10, Departure: now.Add(time.Hour)},
	}
	sig := model.FlexibilitySignal{PowerKW: 5, Duration: time.Hour, Timestamp: now}
	asn := disp.Dispatch(vehicles, sig)
	if _, ok := asn["v1"]; !ok {
		t.Errorf("vehicle should not be excluded when constraints disabled")
	}
}
