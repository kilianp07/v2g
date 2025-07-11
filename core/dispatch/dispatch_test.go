package dispatch

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/kilianp07/v2g/core/events"
	coremetrics "github.com/kilianp07/v2g/core/metrics"
	"github.com/kilianp07/v2g/core/model"
	"github.com/kilianp07/v2g/infra/logger"
	"github.com/kilianp07/v2g/infra/metrics"
	"github.com/kilianp07/v2g/infra/mqtt"
	"github.com/kilianp07/v2g/internal/eventbus"
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
	reg := prometheus.NewRegistry()
	sinkIf, errSink := metrics.NewPromSinkWithRegistry(coremetrics.Config{}, reg)
	if errSink != nil {
		t.Fatalf("prom sink: %v", errSink)
	}
	promSink, ok := sinkIf.(*metrics.PromSink)
	if !ok {
		t.Fatalf("expected PromSink")
	}
	bus := eventbus.New()
	manager, err := NewDispatchManager(SimpleVehicleFilter{}, EqualDispatcher{}, NoopFallback{}, publisher, time.Second, promSink, bus, nil, logger.NopLogger{}, nil, nil)
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
	bus := eventbus.New()
	manager, err := NewDispatchManager(SimpleVehicleFilter{}, EqualDispatcher{}, NoopFallback{}, publisher, time.Second, nil, bus, nil, logger.NopLogger{}, nil, nil)
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

func TestDispatchManager_Events(t *testing.T) {
	vehicles := []model.Vehicle{
		{ID: "v1", SoC: 0.7, IsV2G: true, Available: true, MaxPower: 40},
	}
	publisher := mqtt.NewMockPublisher()
	bus := eventbus.New()
	ch := bus.Subscribe()
	defer bus.Unsubscribe(ch)

	mgr, err := NewDispatchManager(SimpleVehicleFilter{}, EqualDispatcher{}, NoopFallback{}, publisher, time.Second, nil, bus, nil, logger.NopLogger{}, nil, nil)
	if err != nil {
		t.Fatalf("manager: %v", err)
	}
	sig := model.FlexibilitySignal{Type: model.SignalFCR, PowerKW: 10, Timestamp: time.Now()}

	done := make(chan struct{})
	go func() {
		var gotSignal, gotAck bool
		for evt := range ch {
			switch evt.(type) {
			case events.SignalEvent:
				gotSignal = true
			case events.AckEvent:
				gotAck = true
			}
			if gotSignal && gotAck {
				close(done)
				return
			}
		}
	}()

	mgr.Dispatch(sig, vehicles)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("events not received")
	}
}
