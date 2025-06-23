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

// NewPromSink registers dispatch metrics on the provided Prometheus registerer.
// If reg is nil, the default registerer is used. If the collectors are already
// registered, the existing ones are reused.
func NewPromSink(reg prometheus.Registerer) (*PromSink, error) {
	if reg == nil {
		reg = prometheus.DefaultRegisterer
	}
	events := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "dispatch_events_total",
		Help: "Total number of dispatch events",
	}, []string{"vehicle_id", "signal_type", "acknowledged"})
	latency := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "dispatch_latency_seconds",
		Help:    "Time between command send and acknowledgment",
		Buckets: prometheus.DefBuckets,
	}, []string{"vehicle_id", "signal_type", "acknowledged"})

	if err := reg.Register(events); err != nil {
		if are, ok := err.(prometheus.AlreadyRegisteredError); ok {
			events = are.ExistingCollector.(*prometheus.CounterVec)
		} else {
			return nil, err
		}
	}
	if err := reg.Register(latency); err != nil {
		if are, ok := err.(prometheus.AlreadyRegisteredError); ok {
			latency = are.ExistingCollector.(*prometheus.HistogramVec)
		} else {
			return nil, err
		}
	}

	return &PromSink{events: events, latency: latency}, nil
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
