package metrics

import (
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/kilianp07/v2g/model"
)

func TestPromSink_RecordDispatchResult(t *testing.T) {
	reg := prometheus.NewRegistry()
	sinkIf, err := NewPromSinkWithRegistry(Config{}, reg)
	if err != nil {
		t.Fatalf("create sink: %v", err)
	}
	sink, ok := sinkIf.(*PromSink)
	if !ok {
		t.Fatalf("expected PromSink")
	}
	now := time.Now()
	rec := DispatchResult{
		Signal:       model.FlexibilitySignal{Type: model.SignalFCR, PowerKW: 10, Timestamp: now},
		VehicleID:    "veh1",
		PowerKW:      5,
		Acknowledged: true,
		DispatchTime: now,
	}
	if err := sink.RecordDispatchResult([]DispatchResult{rec}); err != nil {
		t.Fatalf("record error: %v", err)
	}
	if err := sink.RecordDispatchLatency([]DispatchLatency{{
		VehicleID:    "veh1",
		Signal:       model.SignalFCR,
		Acknowledged: true,
		Latency:      150 * time.Millisecond,
	}}); err != nil {
		t.Fatalf("latency error: %v", err)
	}

	expected := `
# HELP dispatch_events_total Total number of dispatch events
# TYPE dispatch_events_total counter
dispatch_events_total{acknowledged="true",signal_type="FCR",vehicle_id="veh1"} 1
`
	if err := testutil.CollectAndCompare(sink.events, strings.NewReader(expected)); err != nil {
		t.Errorf("unexpected metrics: %v", err)
	}

	if c := testutil.CollectAndCount(sink.latency); c == 0 {
		t.Errorf("latency not recorded")
	}

	// record fleet size and verify gauge value
	if err := sink.RecordFleetSize(42); err != nil {
		t.Fatalf("fleet size error: %v", err)
	}
	expectedFleet := `
# HELP fleet_discovery_vehicles_total Number of vehicles discovered during fleet discovery
# TYPE fleet_discovery_vehicles_total gauge
fleet_discovery_vehicles_total 42
`
	if err := testutil.CollectAndCompare(sink.fleet, strings.NewReader(expectedFleet)); err != nil {
		t.Errorf("unexpected fleet metric: %v", err)
	}
}
