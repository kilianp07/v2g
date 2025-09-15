package config

import "testing"

func TestTelemetryConfigDefaults(t *testing.T) {
	cfg := TelemetryConfig{}
	if cfg.Interval() != 10 {
		t.Fatalf("expected default interval 10, got %d", cfg.Interval())
	}
	if cfg.Timeout() != 3 {
		t.Fatalf("expected default timeout 3, got %d", cfg.Timeout())
	}
}

func TestTelemetryConfigValues(t *testing.T) {
	cfg := TelemetryConfig{IntervalSeconds: 5, TimeoutSeconds: 2}
	if cfg.Interval() != 5 {
		t.Fatalf("expected interval 5, got %d", cfg.Interval())
	}
	if cfg.Timeout() != 2 {
		t.Fatalf("expected timeout 2, got %d", cfg.Timeout())
	}
}
