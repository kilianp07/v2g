package metrics

// MultiSink fanouts dispatch results to multiple sinks.
type MultiSink struct {
	Sinks []MetricsSink
}

// NewMultiSink creates a MultiSink with the provided sinks.
func NewMultiSink(sinks ...MetricsSink) *MultiSink {
	return &MultiSink{Sinks: sinks}
}

// RecordDispatchResult forwards the record to all sinks, returning the first error encountered.
func (m *MultiSink) RecordDispatchResult(res []DispatchResult) error {
	for _, s := range m.Sinks {
		if err := s.RecordDispatchResult(res); err != nil {
			return err
		}
	}
	return nil
}

// RecordDispatchLatency forwards latency metrics when supported by the sink.
func (m *MultiSink) RecordDispatchLatency(lat []DispatchLatency) error {
	for _, s := range m.Sinks {
		if lr, ok := s.(LatencyRecorder); ok {
			if err := lr.RecordDispatchLatency(lat); err != nil {
				return err
			}
		}
	}
	return nil
}
