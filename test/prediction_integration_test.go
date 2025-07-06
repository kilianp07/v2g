package test

import (
	"testing"
	"time"

	"github.com/kilianp07/v2g/core/dispatch"
	"github.com/kilianp07/v2g/core/model"
	"github.com/kilianp07/v2g/core/prediction"
	"github.com/kilianp07/v2g/infra/logger"
	"github.com/kilianp07/v2g/infra/mqtt"
	"github.com/kilianp07/v2g/internal/eventbus"
)

type countingEngine struct {
	prediction.MockPredictionEngine
	calls int
}

func (c *countingEngine) PredictAvailability(id string, t time.Time) float64 {
	c.calls++
	return c.MockPredictionEngine.PredictAvailability(id, t)
}

func (c *countingEngine) ForecastSoC(id string, h time.Duration) []float64 {
	c.calls++
	return c.MockPredictionEngine.ForecastSoC(id, h)
}

func TestPredictionIntegration(t *testing.T) {
	publisher := mqtt.NewMockPublisher()
	bus := eventbus.New()
	eng := &countingEngine{MockPredictionEngine: prediction.MockPredictionEngine{Availability: map[string]float64{"v1": 0.2, "v2": 1}}}
	smart := dispatch.NewSmartDispatcher()
	mgr, err := dispatch.NewDispatchManager(
		dispatch.SimpleVehicleFilter{},
		&smart,
		dispatch.NoopFallback{},
		publisher,
		time.Second,
		nil,
		bus,
		nil,
		logger.NopLogger{},
		nil,
		eng,
	)
	if err != nil {
		t.Fatalf("manager: %v", err)
	}
	vehicles := []model.Vehicle{{ID: "v1", SoC: 1, IsV2G: true, Available: true, MaxPower: 10, BatteryKWh: 50}, {ID: "v2", SoC: 1, IsV2G: true, Available: true, MaxPower: 10, BatteryKWh: 50}}
	sig := model.FlexibilitySignal{Type: model.SignalFCR, PowerKW: 10, Duration: time.Hour, Timestamp: time.Now()}
	_ = mgr.Dispatch(sig, vehicles)
	if eng.calls == 0 {
		t.Fatalf("prediction engine was not called")
	}
}
