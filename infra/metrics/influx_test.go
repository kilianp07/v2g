package metrics

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/influxdata/influxdb-client-go/v2/api/write"

	coremetrics "github.com/kilianp07/v2g/core/metrics"
	"github.com/kilianp07/v2g/core/model"
)

func TestInfluxSink_RecordDispatchResult(t *testing.T) {
	var body string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		body = string(data)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	sink := NewInfluxSink(srv.URL, "token", "org", "bucket")
	now := time.Now()
	rec := coremetrics.DispatchResult{
		Signal:       model.FlexibilitySignal{Type: model.SignalFCR, PowerKW: 10, Timestamp: now},
		StartTime:    now,
		EndTime:      now.Add(time.Hour),
		VehicleID:    "veh1",
		PowerKW:      5,
		Score:        1.2,
		MarketPrice:  50,
		Acknowledged: true,
		DispatchTime: now,
	}

	if err := sink.RecordDispatchResult([]coremetrics.DispatchResult{rec}); err != nil {
		t.Fatalf("record error: %v", err)
	}
	p := write.NewPointWithMeasurement("dispatch_event").
		AddTag("vehicle_id", "veh1").
		AddTag("signal_type", "FCR").
		AddTag("acknowledged", "true").
		AddTag("dispatch_id", strconv.FormatInt(now.UnixNano(), 10)).
		AddTag("component", "dispatch_manager").
		AddField("power_kw", 5.0).
		AddField("score", 1.2).
		AddField("market_price", 50.0).
		SetTime(now)
	expected := strings.TrimSpace(write.PointToLineProtocol(p, time.Nanosecond))
	if strings.TrimSpace(body) != expected {
		t.Errorf("unexpected body: %s", body)
	}
}

func TestNewInfluxSinkWithFallback(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			called = true
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}))
	defer srv.Close()

	cfg := coremetrics.Config{
		InfluxURL:    srv.URL + "/api/v2/write",
		InfluxToken:  "tok",
		InfluxOrg:    "org",
		InfluxBucket: "bucket",
	}
	sink := NewInfluxSinkWithFallback(cfg)
	if _, ok := sink.(*InfluxSink); ok {
		t.Fatalf("expected NopSink on failing health check")
	}
	if !called {
		t.Fatalf("health endpoint not called")
	}
}

func TestInfluxSink_RecordVehicleState(t *testing.T) {
	var bodies []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		bodies = append(bodies, strings.TrimSpace(string(data)))
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	sink := NewInfluxSink(srv.URL, "token", "org", "bucket")
	now := time.Now()
	ev := coremetrics.VehicleStateEvent{
		Vehicle: model.Vehicle{ID: "v1", SoC: 0.5},
		FleetID: "f1",
		Context: "test",
		Time:    now,
	}
	if err := sink.RecordVehicleState(ev); err != nil {
		t.Fatalf("record error: %v", err)
	}
	p := write.NewPointWithMeasurement("vehicle_state").
		AddTag("vehicle_id", "v1").
		AddTag("fleet_id", "f1").
		AddTag("context", "test").
		AddField("soc", 0.5).
		AddField("available", false).
		AddField("charging", false).
		AddField("power_kw", 0.0).
		SetTime(now)
	p2 := write.NewPointWithMeasurement("vehicle_soc_percent").
		AddTag("vehicle_id", "v1").
		AddTag("fleet_id", "f1").
		AddField("soc", 50.0).
		SetTime(now)
	exp1 := strings.TrimSpace(write.PointToLineProtocol(p, time.Nanosecond))
	exp2 := strings.TrimSpace(write.PointToLineProtocol(p2, time.Nanosecond))
	if len(bodies) != 2 || bodies[0] != exp1 || bodies[1] != exp2 {
		t.Errorf("unexpected bodies: %#v", bodies)
	}
}

func TestInfluxSink_RecordDispatchOrder(t *testing.T) {
	var bodies []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		bodies = append(bodies, strings.TrimSpace(string(b)))
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	sink := NewInfluxSink(srv.URL, "token", "org", "bucket")
	now := time.Now()
	ev := coremetrics.DispatchOrderEvent{
		DispatchID: "d1",
		VehicleID:  "v1",
		Signal:     model.SignalFCR,
		PowerKW:    -5,
		Time:       now,
	}
	if err := sink.RecordDispatchOrder(ev); err != nil {
		t.Fatalf("record: %v", err)
	}
	p := write.NewPointWithMeasurement("dispatch_order_sent").
		AddTag("vehicle_id", "v1").
		AddTag("signal_type", "FCR").
		AddTag("dispatch_id", "d1").
		AddTag("component", "dispatch_manager").
		AddField("power_kw", -5.0).
		AddField("score", 0.0).
		AddField("market_price", 0.0).
		SetTime(now)
	p2 := write.NewPointWithMeasurement("dispatch_order_kw").
		AddTag("vehicle_id", "v1").
		AddTag("signal_type", "FCR").
		AddTag("mode", "charge").
		AddField("power_kw", -5.0).
		SetTime(now)
	exp1 := strings.TrimSpace(write.PointToLineProtocol(p, time.Nanosecond))
	exp2 := strings.TrimSpace(write.PointToLineProtocol(p2, time.Nanosecond))
	if len(bodies) != 2 || bodies[0] != exp1 || bodies[1] != exp2 {
		t.Errorf("bodies: %#v", bodies)
	}
}

func TestInfluxSink_RecordDispatchAck(t *testing.T) {
	var bodies []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		bodies = append(bodies, strings.TrimSpace(string(b)))
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	sink := NewInfluxSink(srv.URL, "token", "org", "bucket")
	now := time.Now()
	ev := coremetrics.DispatchAckEvent{
		DispatchID:   "d1",
		VehicleID:    "v1",
		Signal:       model.SignalFCR,
		Acknowledged: true,
		Latency:      time.Second,
		Time:         now,
	}
	if err := sink.RecordDispatchAck(ev); err != nil {
		t.Fatalf("record: %v", err)
	}
	p := write.NewPointWithMeasurement("dispatch_ack_received").
		AddTag("vehicle_id", "v1").
		AddTag("signal_type", "FCR").
		AddTag("acknowledged", "true").
		AddTag("dispatch_id", "d1").
		AddTag("component", "dispatch_manager").
		AddField("latency_ms", 1000.0).
		AddField("errors", "").
		SetTime(now)
	p2 := write.NewPointWithMeasurement("acknowledgment").
		AddTag("vehicle_id", "v1").
		AddTag("signal_type", "FCR").
		AddField("acknowledged", true).
		AddField("latency_ms", 1000.0).
		SetTime(now)
	exp1 := strings.TrimSpace(write.PointToLineProtocol(p, time.Nanosecond))
	exp2 := strings.TrimSpace(write.PointToLineProtocol(p2, time.Nanosecond))
	if len(bodies) != 2 || bodies[0] != exp1 || bodies[1] != exp2 {
		t.Errorf("bodies: %#v", bodies)
	}
}

func TestInfluxSink_RecordRTESignal(t *testing.T) {
	var bodies []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		bodies = append(bodies, strings.TrimSpace(string(b)))
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	sink := NewInfluxSink(srv.URL, "token", "org", "bucket")
	now := time.Now()
	ev := coremetrics.RTESignalEvent{Signal: model.FlexibilitySignal{Type: model.SignalFCR, PowerKW: 10}, Time: now}
	if err := sink.RecordRTESignal(ev); err != nil {
		t.Fatalf("record: %v", err)
	}
	p := write.NewPointWithMeasurement("rte_signal_received").
		AddTag("signal_type", "FCR").
		AddTag("component", "rte_connector").
		AddField("power_kw", 10.0).
		SetTime(now)
	p2 := write.NewPointWithMeasurement("signal_metadata").
		AddTag("signal_type", "FCR").
		AddField("power_kw", 10.0).
		SetTime(now)
	exp1 := strings.TrimSpace(write.PointToLineProtocol(p, time.Nanosecond))
	exp2 := strings.TrimSpace(write.PointToLineProtocol(p2, time.Nanosecond))
	if len(bodies) != 2 || bodies[0] != exp1 || bodies[1] != exp2 {
		t.Errorf("bodies: %#v", bodies)
	}
}

func TestInfluxSink_RecordVehicleAvailability(t *testing.T) {
	var bodies []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		bodies = append(bodies, strings.TrimSpace(string(b)))
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	sink := NewInfluxSink(srv.URL, "token", "org", "bucket")
	now := time.Now()
	av := []coremetrics.VehicleAvailability{{VehicleID: "v1", Probability: 0.7, Time: now}}
	if err := sink.RecordVehicleAvailability(av); err != nil {
		t.Fatalf("record: %v", err)
	}
	p := write.NewPointWithMeasurement("vehicle_availability").
		AddTag("vehicle_id", "v1").
		AddField("probability", 0.7).
		SetTime(now)
	exp := strings.TrimSpace(write.PointToLineProtocol(p, time.Nanosecond))
	if len(bodies) != 1 || bodies[0] != exp {
		t.Errorf("bodies: %#v", bodies)
	}
}
