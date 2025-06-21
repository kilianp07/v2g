package dispatch

import (
	"testing"
	"time"

	"github.com/kilianp07/v2g/model"
	"github.com/kilianp07/v2g/mqtt"
)

func TestSimpleVehicleFilter_FCR(t *testing.T) {
	vehicles := []model.Vehicle{
		{ID: "v1", SoC: 0.7, IsV2G: true, Available: true},
		{ID: "v2", SoC: 0.5, IsV2G: true, Available: true},
		{ID: "v3", SoC: 0.8, IsV2G: false, Available: true},
	}
	filter := SimpleVehicleFilter{}
	sig := model.FlexibilitySignal{Type: model.SignalFCR}

	res := filter.Filter(vehicles, sig)
	if len(res) != 1 || res[0].ID != "v1" {
		t.Fatalf("unexpected filter result: %+v", res)
	}
}

func TestEqualDispatcher_Dispatch(t *testing.T) {
	vehicles := []model.Vehicle{{ID: "v1", MaxPower: 40}, {ID: "v2", MaxPower: 60}}
	dispatcher := EqualDispatcher{}
	sig := model.FlexibilitySignal{PowerKW: 100}

	assignments := dispatcher.Dispatch(vehicles, sig)
	if len(assignments) != 2 {
		t.Fatalf("expected assignments for 2 vehicles")
	}
	if assignments["v1"] != 40 {
		t.Errorf("expected 40 kW for v1 got %v", assignments["v1"])
	}
	if assignments["v2"] != 60 {
		t.Errorf("expected 60 kW for v2 got %v", assignments["v2"])
	}
}

func TestDispatchManager_Dispatch(t *testing.T) {
	vehicles := []model.Vehicle{
		{ID: "v1", SoC: 0.7, IsV2G: true, Available: true, MaxPower: 50},
		{ID: "v2", SoC: 0.7, IsV2G: true, Available: true, MaxPower: 50},
	}
	publisher := mqtt.NewMockPublisher()
	manager, err := NewDispatchManager(SimpleVehicleFilter{}, EqualDispatcher{}, NoopFallback{}, publisher, time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sig := model.FlexibilitySignal{Type: model.SignalFCR, PowerKW: 80, Timestamp: time.Now()}

	res := manager.Dispatch(sig, vehicles)
	if len(res.Assignments) != 2 {
		t.Fatalf("expected 2 assignments")
	}
	if !res.Acknowledged["v1"] || !res.Acknowledged["v2"] {
		t.Fatalf("expected acknowledgments for both vehicles")
	}
	if publisher.Messages["v1"] == 0 || publisher.Messages["v2"] == 0 {
		t.Fatalf("publisher not invoked")
	}
}

func TestDispatchManager_Fallback(t *testing.T) {
	vehicles := []model.Vehicle{
		{ID: "v1", SoC: 0.7, IsV2G: true, Available: true, MaxPower: 50},
		{ID: "v2", SoC: 0.7, IsV2G: true, Available: true, MaxPower: 50},
	}
	publisher := mqtt.NewMockPublisher()
	publisher.FailIDs["v1"] = true
	manager, err := NewDispatchManager(SimpleVehicleFilter{}, EqualDispatcher{}, NoopFallback{}, publisher, time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sig := model.FlexibilitySignal{Type: model.SignalFCR, PowerKW: 80, Timestamp: time.Now()}

	res := manager.Dispatch(sig, vehicles)
	if res.Acknowledged["v1"] {
		t.Fatalf("v1 should have failed")
	}
	if res.FallbackAssignments == nil {
		t.Fatalf("expected fallback assignments")
	}
}
