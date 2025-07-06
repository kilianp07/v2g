package test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	vehiclesapi "github.com/kilianp07/v2g/api/vehicles"
	"github.com/kilianp07/v2g/core/dispatch"
	"github.com/kilianp07/v2g/core/model"
	"github.com/kilianp07/v2g/core/prediction"
	vehiclestatus "github.com/kilianp07/v2g/core/vehiclestatus"
	"github.com/kilianp07/v2g/infra/logger"
	"github.com/kilianp07/v2g/infra/mqtt"
)

func TestVehicleStatusIntegration(t *testing.T) {
	store := vehiclestatus.NewMemoryStore()
	pred := &prediction.MockPredictionEngine{SoCForecasts: map[string][]float64{"v1": {0.8, 0.7}}}
	publisher := mqtt.NewMockPublisher()
	mgr, err := dispatch.NewDispatchManager(dispatch.SimpleVehicleFilter{}, dispatch.EqualDispatcher{}, dispatch.NoopFallback{}, publisher, time.Second, nil, nil, nil, logger.NopLogger{}, nil, pred)
	if err != nil {
		t.Fatalf("manager: %v", err)
	}
	mgr.SetStatusStore(store)
	store.Set(vehiclestatus.Status{VehicleID: "v1", FleetID: "f1"})
	v := []model.Vehicle{{ID: "v1", IsV2G: true, Available: true, MaxPower: 10, SoC: 0.9}}
	sig := model.FlexibilitySignal{Type: model.SignalFCR, PowerKW: 5, Duration: time.Minute, Timestamp: time.Now()}
	mgr.Dispatch(sig, v)

	h := vehiclesapi.NewStatusHandler(store, pred)
	srv := httptest.NewServer(h)
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			t.Fatalf("close body: %v", cerr)
		}
	}()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	var out []vehiclestatus.Status
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(out) != 1 || out[0].LastDispatchDecision.SignalType != "FCR" {
		t.Fatalf("unexpected response %#v", out)
	}
}
