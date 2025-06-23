package metrics

import (
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
)

// PromSink records dispatch events in Prometheus metrics.
type PromSink struct {
	events  *prometheus.CounterVec
	latency *prometheus.HistogramVec
}

// NewPromSink registers a dispatch_events_total counter on the provided
// Prometheus Registerer. If reg is nil, the default registry is used.
func NewPromSink(reg prometheus.Registerer) *PromSink {
	if reg == nil {
		reg = prometheus.DefaultRegisterer
	}
	s := &PromSink{
		events: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "dispatch_events_total",
			Help: "Total number of dispatch events",
		}, []string{"vehicle_id", "signal_type", "acknowledged"}),
		latency: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "dispatch_latency_seconds",
			Help:    "Time between command send and acknowledgment",
			Buckets: prometheus.DefBuckets,
		}, []string{"vehicle_id", "signal_type", "acknowledged"}),
	}
	reg.MustRegister(s.events, s.latency)
	return s
}

// RecordDispatchResult increments the counter for each dispatch result.
func (s *PromSink) RecordDispatchResult(res []DispatchResult) error {
	for _, r := range res {
		s.events.WithLabelValues(r.VehicleID, r.Signal.Type.String(), strconv.FormatBool(r.Acknowledged)).Inc()
	}
	return nil
}

// RecordDispatchLatency records the dispatch latency histogram.
func (s *PromSink) RecordDispatchLatency(recs []DispatchLatency) error {
	for _, r := range recs {
		s.latency.WithLabelValues(r.VehicleID, r.Signal.String(), strconv.FormatBool(r.Acknowledged)).Observe(r.Latency.Seconds())
	}
	return nil
}
