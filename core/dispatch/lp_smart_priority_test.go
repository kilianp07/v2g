package dispatch

import (
	"errors"
	"testing"
	"time"

	"github.com/kilianp07/v2g/core/events"
	"github.com/kilianp07/v2g/core/model"
	"github.com/kilianp07/v2g/infra/logger"
	"github.com/kilianp07/v2g/infra/mqtt"
	"github.com/kilianp07/v2g/internal/eventbus"
)

type countingSmart struct {
	SmartDispatcher
	called int
}

func (c *countingSmart) Dispatch(v []model.Vehicle, s model.FlexibilitySignal) map[string]float64 {
	c.called++
	return c.SmartDispatcher.Dispatch(v, s)
}

func TestDispatchManager_LPFirstSuccess(t *testing.T) {
	publisher := mqtt.NewMockPublisher()
	bus := eventbus.New()

	smart := &countingSmart{SmartDispatcher: NewSmartDispatcher()}
	mgr, err := NewDispatchManager(SimpleVehicleFilter{}, smart, NoopFallback{}, publisher, time.Second, nil, bus, nil, logger.NopLogger{}, nil)
	if err != nil {
		t.Fatalf("manager: %v", err)
	}
	lp := NewLPDispatcher()
	mgr.lpDispatcher = &lp
	mgr.SetLPFirst(map[model.SignalType]bool{model.SignalFCR: true})

	sig := model.FlexibilitySignal{Type: model.SignalFCR, PowerKW: 10, Timestamp: time.Now()}
	veh := model.Vehicle{ID: "v1", SoC: 1, IsV2G: true, Available: true, MaxPower: 10, BatteryKWh: 50}

	mgr.Dispatch(sig, []model.Vehicle{veh})
	if smart.called != 0 {
		t.Fatalf("smart dispatcher should not be used on success")
	}
}

func TestDispatchManager_LPFailureFallback(t *testing.T) {
	publisher := mqtt.NewMockPublisher()
	bus := eventbus.New()

	smart := &countingSmart{SmartDispatcher: NewSmartDispatcher()}
	mgr, err := NewDispatchManager(SimpleVehicleFilter{}, smart, NoopFallback{}, publisher, time.Second, nil, bus, nil, logger.NopLogger{}, nil)
	if err != nil {
		t.Fatalf("manager: %v", err)
	}
	lp := NewLPDispatcher()
	mgr.lpDispatcher = &lp
	mgr.SetLPFirst(map[model.SignalType]bool{model.SignalFCR: true})

	old := lpSolve
	lpSolve = func(_, _ []float64, _ float64) ([]float64, error) { return nil, errors.New("fail") }
	defer func() { lpSolve = old }()

	sig := model.FlexibilitySignal{Type: model.SignalFCR, PowerKW: 10, Timestamp: time.Now()}
	veh := model.Vehicle{ID: "v1", SoC: 1, IsV2G: true, Available: true, MaxPower: 10, BatteryKWh: 50}

	ch := bus.Subscribe()
	defer bus.Unsubscribe(ch)
	mgr.Dispatch(sig, []model.Vehicle{veh})
	var lpFail, smartFB bool
	timeout := time.After(time.Second)
LOOP:
	for {
		select {
		case e := <-ch:
			if ev, ok := e.(events.StrategyEvent); ok {
				t.Logf("got event %s", ev.Action)
				if ev.Action == "lp_failure" {
					lpFail = true
				}
				if ev.Action == "smart_fallback" {
					smartFB = true
				}
			}
			if lpFail && smartFB {
				break LOOP
			}
		case <-timeout:
			break LOOP
		}
	}
	if !lpFail || !smartFB {
		t.Fatalf("expected lp failure and smart fallback events")
	}
}

func TestDispatchManager_NonStrict(t *testing.T) {
	publisher := mqtt.NewMockPublisher()
	bus := eventbus.New()

	smart := &countingSmart{SmartDispatcher: NewSmartDispatcher()}
	mgr, err := NewDispatchManager(SimpleVehicleFilter{}, smart, NoopFallback{}, publisher, time.Second, nil, bus, nil, logger.NopLogger{}, nil)
	if err != nil {
		t.Fatalf("manager: %v", err)
	}
	lp := NewLPDispatcher()
	mgr.lpDispatcher = &lp
	mgr.SetLPFirst(map[model.SignalType]bool{model.SignalFCR: true})

	sig := model.FlexibilitySignal{Type: model.SignalNEBEF, PowerKW: 10, Timestamp: time.Now()}
	veh := model.Vehicle{ID: "v1", SoC: 1, IsV2G: true, Available: true, MaxPower: 10, BatteryKWh: 50}

	mgr.Dispatch(sig, []model.Vehicle{veh})
	if smart.called == 0 {
		t.Fatalf("smart dispatcher should be used for non strict signals")
	}
}

func TestDispatchManager_LPToggleOff(t *testing.T) {
	publisher := mqtt.NewMockPublisher()
	bus := eventbus.New()

	smart := &countingSmart{SmartDispatcher: NewSmartDispatcher()}
	mgr, err := NewDispatchManager(SimpleVehicleFilter{}, smart, NoopFallback{}, publisher, time.Second, nil, bus, nil, logger.NopLogger{}, nil)
	if err != nil {
		t.Fatalf("manager: %v", err)
	}
	lp := NewLPDispatcher()
	mgr.lpDispatcher = &lp
	mgr.SetLPFirst(map[model.SignalType]bool{})

	sig := model.FlexibilitySignal{Type: model.SignalFCR, PowerKW: 10, Timestamp: time.Now()}
	veh := model.Vehicle{ID: "v1", SoC: 1, IsV2G: true, Available: true, MaxPower: 10, BatteryKWh: 50}

	mgr.Dispatch(sig, []model.Vehicle{veh})
	if smart.called == 0 {
		t.Fatalf("smart dispatcher should be used when toggle off")
	}
}
