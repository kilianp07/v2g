package dispatch

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/kilianp07/v2g/core/events"
	"github.com/kilianp07/v2g/core/model"
	"github.com/kilianp07/v2g/infra/logger"
	"github.com/kilianp07/v2g/infra/mqtt"
	"github.com/kilianp07/v2g/internal/eventbus"
)

func TestDispatchMetricsUpdate(t *testing.T) {
	ResetMetrics(nil)
	t.Cleanup(func() { ResetMetrics(nil) })
	reg := prometheus.NewRegistry()
	MustRegisterMetrics(reg)

	publisher := mqtt.NewMockPublisher()
	mgr, err := NewDispatchManager(SimpleVehicleFilter{}, EqualDispatcher{}, NoopFallback{}, publisher, time.Second, nil, nil, nil, logger.NopLogger{}, nil, nil)
	if err != nil {
		t.Fatalf("manager: %v", err)
	}

	v := []model.Vehicle{{ID: "v1", SoC: 0.8, IsV2G: true, Available: true, MaxPower: 10, BatteryKWh: 40}}
	sig := model.FlexibilitySignal{Type: model.SignalFCR, PowerKW: 5, Duration: time.Second, Timestamp: time.Now()}
	mgr.Dispatch(sig, v)
	time.Sleep(10 * time.Millisecond)

	metric := testutil.ToFloat64(vehiclesDispatched.WithLabelValues("FCR"))
	if metric != 1 {
		t.Errorf("vehiclesDispatched expected 1 got %f", metric)
	}
	if val := testutil.ToFloat64(mqttSuccess); val != 1 {
		t.Errorf("mqttSuccess expected 1 got %f", val)
	}
	if val := testutil.ToFloat64(ackRate.WithLabelValues("FCR")); val != 1 {
		t.Errorf("ackRate expected 1 got %f", val)
	}
	if count := testutil.CollectAndCount(dispatchLatency); count == 0 {
		t.Errorf("dispatchLatency not updated")
	}
}

func TestAckRateCalculation(t *testing.T) {
	ResetMetrics(nil)
	t.Cleanup(func() { ResetMetrics(nil) })
	reg := prometheus.NewRegistry()
	MustRegisterMetrics(reg)

	publisher := mqtt.NewMockPublisher()
	publisher.FailIDs["v2"] = true
	mgr, err := NewDispatchManager(SimpleVehicleFilter{}, EqualDispatcher{}, NoopFallback{}, publisher, time.Millisecond*10, nil, nil, nil, logger.NopLogger{}, nil, nil)
	if err != nil {
		t.Fatalf("manager: %v", err)
	}
	vehicles := []model.Vehicle{{ID: "v2", SoC: 0.8, IsV2G: true, Available: true, MaxPower: 10, BatteryKWh: 40}}
	sig := model.FlexibilitySignal{Type: model.SignalFCR, PowerKW: 5, Duration: time.Second, Timestamp: time.Now()}
	mgr.Dispatch(sig, vehicles)
	time.Sleep(10 * time.Millisecond)

	if val := testutil.ToFloat64(ackRate.WithLabelValues("FCR")); val != 0 {
		t.Errorf("ackRate expected 0 got %f", val)
	}
	if val := testutil.ToFloat64(mqttFailure); val != 1 {
		t.Errorf("mqttFailure expected 1 got %f", val)
	}
}

func TestContextParticipation(t *testing.T) {
	var ctx DispatchContext
	ctx.SetParticipation("v1", 0.7)
	if v := ctx.GetParticipation("v1"); v != 0.7 {
		t.Errorf("unexpected participation %f", v)
	}
}

func TestNoopTuner(t *testing.T) {
	var tuner NoopTuner
	// should not panic or modify input
	tuner.Tune([]DispatchResult{})
}

type dummyDiscovery struct{ closed bool }

func (d *dummyDiscovery) Discover(_ context.Context, _ time.Duration) ([]model.Vehicle, error) {
	return nil, nil
}

func (d *dummyDiscovery) Close() error { d.closed = true; return nil }

func TestManagerRunAndClose(t *testing.T) {
	ResetMetrics(nil)
	t.Cleanup(func() { ResetMetrics(nil) })
	pub := mqtt.NewMockPublisher()
	bus := eventbus.New()
	disc := &dummyDiscovery{}
	mgr, err := NewDispatchManager(SimpleVehicleFilter{}, EqualDispatcher{}, NoopFallback{}, pub, time.Millisecond*10, nil, bus, disc, logger.NopLogger{}, nil, nil)
	if err != nil {
		t.Fatalf("manager: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	sigCh := make(chan model.FlexibilitySignal, 1)
	sub := bus.Subscribe()
	t.Cleanup(func() { bus.Unsubscribe(sub); bus.Close() })
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for ev := range sub {
			if _, ok := ev.(events.SignalEvent); ok {
				return
			}
		}
	}()
	go mgr.Run(ctx, sigCh)
	sigCh <- model.FlexibilitySignal{Type: model.SignalFCR, Timestamp: time.Now()}
	wg.Wait()
	cancel()
	if err := mgr.Close(); err != nil {
		t.Fatalf("close error: %v", err)
	}
	if !disc.closed {
		t.Errorf("discovery not closed")
	}
}
