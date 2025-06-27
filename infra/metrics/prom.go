package metrics

import (
	"strconv"

	coremetrics "github.com/kilianp07/v2g/core/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

// PromSink records dispatch events in Prometheus metrics.
type PromSink struct {
	events  *prometheus.CounterVec
	latency *prometheus.HistogramVec
	fleet   prometheus.Gauge
}

// NewPromSink registers dispatch metrics on the default Prometheus registerer.
// The Prometheus server should be started separately using cfg.PrometheusPort.
func NewPromSink(cfg coremetrics.Config) (coremetrics.MetricsSink, error) {
	return NewPromSinkWithRegistry(cfg, prometheus.DefaultRegisterer)
}

// NewPromSinkWithRegistry registers metrics on the provided registerer.
// A nil registerer defaults to the global Prometheus registerer.
func NewPromSinkWithRegistry(cfg coremetrics.Config, reg prometheus.Registerer) (coremetrics.MetricsSink, error) {
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
	fleet := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "fleet_discovery_vehicles_total",
		Help: "Number of vehicles discovered during fleet discovery",
	})

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
	if err := reg.Register(fleet); err != nil {
		if are, ok := err.(prometheus.AlreadyRegisteredError); ok {
			fleet = are.ExistingCollector.(prometheus.Gauge)
		} else {
			return nil, err
		}
	}

	return &PromSink{events: events, latency: latency, fleet: fleet}, nil
}

// RecordDispatchResult increments the counter for each dispatch result.
func (s *PromSink) RecordDispatchResult(res []coremetrics.DispatchResult) error {
	for _, r := range res {
		s.events.WithLabelValues(r.VehicleID, r.Signal.Type.String(), strconv.FormatBool(r.Acknowledged)).Inc()
	}
	return nil
}

// RecordDispatchLatency records the dispatch latency histogram.
func (s *PromSink) RecordDispatchLatency(recs []coremetrics.DispatchLatency) error {
	for _, r := range recs {
		s.latency.WithLabelValues(r.VehicleID, r.Signal.String(), strconv.FormatBool(r.Acknowledged)).Observe(r.Latency.Seconds())
	}
	return nil
}

// RecordFleetSize sets the gauge to the number of discovered vehicles.
func (s *PromSink) RecordFleetSize(size int) error {
	if s.fleet != nil {
		s.fleet.Set(float64(size))
	}
	return nil
}
