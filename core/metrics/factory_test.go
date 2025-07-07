package metrics_test

import (
	"testing"

	"github.com/kilianp07/v2g/core/factory"
	metrics "github.com/kilianp07/v2g/core/metrics"
	_ "github.com/kilianp07/v2g/infra/metrics"
)

/*
TestMetricsFactory_Builtins verifies registration via infra/metrics/factory.go.

	Cases:
	- instantiate builtin nop sink
	- unknown type returns error
*/
func TestMetricsFactory_Builtins(t *testing.T) {
	s, err := metrics.NewMetricsSink([]factory.ModuleConfig{{Type: "nop"}})
	if err != nil {
		t.Fatalf("create nop: %v", err)
	}
	if s == nil {
		t.Fatal("expected sink instance")
	}
	if _, err := metrics.NewMetricsSink([]factory.ModuleConfig{{Type: "missing"}}); err == nil {
		t.Fatal("expected error for unknown type")
	}
}

/*
TestNewMetricsSink_Multi validates NewMetricsSink behavior with zero, one, and multiple configs.
Cases:
  - no config -> NopSink
  - two configs -> MultiSink with two sub-sinks
*/
func TestNewMetricsSink_Multi(t *testing.T) {
	// No config defaults to NopSink
	s, err := metrics.NewMetricsSink(nil)
	if err != nil {
		t.Fatalf("create nop default: %v", err)
	}
	if _, ok := s.(metrics.NopSink); !ok {
		t.Fatalf("expected NopSink, got %T", s)
	}

	// Multiple configs returns MultiSink
	cfgs := []factory.ModuleConfig{{Type: "nop"}, {Type: "nop"}}
	s, err = metrics.NewMetricsSink(cfgs)
	if err != nil {
		t.Fatalf("create multi: %v", err)
	}
	m, ok := s.(*metrics.MultiSink)
	if !ok {
		t.Fatalf("expected MultiSink, got %T", s)
	}
	if len(m.Sinks) != 2 {
		t.Fatalf("expected 2 sinks, got %d", len(m.Sinks))
	}
}
