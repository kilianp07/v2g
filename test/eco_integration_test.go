package test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	apiveh "github.com/kilianp07/v2g/api/vehicles"
	coremetrics "github.com/kilianp07/v2g/core/metrics"
	eco "github.com/kilianp07/v2g/core/metrics/eco"
	"github.com/kilianp07/v2g/core/model"
	infmetrics "github.com/kilianp07/v2g/infra/metrics"
)

func TestEcoIntegration(t *testing.T) {
	store := eco.NewMemoryStore()
	reg := prometheus.NewRegistry()
	sink := infmetrics.NewEcoSink(store, 10, reg)

	flex := newFlex()
	if err := sink.RecordDispatchResult([]coremetrics.DispatchResult{{Signal: flex, VehicleID: "v1", PowerKW: 5}}); err != nil {
		t.Fatalf("record: %v", err)
	}

	// check prom metrics
	expected := "# HELP vehicle_injected_energy_kwh Daily injected energy per vehicle\n# TYPE vehicle_injected_energy_kwh gauge\nvehicle_injected_energy_kwh{day=\"" + eco.Day(time.Now()).Format("2006-01-02") + "\",vehicle_id=\"v1\"} 5\n"
	if err := testutil.GatherAndCompare(reg, strings.NewReader(expected), "vehicle_injected_energy_kwh"); err != nil {
		t.Fatalf("prom: %v", err)
	}

	h := apiveh.NewKPIHandler(store, 10)
	req := httptest.NewRequest("GET", "/api/vehicles/v1/kpis", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d", rr.Code)
	}
	var out []map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("json: %v", err)
	}
	if len(out) == 0 || out[0]["injected_kwh"].(float64) != 5 {
		t.Fatalf("bad json %+v", out)
	}
}

func newFlex() model.FlexibilitySignal {
	return model.FlexibilitySignal{Type: model.SignalFCR, PowerKW: 5, Duration: time.Hour, Timestamp: time.Now()}
}
