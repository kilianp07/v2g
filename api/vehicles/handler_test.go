package vehicles

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kilianp07/v2g/core/prediction"
	vehiclestatus "github.com/kilianp07/v2g/core/vehiclestatus"
)

func TestStatusHandler_Basic(t *testing.T) {
	store := vehiclestatus.NewMemoryStore()
	store.Set(vehiclestatus.Status{VehicleID: "v1", FleetID: "f1", CurrentStatus: "idle"})
	h := NewStatusHandler(store, nil)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/vehicles/status", nil)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d", rr.Code)
	}
	var out []vehiclestatus.Status
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(out) != 1 || out[0].VehicleID != "v1" {
		t.Fatalf("unexpected output %#v", out)
	}
}

func TestStatusHandler_Filter(t *testing.T) {
	store := vehiclestatus.NewMemoryStore()
	store.Set(vehiclestatus.Status{VehicleID: "v1", FleetID: "f1", Site: "s1"})
	store.Set(vehiclestatus.Status{VehicleID: "v2", FleetID: "f2", Site: "s2"})
	h := NewStatusHandler(store, nil)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/vehicles/status?fleet_id=f1&site=s1", nil)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d", rr.Code)
	}
	var out []vehiclestatus.Status
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(out) != 1 || out[0].VehicleID != "v1" {
		t.Fatalf("unexpected filter result %#v", out)
	}
}

func TestStatusHandler_FilterCluster(t *testing.T) {
	store := vehiclestatus.NewMemoryStore()
	store.Set(vehiclestatus.Status{VehicleID: "v1", Cluster: "c1"})
	store.Set(vehiclestatus.Status{VehicleID: "v2", Cluster: "c2"})
	h := NewStatusHandler(store, nil)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/vehicles/status?cluster=c2", nil)
	h.ServeHTTP(rr, req)
	var out []vehiclestatus.Status
	_ = json.Unmarshal(rr.Body.Bytes(), &out)
	if len(out) != 1 || out[0].VehicleID != "v2" {
		t.Fatalf("cluster filter bad %#v", out)
	}
}

func TestStatusHandler_Empty(t *testing.T) {
	store := vehiclestatus.NewMemoryStore()
	h := NewStatusHandler(store, nil)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/vehicles/status", nil)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d", rr.Code)
	}
	if rr.Body.String() != "[]\n" {
		t.Fatalf("expected empty array got %s", rr.Body.String())
	}
}

func TestStatusHandler_Prediction(t *testing.T) {
	store := vehiclestatus.NewMemoryStore()
	store.Set(vehiclestatus.Status{VehicleID: "v1"})
	pred := &prediction.MockPredictionEngine{SoCForecasts: map[string][]float64{"v1": {0.8, 0.7}}}
	h := NewStatusHandler(store, pred)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/vehicles/status", nil)
	h.ServeHTTP(rr, req)
	var out []vehiclestatus.Status
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out[0].ForecastedSoC["t+15m"] != 0.7 {
		t.Fatalf("prediction not applied")
	}
	if out[0].ForecastedPluginWindow.End.IsZero() {
		t.Fatalf("plugin window missing")
	}
}

func TestStatusHandler_NoForecast(t *testing.T) {
	store := vehiclestatus.NewMemoryStore()
	store.Set(vehiclestatus.Status{VehicleID: "v1"})
	pred := &prediction.MockPredictionEngine{}
	h := NewStatusHandler(store, pred)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/vehicles/status", nil)
	h.ServeHTTP(rr, req)
	var out []vehiclestatus.Status
	_ = json.Unmarshal(rr.Body.Bytes(), &out)
	if len(out) != 1 {
		t.Fatalf("expected 1")
	}
}
