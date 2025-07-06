package test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	dispatchapi "github.com/kilianp07/v2g/api/dispatch"
	"github.com/kilianp07/v2g/core/dispatch"
	"github.com/kilianp07/v2g/core/dispatch/logging"
	"github.com/kilianp07/v2g/core/model"
	"github.com/kilianp07/v2g/infra/logger"
	"github.com/kilianp07/v2g/infra/mqtt"
)

func TestDispatchLoggingIntegration(t *testing.T) {
	store, err := logging.NewSQLiteStore("file:testlog.db?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	defer func() { _ = store.Close() }()
	dispatch.ResetMetrics(nil)

	publisher := mqtt.NewMockPublisher()
	mgr, err := dispatch.NewDispatchManager(dispatch.SimpleVehicleFilter{}, dispatch.EqualDispatcher{}, dispatch.NoopFallback{}, publisher, time.Second, nil, nil, nil, logger.NopLogger{}, nil, nil)
	if err != nil {
		t.Fatalf("manager: %v", err)
	}
	mgr.SetLogStore(store)

	vehicles := []model.Vehicle{{ID: "v1", SoC: 0.9, IsV2G: true, Available: true, MaxPower: 10}}
	sig := model.FlexibilitySignal{Type: model.SignalFCR, PowerKW: 5, Duration: time.Second, Timestamp: time.Now()}
	mgr.Dispatch(sig, vehicles)

	h := dispatchapi.NewLogHandler(store, "token")
	srv := httptest.NewServer(h)
	defer srv.Close()

	req, _ := http.NewRequest("GET", srv.URL+"?vehicle_id=v1", nil)
	req.Header.Set("Authorization", "Bearer token")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	var out []logging.LogRecord
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("expected 1 record got %d", len(out))
	}
}
