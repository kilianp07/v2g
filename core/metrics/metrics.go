package metrics

import (
	"time"

	"github.com/kilianp07/v2g/core/model"
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

// FleetDiscoveryEvent captures data about a discovery cycle.
type FleetDiscoveryEvent struct {
	Pings     int
	Responses int
	Component string
	Time      time.Time
}

// FleetDiscoveryRecorder records fleet discovery events.
type FleetDiscoveryRecorder interface {
	RecordFleetDiscovery(ev FleetDiscoveryEvent) error
}

// VehicleStateEvent is a snapshot of a vehicle.
type VehicleStateEvent struct {
	Vehicle   model.Vehicle
	Context   string
	Component string
	Time      time.Time
}

// VehicleStateRecorder records vehicle state snapshots.
type VehicleStateRecorder interface {
	RecordVehicleState(ev VehicleStateEvent) error
}

// DispatchOrderEvent represents a command sent to a vehicle.
type DispatchOrderEvent struct {
	DispatchID  string
	VehicleID   string
	Signal      model.SignalType
	PowerKW     float64
	Score       float64
	MarketPrice float64
	Time        time.Time
}

// DispatchOrderRecorder records orders sent to vehicles.
type DispatchOrderRecorder interface {
	RecordDispatchOrder(ev DispatchOrderEvent) error
}

// DispatchAckEvent captures the acknowledgment for an order.
type DispatchAckEvent struct {
	DispatchID   string
	VehicleID    string
	Signal       model.SignalType
	Acknowledged bool
	Latency      time.Duration
	Error        string
	Time         time.Time
}

// DispatchAckRecorder records ACK events.
type DispatchAckRecorder interface {
	RecordDispatchAck(ev DispatchAckEvent) error
}

// FallbackEvent records power reallocation.
type FallbackEvent struct {
	DispatchID    string
	VehicleID     string
	Signal        model.SignalType
	Reason        string
	ResidualPower float64
	Time          time.Time
}

// FallbackRecorder records fallback applications.
type FallbackRecorder interface {
	RecordFallback(ev FallbackEvent) error
}

// RTESignalEvent records a received flexibility signal.
type RTESignalEvent struct {
	Signal model.FlexibilitySignal
	Time   time.Time
}

// RTESignalRecorder records incoming RTE signals.
type RTESignalRecorder interface {
	RecordRTESignal(ev RTESignalEvent) error
}

// NopSink implements MetricsSink with no-op methods.
type NopSink struct{}

func (NopSink) RecordDispatchResult([]DispatchResult) error { return nil }

func (NopSink) RecordFleetDiscovery(FleetDiscoveryEvent) error { return nil }
func (NopSink) RecordVehicleState(VehicleStateEvent) error     { return nil }
func (NopSink) RecordDispatchOrder(DispatchOrderEvent) error   { return nil }
func (NopSink) RecordDispatchAck(DispatchAckEvent) error       { return nil }
func (NopSink) RecordFallback(FallbackEvent) error             { return nil }
func (NopSink) RecordRTESignal(RTESignalEvent) error           { return nil }

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
