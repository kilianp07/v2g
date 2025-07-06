package dispatch

import (
	"testing"
	"time"

	"github.com/kilianp07/v2g/core/model"
	"github.com/kilianp07/v2g/core/prediction"
	"github.com/kilianp07/v2g/infra/logger"
	imqtt "github.com/kilianp07/v2g/infra/mqtt"
	"github.com/kilianp07/v2g/internal/eventbus"
)

// helper to create a manager with optional prediction engine
func newTestManager(pred prediction.PredictionEngine) *DispatchManager {
	pub := imqtt.NewMockPublisher()
	bus := eventbus.New()
	disp := NewSmartDispatcher()
	mgr, err := NewDispatchManager(SimpleVehicleFilter{}, &disp, NoopFallback{}, pub, time.Second, nil, bus, nil, logger.NopLogger{}, nil, pred)
	if err != nil {
		panic(err)
	}
	return mgr
}

func TestDispatchManager_UsesPrediction(t *testing.T) {
	eng := &prediction.MockPredictionEngine{
		Availability: map[string]float64{"v1": 1, "v2": 0.1},
	}
	mgr := newTestManager(eng)

	vehicles := []model.Vehicle{
		{ID: "v1", SoC: 1, IsV2G: true, Available: true, MaxPower: 10, BatteryKWh: 50},
		{ID: "v2", SoC: 1, IsV2G: true, Available: true, MaxPower: 10, BatteryKWh: 50},
	}
	sig := model.FlexibilitySignal{Type: model.SignalFCR, PowerKW: 10, Duration: time.Hour, Timestamp: time.Now()}

	res := mgr.Dispatch(sig, vehicles)
	if res.Assignments["v1"] <= res.Assignments["v2"] {
		t.Fatalf("expected v1 prioritized")
	}
}

func TestDispatchManager_NoPrediction(t *testing.T) {
	mgr := newTestManager(nil)
	vehicles := []model.Vehicle{
		{ID: "v1", SoC: 1, IsV2G: true, Available: true, MaxPower: 10, BatteryKWh: 50},
		{ID: "v2", SoC: 1, IsV2G: true, Available: true, MaxPower: 10, BatteryKWh: 50},
	}
	sig := model.FlexibilitySignal{Type: model.SignalFCR, PowerKW: 10, Duration: time.Hour, Timestamp: time.Now()}

	res := mgr.Dispatch(sig, vehicles)
	total := res.Assignments["v1"] + res.Assignments["v2"]
	if int(total+0.5) != 10 {
		t.Fatalf("unexpected total %v", total)
	}
}
