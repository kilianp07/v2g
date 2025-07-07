package scenarios

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/kilianp07/v2g/core/dispatch"
	coremetrics "github.com/kilianp07/v2g/core/metrics"
	"github.com/kilianp07/v2g/core/model"
	"github.com/kilianp07/v2g/infra/logger"
	"github.com/kilianp07/v2g/infra/metrics"
	"github.com/kilianp07/v2g/infra/mqtt"
	"github.com/kilianp07/v2g/internal/eventbus"
)

func RunScenario(t *testing.T, sc *Scenario) {
	reg := prometheus.NewRegistry()
	sinkIf, err := metrics.NewPromSinkWithRegistry(coremetrics.Config{}, reg)
	if err != nil {
		t.Fatalf("prom sink: %v", err)
	}
	sink, ok := sinkIf.(*metrics.PromSink)
	if !ok {
		t.Fatalf("expected *metrics.PromSink, got %T", sinkIf)
	}

	pub := mqtt.NewMockPublisher()
	for _, id := range sc.FailVehicles {
		pub.FailIDs[id] = true
	}

	bus := eventbus.New()

	mgr, err := dispatch.NewDispatchManager(
		dispatch.SimpleVehicleFilter{},
		&dispatch.SmartDispatcher{},
		dispatch.NewBalancedFallback(logger.NopLogger{}),
		pub,
		10*time.Millisecond,
		sink,
		bus,
		nil,
		logger.NopLogger{},
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("manager: %v", err)
	}

	vehicles := make([]model.Vehicle, len(sc.Vehicles))
	for i, v := range sc.Vehicles {
		vehicles[i] = v.ToModel()
	}

	ackCount := 0
	activeVehicles := vehicles
	ts := time.Unix(0, 0)

	for i, sigDef := range sc.Signals {
		for vid, after := range sc.AckFailAfter {
			if i >= after {
				pub.FailIDs[vid] = true
			}
		}
		for vid, after := range sc.DisconnectAfter {
			if i >= after {
				activeVehicles = removeVehicle(activeVehicles, vid)
			}
		}
		res := mgr.Dispatch(sigDef.ToModel(ts), activeVehicles)
		for _, ok := range res.Acknowledged {
			if ok {
				ackCount++
			}
		}
	}

	if ackCount != sc.Expected.Acked {
		t.Errorf("scenario %s expected %d acked, got %d", sc.Name, sc.Expected.Acked, ackCount)
	}
}

func removeVehicle(vehicles []model.Vehicle, id string) []model.Vehicle {
	out := vehicles[:0]
	for _, v := range vehicles {
		if v.ID != id {
			out = append(out, v)
		}
	}
	return out
}
