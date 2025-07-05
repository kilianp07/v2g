package dispatch

import (
	"testing"

	"github.com/kilianp07/v2g/core/model"
)

func TestDispatchManager_SetLPFirstNil(t *testing.T) {
	m := &DispatchManager{lpFirst: map[model.SignalType]bool{model.SignalFCR: true}}
	m.SetLPFirst(nil)
	if !m.lpFirst[model.SignalFCR] {
		t.Fatalf("existing lpFirst entry changed on nil input")
	}
}

func TestDispatchManager_SetLPFirstAssign(t *testing.T) {
	m := &DispatchManager{}
	cfg := map[model.SignalType]bool{model.SignalFCR: true, model.SignalNEBEF: false}
	m.SetLPFirst(cfg)
	if len(m.lpFirst) != len(cfg) {
		t.Fatalf("expected %d entries got %d", len(cfg), len(m.lpFirst))
	}
	for k, v := range cfg {
		if m.lpFirst[k] != v {
			t.Fatalf("value mismatch for %v", k)
		}
	}
}
