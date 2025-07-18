package dispatch

import (
	"fmt"
	"testing"
	"time"

	"github.com/kilianp07/v2g/core/model"
	coremqtt "github.com/kilianp07/v2g/core/mqtt"
	"github.com/kilianp07/v2g/infra/logger"
	imqtt "github.com/kilianp07/v2g/infra/mqtt"
	"github.com/kilianp07/v2g/internal/eventbus"
)

func TestAckBasedTuner_Increase(t *testing.T) {
	disp := NewSmartDispatcher()
	disp.AvailabilityWeight = 0
	tuner := NewAckBasedTuner(&disp)

	history := []DispatchResult{{
		Assignments:  map[string]float64{"v1": 10, "v2": 10},
		Acknowledged: map[string]bool{"v1": true, "v2": true},
		Errors:       map[string]error{},
	}}

	tuner.Tune(history)
	if disp.AvailabilityWeight <= 0 {
		t.Fatalf("expected weight increase")
	}
}

func TestAckBasedTuner_Decrease(t *testing.T) {
	disp := NewSmartDispatcher()
	disp.AvailabilityWeight = 0.5
	tuner := NewAckBasedTuner(&disp)

	history := []DispatchResult{{
		Assignments:  map[string]float64{"v1": 10},
		Acknowledged: map[string]bool{"v1": false},
		Errors:       map[string]error{"v1": fmt.Errorf("%w", coremqtt.ErrAckTimeout)},
	}}

	tuner.Tune(history)
	if disp.AvailabilityWeight >= 0.5 {
		t.Fatalf("expected weight decrease")
	}
}

func TestDispatchManager_NoTuner(t *testing.T) {
	disp := NewSmartDispatcher()
	dispatcher := &disp
	publisher := imqtt.NewMockPublisher()
	bus := eventbus.New()
	mgr, err := NewDispatchManager(SimpleVehicleFilter{}, dispatcher, NoopFallback{}, publisher, time.Second, nil, bus, nil, logger.NopLogger{}, nil, nil)
	if err != nil {
		t.Fatalf("manager: %v", err)
	}
	v := model.Vehicle{ID: "v1", SoC: 1, BatteryKWh: 50, IsV2G: true, Available: true, MaxPower: 40, Departure: time.Now().Add(time.Hour)}
	sig := model.FlexibilitySignal{PowerKW: 20, Timestamp: time.Now()}
	mgr.Dispatch(sig, []model.Vehicle{v})
	if disp.AvailabilityWeight != dispatcher.AvailabilityWeight {
		t.Fatalf("expected weight unchanged")
	}
}

func TestAckBasedTuner_Integration(t *testing.T) {
	disp := NewSmartDispatcher()
	disp.SocWeight = 1
	disp.TimeWeight = 0
	disp.AvailabilityWeight = 0
	tuner := NewAckBasedTuner(&disp)

	publisher := imqtt.NewMockPublisher()
	bus := eventbus.New()
	mgr, err := NewDispatchManager(SimpleVehicleFilter{}, &disp, NoopFallback{}, publisher, time.Second, nil, bus, nil, logger.NopLogger{}, tuner, nil)
	if err != nil {
		t.Fatalf("manager: %v", err)
	}

	vehicles := []model.Vehicle{
		{ID: "v1", SoC: 0.8, BatteryKWh: 40, IsV2G: true, Available: true, MaxPower: 20, Departure: time.Now().Add(time.Hour), AvailabilityProb: 1},
		{ID: "v2", SoC: 0.8, BatteryKWh: 40, IsV2G: true, Available: true, MaxPower: 20, Departure: time.Now().Add(time.Hour), AvailabilityProb: 0.5},
	}
	sig := model.FlexibilitySignal{PowerKW: 20, Timestamp: time.Now()}
	mgr.Dispatch(sig, vehicles) // first dispatch tunes weights
	res := mgr.Dispatch(sig, vehicles)

	if res.Assignments["v1"] <= res.Assignments["v2"] {
		t.Fatalf("expected higher allocation for v1 after tuning")
	}
}

func TestNewAckBasedTunerWithConfig_Validation(t *testing.T) {
	disp := NewSmartDispatcher()
	if NewAckBasedTunerWithConfig(nil, 0.1, 0.1, 1, 0, DefaultAckThreshold) != nil {
		t.Fatal("expected nil with nil dispatcher")
	}
	if NewAckBasedTunerWithConfig(&disp, -0.1, 0.1, 1, 0, DefaultAckThreshold) != nil {
		t.Fatal("expected nil with negative increase")
	}
	if NewAckBasedTunerWithConfig(&disp, 0.1, -0.1, 1, 0, DefaultAckThreshold) != nil {
		t.Fatal("expected nil with negative decrease")
	}
	if NewAckBasedTunerWithConfig(&disp, 0.1, 0.1, 0, 1, DefaultAckThreshold) != nil {
		t.Fatal("expected nil with max < min")
	}
	tnr := NewAckBasedTunerWithConfig(&disp, 0.1, 0.1, 1, 0, DefaultAckThreshold)
	if tnr == nil {
		t.Fatal("expected tuner instance")
	}
}
