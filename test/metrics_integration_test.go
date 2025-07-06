package test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/kilianp07/v2g/core/dispatch"
	"github.com/kilianp07/v2g/core/model"
	"github.com/kilianp07/v2g/infra/logger"
	"github.com/kilianp07/v2g/infra/mqtt"
)

func TestMetricsHTTPExposure(t *testing.T) {
	dispatch.ResetMetrics(nil)
	reg := prometheus.NewRegistry()
	dispatch.MustRegisterMetrics(reg)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	srv := httptest.NewServer(mux)
	defer srv.Close()

	publisher := mqtt.NewMockPublisher()
	mgr, err := dispatch.NewDispatchManager(dispatch.SimpleVehicleFilter{}, dispatch.EqualDispatcher{}, dispatch.NoopFallback{}, publisher, time.Second, nil, nil, nil, logger.NopLogger{}, nil, nil)
	if err != nil {
		t.Fatalf("manager: %v", err)
	}
	vehicles := []model.Vehicle{{ID: "v1", SoC: 0.8, IsV2G: true, Available: true, MaxPower: 10, BatteryKWh: 40}}
	sig := model.FlexibilitySignal{Type: model.SignalFCR, PowerKW: 5, Duration: time.Second, Timestamp: time.Now()}
	mgr.Dispatch(sig, vehicles)

	resp, err := http.Get(srv.URL + "/metrics")
	if err != nil {
		t.Fatalf("get metrics: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	out := string(body)
	if !strings.Contains(out, "vehicles_dispatched_total") {
		t.Errorf("metrics output missing counter: %s", out)
	}
}
