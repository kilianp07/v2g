package metrics

// Package metrics defines interfaces and implementations for collecting
// dispatch metrics. Sinks like PromSink and InfluxSink record events such
// as vehicle assignments or acknowledgments and can be combined with
// NewMultiSink. The factory helpers return a MultiSink automatically when
// multiple sinks are configured. Helper functions expose Prometheus metrics
// and collect events from the internal event bus.
