package dispatch

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestMetricsRegistration(t *testing.T) {
	ResetMetrics(nil)
	t.Cleanup(func() { ResetMetrics(nil) })
	reg := prometheus.NewRegistry()
	MustRegisterMetrics(reg)
	// touch metrics so they are exported
	vehiclesDispatched.WithLabelValues("FCR").Inc()
	dispatchLatency.WithLabelValues("FCR").Observe(0.1)
	ackRate.WithLabelValues("FCR").Set(1)
	mqttSuccess.Inc()
	mqttFailure.Inc()
	mfs, err := reg.Gather()
	if err != nil {
		t.Fatalf("gather: %v", err)
	}
	names := map[string]bool{}
	for _, mf := range mfs {
		names[*mf.Name] = true
	}
	expected := []string{
		"dispatch_execution_latency_seconds",
		"vehicles_dispatched_total",
		"ack_rate",
		"mqtt_publish_success_total",
		"mqtt_publish_failure_total",
	}
	for _, n := range expected {
		if !names[n] {
			t.Errorf("metric %s not registered", n)
		}
	}
}
