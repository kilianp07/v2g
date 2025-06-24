package test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/kilianp07/v2g/config"
	"github.com/kilianp07/v2g/dispatch"
	"github.com/kilianp07/v2g/logger"
	"github.com/kilianp07/v2g/metrics"
	"github.com/kilianp07/v2g/model"
	"github.com/kilianp07/v2g/mqtt"
	"github.com/kilianp07/v2g/rte"
)

type managerWrapper struct {
	mgr      *dispatch.DispatchManager
	vehicles []model.Vehicle
}

func (m managerWrapper) Dispatch(sig model.FlexibilitySignal, _ []model.Vehicle) dispatch.DispatchResult {
	return m.mgr.Dispatch(sig, m.vehicles)
}

func TestRTEDispatchEndToEnd(t *testing.T) {
	reg := prometheus.NewRegistry()
	prometheus.DefaultRegisterer = reg
	prometheus.DefaultGatherer = reg

	sinkIf, err := metrics.NewPromSinkWithRegistry(metrics.Config{}, reg)
	if err != nil {
		t.Fatalf("prom sink: %v", err)
	}
	sink := sinkIf.(*metrics.PromSink)

	publisher := mqtt.NewMockPublisher()
	mgr, err := dispatch.NewDispatchManager(
		dispatch.SimpleVehicleFilter{},
		dispatch.EqualDispatcher{},
		dispatch.NoopFallback{},
		publisher,
		time.Second,
		logger.NopLogger{},
		sink,
	)
	if err != nil {
		t.Fatalf("manager: %v", err)
	}

	vehicles := []model.Vehicle{
		{ID: "veh1", SoC: 0.8, IsV2G: true, Available: true, MaxPower: 40, BatteryKWh: 50},
	}

	wrapper := managerWrapper{mgr: mgr, vehicles: vehicles}
	ctx, cancel := context.WithCancel(context.Background())
	srv := rte.NewRTEServerMock(config.RTEMockConfig{Address: "127.0.0.1:0"}, wrapper, nil)
	go func() { _ = srv.Start(ctx) }()
	time.Sleep(100 * time.Millisecond)
	defer cancel()

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	metricsTS := httptest.NewServer(mux)
	defer metricsTS.Close()

	sig := rte.Signal{
		SignalType: "FCR",
		StartTime:  time.Now(),
		EndTime:    time.Now().Add(5 * time.Minute),
		Power:      20,
	}
	data, _ := json.Marshal(sig)
	resp, err := http.Post("http://"+srv.Addr()+"/rte/signal", "application/json", bytes.NewReader(data))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: %d", resp.StatusCode)
	}

	if publisher.Messages["veh1"] != 20 {
		t.Fatalf("order not published")
	}

	metricsResp, err := http.Get(metricsTS.URL + "/metrics")
	if err != nil {
		t.Fatalf("metrics: %v", err)
	}
	body, _ := io.ReadAll(metricsResp.Body)
	log.Printf("metrics output:\n%s", string(body))
	metricsResp.Body.Close()
	out := string(body)
	if !strings.Contains(out, "rte_signals_total") {
		t.Errorf("rte_signals_total missing")
	}
	expectedDispatch := `dispatch_events_total{acknowledged="true",signal_type="FCR",vehicle_id="veh1"} 1`
	if !strings.Contains(out, expectedDispatch) {
		t.Errorf("dispatch metric missing: %s", expectedDispatch)
	}
}
