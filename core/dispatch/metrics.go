package dispatch

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	dispatchLatency    *prometheus.HistogramVec
	vehiclesDispatched *prometheus.CounterVec
	ackRate            *prometheus.GaugeVec
	mqttSuccess        prometheus.Counter
	mqttFailure        prometheus.Counter
)

// newCollectors creates new metric collectors.
func newCollectors() (*prometheus.HistogramVec, *prometheus.CounterVec, *prometheus.GaugeVec, prometheus.Counter, prometheus.Counter) {
	lat := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "dispatch_execution_latency_seconds",
			Help:    "Latency of dispatch orders from publish to acknowledgment",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"signal_type"},
	)
	veh := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "vehicles_dispatched_total",
			Help: "Number of vehicles dispatched",
		},
		[]string{"signal_type"},
	)
	ack := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ack_rate",
			Help: "Acknowledgment rate for dispatch orders",
		},
		[]string{"signal_type"},
	)
	suc := prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "mqtt_publish_success_total",
			Help: "Number of successful MQTT publish operations",
		},
	)
	fail := prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "mqtt_publish_failure_total",
			Help: "Number of failed MQTT publish operations",
		},
	)
	return lat, veh, ack, suc, fail
}

func init() {
	dispatchLatency, vehiclesDispatched, ackRate, mqttSuccess, mqttFailure = newCollectors()
	MustRegisterMetrics(nil)
}

// MustRegisterMetrics registers dispatch metrics on the provided registry.
// If reg is nil, prometheus.DefaultRegisterer is used.
func MustRegisterMetrics(reg prometheus.Registerer) {
	if reg == nil {
		reg = prometheus.DefaultRegisterer
	}
	reg.MustRegister(dispatchLatency, vehiclesDispatched, ackRate, mqttSuccess, mqttFailure)
}

// ResetMetrics reinitializes metrics collectors for testing purposes and
// registers them on the provided registry if not nil.
func ResetMetrics(reg prometheus.Registerer) {
	dispatchLatency, vehiclesDispatched, ackRate, mqttSuccess, mqttFailure = newCollectors()
	if reg != nil {
		MustRegisterMetrics(reg)
	}
}
