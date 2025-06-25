package test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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
	"github.com/kilianp07/v2g/metrics"
	"github.com/kilianp07/v2g/model"
	"github.com/kilianp07/v2g/mqtt"
	"github.com/kilianp07/v2g/rte"
)

type managerWrapper struct {
	mgr      *dispatch.DispatchManager
	vehicles []model.Vehicle
}

func waitForHTTPServer(s *rte.RTEServerMock, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		url := "http://" + s.Addr() + "/rte/ping"
		resp, err := http.Get(url)
		if err == nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			if err := resp.Body.Close(); err != nil {
				return err
			}
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	return fmt.Errorf("server not ready: %s", s.Addr())
}

func (m managerWrapper) Dispatch(sig model.FlexibilitySignal, _ []model.Vehicle) dispatch.DispatchResult {
	return m.mgr.Dispatch(sig, m.vehicles)
}

func TestRTEDispatchEndToEnd(t *testing.T) {
	reg := prometheus.NewRegistry()

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
	srv := rte.NewRTEServerMockWithRegistry(config.RTEMockConfig{Address: "127.0.0.1:0"}, wrapper, reg)
	go func() { _ = srv.Start(ctx) }()
	if err := waitForHTTPServer(srv, 2*time.Second); err != nil {
		cancel()
		t.Fatalf("server not ready: %v", err)
	}
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
	if err := metricsResp.Body.Close(); err != nil {
		t.Fatalf("close body: %v", err)
	}
	out := string(body)
	expectedSignal := `rte_signals_total{signal_type="FCR"} 1`
	if !strings.Contains(out, expectedSignal) {
		t.Errorf("signal metric missing: %s", expectedSignal)
	}
	expectedDispatch := `dispatch_events_total{acknowledged="true",signal_type="FCR",vehicle_id="veh1"} 1`
	if !strings.Contains(out, expectedDispatch) {
		t.Errorf("dispatch metric missing: %s", expectedDispatch)
	}
}
