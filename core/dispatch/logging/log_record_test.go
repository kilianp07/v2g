package logging

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/kilianp07/v2g/core/model"
)

func TestLogRecord_JSON(t *testing.T) {
	rec := LogRecord{
		Timestamp:        time.Unix(0, 0),
		Signal:           model.FlexibilitySignal{Type: model.SignalFCR},
		TargetPower:      10,
		VehiclesSelected: []string{"v1"},
		Response:         Result{Assignments: map[string]float64{"v1": 10}},
	}
	data, err := json.Marshal(rec)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	keys := []string{"timestamp", "signal", "target_power", "vehicles_selected", "response"}
	for _, k := range keys {
		if _, ok := m[k]; !ok {
			t.Errorf("missing key %s", k)
		}
	}
}
