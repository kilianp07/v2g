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

	"github.com/kilianp07/v2g/model"
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
	rec := DispatchResult{
		Signal:       model.FlexibilitySignal{Type: model.SignalFCR, PowerKW: 10, Timestamp: now},
		VehicleID:    "veh1",
		PowerKW:      5,
		Score:        1.2,
		MarketPrice:  50,
		Acknowledged: true,
		DispatchTime: now,
	}

	if err := sink.RecordDispatchResult([]DispatchResult{rec}); err != nil {
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

	cfg := Config{
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
