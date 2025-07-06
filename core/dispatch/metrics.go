package dispatch

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	dispatchLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "dispatch_execution_latency_seconds",
			Help:    "Latency of dispatch orders from publish to acknowledgment",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"signal_type"},
	)

	vehiclesDispatched = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "vehicles_dispatched_total",
			Help: "Number of vehicles dispatched",
		},
		[]string{"signal_type"},
	)

	ackRate = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ack_rate",
			Help: "Acknowledgment rate for dispatch orders",
		},
		[]string{"signal_type"},
	)

	mqttSuccess = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "mqtt_publish_success_total",
			Help: "Number of successful MQTT publish operations",
		},
	)

	mqttFailure = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "mqtt_publish_failure_total",
			Help: "Number of failed MQTT publish operations",
		},
	)
)

func init() { MustRegisterMetrics(nil) }

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
	dispatchLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "dispatch_execution_latency_seconds",
			Help:    "Latency of dispatch orders from publish to acknowledgment",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"signal_type"},
	)
	vehiclesDispatched = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "vehicles_dispatched_total",
			Help: "Number of vehicles dispatched",
		},
		[]string{"signal_type"},
	)
	ackRate = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ack_rate",
			Help: "Acknowledgment rate for dispatch orders",
		},
		[]string{"signal_type"},
	)
	mqttSuccess = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "mqtt_publish_success_total",
			Help: "Number of successful MQTT publish operations",
		},
	)
	mqttFailure = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "mqtt_publish_failure_total",
			Help: "Number of failed MQTT publish operations",
		},
	)
	if reg != nil {
		MustRegisterMetrics(reg)
	}
}
