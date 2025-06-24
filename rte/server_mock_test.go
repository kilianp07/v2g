package rte

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kilianp07/v2g/config"
	"github.com/kilianp07/v2g/dispatch"
	"github.com/kilianp07/v2g/model"
	"github.com/prometheus/client_golang/prometheus"
)

type dmMock struct{ received int }

func (d *dmMock) Dispatch(model.FlexibilitySignal, []model.Vehicle) dispatch.DispatchResult {
	d.received++
	return dispatch.DispatchResult{}
}

func TestRTEServerMock(t *testing.T) {
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
	prometheus.DefaultGatherer = prometheus.DefaultRegisterer.(prometheus.Gatherer)
	dm := &dmMock{}
	cfg := config.RTEMockConfig{Address: ""}
	srv := NewRTEServerMock(cfg, dm, nil)
	handler := srv.routes()
	ts := httptest.NewServer(handler)
	defer ts.Close()

	sig := Signal{
		SignalType: "FCR",
		StartTime:  time.Now(),
		EndTime:    time.Now().Add(15 * time.Minute),
		Power:      50,
	}
	data, _ := json.Marshal(sig)
	resp, err := http.Post(ts.URL+"/rte/signal", "application/json", bytes.NewReader(data))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: %d", resp.StatusCode)
	}
	if dm.received != 1 {
		t.Fatalf("dispatch not called")
	}
}

func TestNewConnectorSelectsMock(t *testing.T) {
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
	prometheus.DefaultGatherer = prometheus.DefaultRegisterer.(prometheus.Gatherer)
	dm := &dmMock{}
	cfg := config.RTEConfig{Mode: "mock"}
	c := NewConnector(cfg, dm, nil)
	if _, ok := c.(*RTEServerMock); !ok {
		t.Fatalf("expected mock server")
	}
}
