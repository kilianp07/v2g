package metrics_test

import (
	"encoding/json"
	"testing"

	"gopkg.in/yaml.v3"

	metrics "github.com/kilianp07/v2g/core/metrics"
	_ "github.com/kilianp07/v2g/infra/metrics"
)

// Test decoding from YAML with multiple sinks.
func TestMetricsConfigDecodeYAML(t *testing.T) {
	data := `sinks:
  - type: nop
  - type: nop
`
	var cfg metrics.Config
	if err := yaml.Unmarshal([]byte(data), &cfg); err != nil {
		t.Fatalf("yaml unmarshal: %v", err)
	}
	s, err := metrics.NewMetricsSink(cfg.Sinks)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, ok := s.(*metrics.MultiSink); !ok {
		t.Fatalf("expected MultiSink")
	}
}

// Test decoding from JSON with invalid sink type.
func TestMetricsConfigDecodeJSON_Invalid(t *testing.T) {
	data := `{"sinks":[{"type":"missing"}]}`
	var cfg metrics.Config
	if err := json.Unmarshal([]byte(data), &cfg); err != nil {
		t.Fatalf("json unmarshal: %v", err)
	}
	if _, err := metrics.NewMetricsSink(cfg.Sinks); err == nil {
		t.Fatalf("expected error for unknown type")
	}
}
