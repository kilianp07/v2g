package test

import (
	"testing"
	"time"

	"github.com/kilianp07/v2g/core/dispatch"
	"github.com/kilianp07/v2g/core/model"
	"github.com/kilianp07/v2g/infra/logger"
	"github.com/kilianp07/v2g/infra/mqtt"
	"github.com/kilianp07/v2g/internal/eventbus"
)

// TestSegmentedDispatcherIntegration exercises dispatch across segments.
func TestSegmentedDispatcherIntegration(t *testing.T) {
	d := dispatch.NewSegmentedSmartDispatcher(nil)
	pub := mqtt.NewMockPublisher()
	bus := eventbus.New()
	mgr, err := dispatch.NewDispatchManager(
		dispatch.SimpleVehicleFilter{},
		d,
		dispatch.NoopFallback{},
		pub,
		time.Second,
		nil,
		bus,
		nil,
		logger.NopLogger{},
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("manager: %v", err)
	}

	vehicles := []model.Vehicle{
		{ID: "c1", Segment: "commuter", SoC: 0.9, BatteryKWh: 40, MaxPower: 10, IsV2G: true, Available: true},
		{ID: "f1", Segment: "captive_fleet", SoC: 0.9, BatteryKWh: 40, MaxPower: 5, IsV2G: true, Available: true},
	}
	sig := model.FlexibilitySignal{Type: model.SignalFCR, PowerKW: 10, Duration: time.Hour, Timestamp: time.Now()}
	res := mgr.Dispatch(sig, vehicles)
	if len(res.Assignments) != 2 {
		t.Fatalf("expected two assignments got %d", len(res.Assignments))
	}
	total := 0.0
	for _, p := range res.Assignments {
		total += p
	}
	if total < 9.9 || total > 10.1 {
		t.Fatalf("expected ~10kW total got %v", total)
	}
}
