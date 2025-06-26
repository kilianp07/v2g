package metrics

import (
	"time"

	"github.com/kilianp07/v2g/model"
)

// DispatchResult represents a per-vehicle dispatch event to be recorded.
type DispatchResult struct {
	Signal       model.FlexibilitySignal
	VehicleID    string
	PowerKW      float64
	Score        float64
	MarketPrice  float64
	Acknowledged bool
	DispatchTime time.Time
}

// MetricsSink records dispatch results for observability purposes.
type MetricsSink interface {
	RecordDispatchResult(results []DispatchResult) error
}

// NopSink implements MetricsSink with no-op methods.
type NopSink struct{}

func (NopSink) RecordDispatchResult([]DispatchResult) error { return nil }

// DispatchLatency represents the time to receive an acknowledgment for an order.
type DispatchLatency struct {
	VehicleID    string
	Signal       model.SignalType
	Acknowledged bool
	Latency      time.Duration
}

// LatencyRecorder is implemented by sinks able to record dispatch latency.
type LatencyRecorder interface {
	RecordDispatchLatency(latencies []DispatchLatency) error
}

// Ensure NopSink implements LatencyRecorder.
func (NopSink) RecordDispatchLatency([]DispatchLatency) error { return nil }

// FleetSizeRecorder records the number of vehicles discovered during fleet discovery.
type FleetSizeRecorder interface {
	RecordFleetSize(size int) error
}

// Ensure NopSink implements FleetSizeRecorder.
func (NopSink) RecordFleetSize(int) error { return nil }
