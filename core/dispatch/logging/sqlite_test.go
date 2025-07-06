package logging

import (
	"context"
	"testing"
	"time"

	"github.com/kilianp07/v2g/core/model"
)

func TestSQLiteStore_PersistQuery(t *testing.T) {
	store, err := NewSQLiteStore("file:test.db?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = store.Close() }()
	rec := LogRecord{
		Timestamp:        time.Now(),
		Signal:           model.FlexibilitySignal{Type: model.SignalFCR},
		TargetPower:      5,
		VehiclesSelected: []string{"v1"},
		Response:         Result{Assignments: map[string]float64{"v1": 5}},
	}
	if err := store.Append(context.Background(), rec); err != nil {
		t.Fatalf("append: %v", err)
	}
	out, err := store.Query(context.Background(), LogQuery{VehicleID: "v1"})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("expected 1 record, got %d", len(out))
	}
}
