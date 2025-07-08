package plugins

import (
	"github.com/kilianp07/v2g/core/dispatch"
	dispatchlog "github.com/kilianp07/v2g/core/dispatch/logging"
	coremetrics "github.com/kilianp07/v2g/core/metrics"
	"github.com/kilianp07/v2g/core/prediction"
)

// DispatcherFactory builds a dispatcher from a raw configuration map.
type DispatcherFactory func(name string, conf map[string]any) (dispatch.Dispatcher, error)

// FallbackFactory builds a fallback strategy from a raw configuration map.
type FallbackFactory func(name string, conf map[string]any) (dispatch.FallbackStrategy, error)

// TunerFactory binds a tuner to the dispatcher using its configuration.
type TunerFactory func(name string, conf map[string]any, d dispatch.Dispatcher) (dispatch.LearningTuner, error)

// MetricsFactory builds a metrics exporter from raw config.
type MetricsFactory func(name string, conf map[string]any) (coremetrics.MetricsSink, error)

// LogStoreFactory builds a dispatch log store from raw config.
type LogStoreFactory func(name string, conf map[string]any) (dispatchlog.LogStore, error)

// PredictionFactory builds a prediction engine from raw config.
type PredictionFactory func(name string, conf map[string]any) (prediction.PredictionEngine, error)

var (
	Dispatchers      = map[string]DispatcherFactory{}
	Fallbacks        = map[string]FallbackFactory{}
	Tuners           = map[string]TunerFactory{}
	MetricsExporters = map[string]MetricsFactory{}
	LogStores        = map[string]LogStoreFactory{}
	Predictions      = map[string]PredictionFactory{}
)

func RegisterDispatcher(name string, f DispatcherFactory) { Dispatchers[name] = f }
func RegisterFallback(name string, f FallbackFactory)     { Fallbacks[name] = f }
func RegisterTuner(name string, f TunerFactory)           { Tuners[name] = f }
func RegisterMetrics(name string, f MetricsFactory)       { MetricsExporters[name] = f }
func RegisterLogStore(name string, f LogStoreFactory)     { LogStores[name] = f }
func RegisterPrediction(name string, f PredictionFactory) { Predictions[name] = f }
