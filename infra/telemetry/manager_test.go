package telemetry

import (
	"context"
	"testing"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/kilianp07/v2g/config"
	coremetrics "github.com/kilianp07/v2g/core/metrics"
	"github.com/kilianp07/v2g/core/model"
)

type mockRecorder struct {
	count int
	last  coremetrics.VehicleStateEvent
}

func (m *mockRecorder) RecordVehicleState(ev coremetrics.VehicleStateEvent) error {
	m.count++
	m.last = ev
	return nil
}

func TestProcess(t *testing.T) {
	rec := &mockRecorder{}
	mgr := &Manager{sink: rec}
	payload := []byte(`{"vehicle_id":"veh1","soc":0.5,"available":true,"charging":false,"power_kw":3}`)
	if err := mgr.process(payload, "", "push"); err != nil {
		t.Fatalf("process: %v", err)
	}
	if rec.count != 1 {
		t.Fatalf("expected 1 record, got %d", rec.count)
	}
	if rec.last.Vehicle.ID != "veh1" || rec.last.Vehicle.SoC != 0.5 {
		t.Fatalf("unexpected vehicle: %#v", rec.last.Vehicle)
	}
}

func TestProcessFromTopic(t *testing.T) {
	rec := &mockRecorder{}
	mgr := &Manager{sink: rec}
	topic := "v2g/vehicle/state/veh9"
	payload := []byte(`{"soc":1.5}`)
	if err := mgr.process(payload, topic, "push"); err != nil {
		t.Fatalf("process: %v", err)
	}
	if rec.last.Vehicle.ID != "veh9" {
		t.Fatalf("expected veh9, got %s", rec.last.Vehicle.ID)
	}
	if rec.last.Vehicle.SoC != 1 {
		t.Fatalf("expected SoC clamp to 1, got %v", rec.last.Vehicle.SoC)
	}
}

func TestExtractID(t *testing.T) {
	id := extractID("v2g/telemetry/response/veh42")
	if id != "veh42" {
		t.Fatalf("unexpected id %s", id)
	}
}

type fakeMessage struct {
	topic   string
	payload []byte
}

func (m *fakeMessage) Duplicate() bool   { return false }
func (m *fakeMessage) Qos() byte         { return 0 }
func (m *fakeMessage) Retained() bool    { return false }
func (m *fakeMessage) Topic() string     { return m.topic }
func (m *fakeMessage) MessageID() uint16 { return 0 }
func (m *fakeMessage) Payload() []byte   { return m.payload }
func (m *fakeMessage) Ack()              {}

func TestOnResponse(t *testing.T) {
	mgr := &Manager{respCh: make(chan telemetryMessage, 1)}
	msg := &fakeMessage{topic: "v2g/telemetry/response/veh7", payload: []byte("hi")}
	mgr.onResponse(nil, msg)
	select {
	case m := <-mgr.respCh:
		if m.VehicleID != "veh7" || string(m.Payload) != "hi" {
			t.Fatalf("unexpected message %#v", m)
		}
	default:
		t.Fatal("no message received")
	}
}

func TestOnPush(t *testing.T) {
	rec := &mockRecorder{}
	mgr := &Manager{sink: rec}
	msg := &fakeMessage{topic: "v2g/vehicle/state/veh1", payload: []byte(`{"vehicle_id":"veh1"}`)}
	mgr.onPush(nil, msg)
	if rec.count != 1 {
		t.Fatalf("expected 1 record, got %d", rec.count)
	}
}

type stubToken struct{}

func (stubToken) Wait() bool                     { return true }
func (stubToken) WaitTimeout(time.Duration) bool { return true }
func (stubToken) Done() <-chan struct{}          { ch := make(chan struct{}); close(ch); return ch }
func (stubToken) Error() error                   { return nil }

type mockClient struct{ publishCount int }

func (m *mockClient) IsConnected() bool       { return true }
func (m *mockClient) IsConnectionOpen() bool  { return true }
func (m *mockClient) Connect() paho.Token     { return stubToken{} }
func (m *mockClient) Disconnect(quiesce uint) {}
func (m *mockClient) Publish(topic string, qos byte, retained bool, payload interface{}) paho.Token {
	m.publishCount++
	return stubToken{}
}
func (m *mockClient) Subscribe(topic string, qos byte, callback paho.MessageHandler) paho.Token {
	return stubToken{}
}
func (m *mockClient) SubscribeMultiple(map[string]byte, paho.MessageHandler) paho.Token {
	return stubToken{}
}
func (m *mockClient) Unsubscribe(...string) paho.Token        { return stubToken{} }
func (m *mockClient) AddRoute(string, paho.MessageHandler)    {}
func (m *mockClient) OptionsReader() paho.ClientOptionsReader { return paho.ClientOptionsReader{} }

type mockDiscovery struct{ vehicles []model.Vehicle }

func (m mockDiscovery) Discover(ctx context.Context, timeout time.Duration) ([]model.Vehicle, error) {
	return m.vehicles, nil
}
func (m mockDiscovery) Close() error { return nil }

func TestDoPoll(t *testing.T) {
	rec := &mockRecorder{}
	mc := &mockClient{}
	mgr := &Manager{
		cfg:         config.TelemetryConfig{RequestTopic: "req", TimeoutSeconds: 1},
		cli:         mc,
		sink:        rec,
		respCh:      make(chan telemetryMessage, 1),
		pollReq:     prometheus.NewCounter(prometheus.CounterOpts{Name: "test_poll_requests_total"}),
		pollResp:    prometheus.NewCounter(prometheus.CounterOpts{Name: "test_poll_responses_total"}),
		pollTimeout: prometheus.NewCounter(prometheus.CounterOpts{Name: "test_poll_timeout_total"}),
		lastCollect: prometheus.NewGauge(prometheus.GaugeOpts{Name: "test_last_collect"}),
		latency:     prometheus.NewHistogram(prometheus.HistogramOpts{Name: "test_latency"}),
		disc:        mockDiscovery{vehicles: []model.Vehicle{{ID: "veh1"}, {ID: "veh2"}}},
	}
	mgr.respCh <- telemetryMessage{VehicleID: "veh1", Payload: []byte(`{"vehicle_id":"veh1"}`), Arrived: time.Now()}
	mgr.doPoll(context.Background())
	if mc.publishCount != 1 {
		t.Fatalf("expected publish 1, got %d", mc.publishCount)
	}
	if v := testutil.ToFloat64(mgr.pollReq); v != 1 {
		t.Fatalf("expected pollReq 1, got %v", v)
	}
	if v := testutil.ToFloat64(mgr.pollResp); v != 1 {
		t.Fatalf("expected pollResp 1, got %v", v)
	}
	if v := testutil.ToFloat64(mgr.pollTimeout); v != 1 {
		t.Fatalf("expected pollTimeout 1, got %v", v)
	}
}
