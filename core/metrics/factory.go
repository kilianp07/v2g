package metrics

import "github.com/kilianp07/v2g/core/factory"

var sinkRegistry = factory.NewRegistry[MetricsSink]()

// RegisterMetricsSink adds a metrics sink factory identified by name.
func RegisterMetricsSink(name string, f factory.Factory[MetricsSink]) error {
	return sinkRegistry.Register(name, f)
}

// NewMetricsSink creates a MetricsSink from the provided configuration.
func NewMetricsSink(cfgs []factory.ModuleConfig) (MetricsSink, error) {
	if len(cfgs) == 0 {
		return NopSink{}, nil
	}
	if len(cfgs) == 1 {
		return sinkRegistry.Create(cfgs[0])
	}
	sinks := make([]MetricsSink, len(cfgs))
	for i, c := range cfgs {
		s, err := sinkRegistry.Create(c)
		if err != nil {
			return nil, err
		}
		sinks[i] = s
	}
	return NewMultiSink(sinks...), nil
}
