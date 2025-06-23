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
