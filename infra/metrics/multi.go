package metrics

import coremetrics "github.com/kilianp07/v2g/core/metrics"

// MultiSink fanouts dispatch results to multiple sinks.
type MultiSink struct {
	Sinks []coremetrics.MetricsSink
}

// NewMultiSink creates a MultiSink with the provided sinks.
func NewMultiSink(sinks ...coremetrics.MetricsSink) *MultiSink {
	return &MultiSink{Sinks: sinks}
}

// RecordDispatchResult forwards the record to all sinks, returning the first error encountered.
func (m *MultiSink) RecordDispatchResult(res []coremetrics.DispatchResult) error {
	for _, s := range m.Sinks {
		if err := s.RecordDispatchResult(res); err != nil {
			return err
		}
	}
	return nil
}

// RecordFleetDiscovery forwards discovery events.
func (m *MultiSink) RecordFleetDiscovery(ev coremetrics.FleetDiscoveryEvent) error {
	for _, s := range m.Sinks {
		if rec, ok := s.(coremetrics.FleetDiscoveryRecorder); ok {
			if err := rec.RecordFleetDiscovery(ev); err != nil {
				return err
			}
		}
	}
	return nil
}

// RecordVehicleState forwards vehicle snapshots.
func (m *MultiSink) RecordVehicleState(ev coremetrics.VehicleStateEvent) error {
	for _, s := range m.Sinks {
		if rec, ok := s.(coremetrics.VehicleStateRecorder); ok {
			if err := rec.RecordVehicleState(ev); err != nil {
				return err
			}
		}
	}
	return nil
}

// RecordDispatchOrder forwards order events.
func (m *MultiSink) RecordDispatchOrder(ev coremetrics.DispatchOrderEvent) error {
	for _, s := range m.Sinks {
		if rec, ok := s.(coremetrics.DispatchOrderRecorder); ok {
			if err := rec.RecordDispatchOrder(ev); err != nil {
				return err
			}
		}
	}
	return nil
}

// RecordDispatchAck forwards ack events.
func (m *MultiSink) RecordDispatchAck(ev coremetrics.DispatchAckEvent) error {
	for _, s := range m.Sinks {
		if rec, ok := s.(coremetrics.DispatchAckRecorder); ok {
			if err := rec.RecordDispatchAck(ev); err != nil {
				return err
			}
		}
	}
	return nil
}

// RecordFallback forwards fallback events.
func (m *MultiSink) RecordFallback(ev coremetrics.FallbackEvent) error {
	for _, s := range m.Sinks {
		if rec, ok := s.(coremetrics.FallbackRecorder); ok {
			if err := rec.RecordFallback(ev); err != nil {
				return err
			}
		}
	}
	return nil
}

// RecordRTESignal forwards RTE signal events.
func (m *MultiSink) RecordRTESignal(ev coremetrics.RTESignalEvent) error {
	for _, s := range m.Sinks {
		if rec, ok := s.(coremetrics.RTESignalRecorder); ok {
			if err := rec.RecordRTESignal(ev); err != nil {
				return err
			}
		}
	}
	return nil
}

// RecordDispatchLatency forwards latency metrics when supported by the sink.
func (m *MultiSink) RecordDispatchLatency(lat []coremetrics.DispatchLatency) error {
	for _, s := range m.Sinks {
		if lr, ok := s.(coremetrics.LatencyRecorder); ok {
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
		if fr, ok := s.(coremetrics.FleetSizeRecorder); ok {
			if err := fr.RecordFleetSize(size); err != nil {
				return err
			}
		}
	}
	return nil
}
