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

// RecordFleetDiscovery forwards discovery events.
func (m *MultiSink) RecordFleetDiscovery(ev FleetDiscoveryEvent) error {
	for _, s := range m.Sinks {
		if rec, ok := s.(FleetDiscoveryRecorder); ok {
			if err := rec.RecordFleetDiscovery(ev); err != nil {
				return err
			}
		}
	}
	return nil
}

// RecordVehicleState forwards vehicle snapshots.
func (m *MultiSink) RecordVehicleState(ev VehicleStateEvent) error {
	for _, s := range m.Sinks {
		if rec, ok := s.(VehicleStateRecorder); ok {
			if err := rec.RecordVehicleState(ev); err != nil {
				return err
			}
		}
	}
	return nil
}

// RecordDispatchOrder forwards order events.
func (m *MultiSink) RecordDispatchOrder(ev DispatchOrderEvent) error {
	for _, s := range m.Sinks {
		if rec, ok := s.(DispatchOrderRecorder); ok {
			if err := rec.RecordDispatchOrder(ev); err != nil {
				return err
			}
		}
	}
	return nil
}

// RecordDispatchAck forwards ack events.
func (m *MultiSink) RecordDispatchAck(ev DispatchAckEvent) error {
	for _, s := range m.Sinks {
		if rec, ok := s.(DispatchAckRecorder); ok {
			if err := rec.RecordDispatchAck(ev); err != nil {
				return err
			}
		}
	}
	return nil
}

// RecordFallback forwards fallback events.
func (m *MultiSink) RecordFallback(ev FallbackEvent) error {
	for _, s := range m.Sinks {
		if rec, ok := s.(FallbackRecorder); ok {
			if err := rec.RecordFallback(ev); err != nil {
				return err
			}
		}
	}
	return nil
}

// RecordRTESignal forwards RTE signal events.
func (m *MultiSink) RecordRTESignal(ev RTESignalEvent) error {
	for _, s := range m.Sinks {
		if rec, ok := s.(RTESignalRecorder); ok {
			if err := rec.RecordRTESignal(ev); err != nil {
				return err
			}
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

// RecordFleetSize forwards fleet size metrics when supported by the sink.
func (m *MultiSink) RecordFleetSize(size int) error {
	for _, s := range m.Sinks {
		if fr, ok := s.(FleetSizeRecorder); ok {
			if err := fr.RecordFleetSize(size); err != nil {
				return err
			}
		}
	}
	return nil
}
