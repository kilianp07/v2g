package dispatch

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kilianp07/v2g/core/dispatch/logging"
	"github.com/kilianp07/v2g/core/model"
)

type memStore struct{ recs []logging.LogRecord }

func (m *memStore) Append(ctx context.Context, r logging.LogRecord) error {
	m.recs = append(m.recs, r)
	return nil
}

func (m *memStore) Query(ctx context.Context, q logging.LogQuery) ([]logging.LogRecord, error) {
	var res []logging.LogRecord
	for _, r := range m.recs {
		if q.VehicleID != "" {
			found := false
			for _, id := range r.VehiclesSelected {
				if id == q.VehicleID {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		res = append(res, r)
	}
	return res, nil
}

func (m *memStore) Close() error { return nil }

func TestLogHandler_AuthAndFilters(t *testing.T) {
	store := &memStore{}
	if err := store.Append(context.Background(), logging.LogRecord{
		Timestamp:        time.Now(),
		Signal:           model.FlexibilitySignal{Type: model.SignalFCR},
		TargetPower:      1,
		VehiclesSelected: []string{"v1"},
		Response:         logging.Result{},
	}); err != nil {
		t.Fatalf("append: %v", err)
	}
	h := NewLogHandler(store, "tok")

	req := httptest.NewRequest("GET", "/api/dispatch/logs?vehicle_id=v1", nil)
	req.Header.Set("Authorization", "Bearer tok")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d", rr.Code)
	}
	var out []logging.LogRecord
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("expected 1 record")
	}
	// unauthorized
	req = httptest.NewRequest("GET", "/api/dispatch/logs", nil)
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", rr.Code)
	}
}
